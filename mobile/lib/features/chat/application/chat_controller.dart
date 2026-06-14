import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_exception.dart';
import '../../auth/application/auth_controller.dart';
import '../data/chat_repository.dart';
import '../domain/chat_models.dart';

/// Immutable view state for the chat screen. [messages] is chronological
/// (oldest first). [nextCursor] paginates older history; null means the start
/// of the conversation has been reached (or is not yet known).
class ChatState {
  const ChatState({
    this.messages = const [],
    this.sending = false,
    this.loadingOlder = false,
    this.nextCursor,
    this.feedback = const {},
  });

  final List<ChatMessage> messages;
  final bool sending;
  final bool loadingOlder;
  final String? nextCursor;

  /// Thumbs feedback the user gave this session, keyed by message id
  /// (`up`/`down`). Client-only: the buttons reflect taps without a reload
  /// round-trip; the backend persists the row.
  final Map<String, String> feedback;

  bool get hasMore => nextCursor != null;

  ChatState copyWith({
    List<ChatMessage>? messages,
    bool? sending,
    bool? loadingOlder,
    String? nextCursor,
    bool clearCursor = false,
    Map<String, String>? feedback,
  }) {
    return ChatState(
      messages: messages ?? this.messages,
      sending: sending ?? this.sending,
      loadingOlder: loadingOlder ?? this.loadingOlder,
      nextCursor: clearCursor ? null : (nextCursor ?? this.nextCursor),
      feedback: feedback ?? this.feedback,
    );
  }
}

/// Drives the chat screen: loads history (cache-first), streams the assistant
/// reply over SSE, and reconciles with the server after each turn.
class ChatController extends AsyncNotifier<ChatState> {
  ChatRepository get _repo => ref.read(chatRepositoryProvider);

  CancelToken? _cancelToken;
  int _seq = 0;

  @override
  Future<ChatState> build() async {
    // Clear cached messages when the session ends so a different user on the
    // same device never sees the previous user's chat (ownership invariant).
    ref.listen<AuthStatus>(authStatusProvider, (_, next) {
      if (next == AuthStatus.unauthenticated) {
        unawaited(_repo.clearCache());
      }
    });

    final cached = await _repo.loadCached();
    if (cached.isEmpty) {
      // Nothing cached: load from the server (loading spinner shows meanwhile).
      try {
        final page = await _repo.fetchConversation();
        await _repo.cache(page.messages);
        return ChatState(messages: page.messages, nextCursor: page.nextCursor);
      } on AppException {
        return const ChatState();
      }
    }
    // Have a cache: show it immediately, reconcile with the server in the
    // background.
    unawaited(_refreshFromServer());
    return ChatState(messages: cached);
  }

  /// Sends a message of [text] and/or already-uploaded photos (referenced by
  /// [imageStorageKeys]) and streams the assistant reply.
  Future<void> sendMessage(
    String text, {
    List<String> imageStorageKeys = const [],
    String? audioStorageKey,
    int audioDurationMs = 0,
  }) async {
    final trimmed = text.trim();
    final current = state.valueOrNull;
    if ((trimmed.isEmpty &&
            imageStorageKeys.isEmpty &&
            audioStorageKey == null) ||
        current == null ||
        current.sending) {
      return;
    }

    final now = DateTime.now().toUtc();
    final localUserId = _localId('user');
    final localAssistantId = _localId('assistant');
    final userMsg = ChatMessage(
      id: localUserId,
      role: MessageRole.user,
      status: MessageStatus.complete,
      createdAt: now,
      content: [
        if (trimmed.isNotEmpty) ContentBlock(type: 'text', text: trimmed),
        for (final key in imageStorageKeys)
          ContentBlock(type: 'image', storageKey: key),
        if (audioStorageKey != null)
          ContentBlock(
            type: 'audio',
            storageKey: audioStorageKey,
            durationMs: audioDurationMs,
          ),
      ],
    );
    final assistantMsg = ChatMessage(
      id: localAssistantId,
      role: MessageRole.assistant,
      status: MessageStatus.pending,
      createdAt: now.add(const Duration(milliseconds: 1)),
      streaming: true,
    );

    state = AsyncData(
      current.copyWith(
        messages: [...current.messages, userMsg, assistantMsg],
        sending: true,
      ),
    );

    final cancelToken = CancelToken();
    _cancelToken = cancelToken;
    final buffer = StringBuffer();
    // Fertilizer cards stream in alongside text deltas; collect them so each
    // delta patch (which rebuilds content) keeps the cards instead of wiping
    // them, and so they persist into the finished message.
    final cards = <ContentBlock>[];
    var failed = false;
    String? errorCode;

    try {
      await for (final event in _repo.sendMessage(
        text: trimmed,
        imageStorageKeys: imageStorageKeys,
        audioStorageKey: audioStorageKey,
        cancelToken: cancelToken,
      )) {
        switch (event) {
          case SseDelta(:final text):
            buffer.write(text);
            _patchAssistant(
              localAssistantId,
              (m) => m.copyWith(
                status: MessageStatus.pending,
                streaming: true,
                content: _composeContent(buffer.toString(), cards),
              ),
            );
          case SseTranscription(:final text, :final durationMs):
            // The user's voice transcription, emitted before the reply: attach
            // it to the optimistic user message so it shows under the player.
            _patchUser(
              localUserId,
              (m) => m.copyWith(
                content: [
                  ...m.content.where((b) => b.type != 'transcription'),
                  ContentBlock(
                    type: 'transcription',
                    text: text,
                    durationMs: durationMs,
                  ),
                ],
              ),
            );
          case SseFertilizerCard(:final products):
            if (products.isNotEmpty) {
              cards.add(
                ContentBlock(type: 'fertilizer_card', products: products),
              );
              _patchAssistant(
                localAssistantId,
                (m) => m.copyWith(
                  status: MessageStatus.pending,
                  streaming: true,
                  content: _composeContent(buffer.toString(), cards),
                ),
              );
            }
          case SseError(:final code):
            failed = true;
            errorCode = code;
          case SseMessageStarted():
          case SseToolUse():
          case SseDone():
            break;
        }
      }
      _finishStream(
        localAssistantId,
        buffer.toString(),
        cards,
        failed ? MessageStatus.failed : MessageStatus.complete,
        errorCode,
      );
      unawaited(_refreshFromServer());
    } on DioException catch (e) {
      if (CancelToken.isCancel(e)) {
        // User stopped the stream — the backend persists 'cancelled' + partial.
        _finishStream(
          localAssistantId,
          buffer.toString(),
          cards,
          MessageStatus.cancelled,
          null,
        );
        unawaited(_refreshFromServer());
      } else {
        final mapped = mapDioException(e);
        _finishStream(
          localAssistantId,
          buffer.toString(),
          cards,
          MessageStatus.failed,
          mapped is ApiException ? mapped.code : 'network',
        );
      }
    } on ApiException catch (e) {
      // Pre-stream rejection (rate limit, too large, …). Nothing persisted on
      // the server, so keep the optimistic pair locally as failed.
      _finishStream(
        localAssistantId,
        buffer.toString(),
        cards,
        MessageStatus.failed,
        e.code,
      );
    } finally {
      _cancelToken = null;
    }
  }

  /// Builds the assistant bubble content from the streamed text and any
  /// fertilizer cards: the text block first (when non-empty), then the cards.
  List<ContentBlock> _composeContent(String text, List<ContentBlock> cards) {
    return [
      if (text.isNotEmpty) ContentBlock(type: 'text', text: text),
      ...cards,
    ];
  }

  /// Cancels the in-flight stream (closes the connection).
  void cancel() => _cancelToken?.cancel('user_cancelled');

  /// Resends the last user message after a failed/cancelled reply, preserving
  /// both its text and any attached photos (still valid in object storage).
  Future<void> retry() async {
    final current = state.valueOrNull;
    if (current == null || current.sending) {
      return;
    }
    String text = '';
    final imageKeys = <String>[];
    var found = false;
    for (final m in current.messages.reversed) {
      if (m.role == MessageRole.user && m.content.isNotEmpty) {
        for (final b in m.content) {
          if (b.type == 'text' && text.isEmpty) {
            text = b.text;
          } else if (b.type == 'image' && b.storageKey.isNotEmpty) {
            imageKeys.add(b.storageKey);
          }
        }
        found = true;
        break;
      }
    }
    if (!found || (text.isEmpty && imageKeys.isEmpty)) {
      return;
    }
    final trimmed = [...current.messages];
    while (trimmed.isNotEmpty &&
        trimmed.last.role == MessageRole.assistant &&
        trimmed.last.status != MessageStatus.complete) {
      trimmed.removeLast();
    }
    if (trimmed.isNotEmpty && trimmed.last.role == MessageRole.user) {
      trimmed.removeLast();
    }
    state = AsyncData(current.copyWith(messages: trimmed));
    await sendMessage(text, imageStorageKeys: imageKeys);
  }

  /// Regenerates the answer to the most recent turn: drops the last assistant
  /// message and re-sends the preceding user message. No-op while sending or if
  /// the last message is not a finished assistant turn.
  Future<void> regenerate() async {
    final current = state.valueOrNull;
    if (current == null || current.sending || current.messages.isEmpty) {
      return;
    }
    final last = current.messages.last;
    if (last.role != MessageRole.assistant ||
        last.status == MessageStatus.pending) {
      return;
    }
    state = AsyncData(
      current.copyWith(messages: [...current.messages]..removeLast()),
    );
    await retry();
  }

  /// Sets or clears thumbs feedback on an assistant message. Optimistic: the UI
  /// updates immediately; a failed request (e.g. offline) is swallowed so the
  /// affordance never blocks. Server-side optimistic messages have no id yet.
  void setFeedback(String messageId, String? value) {
    if (messageId.startsWith('local-')) {
      return;
    }
    final current = state.valueOrNull;
    if (current == null) {
      return;
    }
    final next = Map<String, String>.from(current.feedback);
    if (value == null) {
      next.remove(messageId);
    } else {
      next[messageId] = value;
    }
    state = AsyncData(current.copyWith(feedback: next));
    unawaited(_repo.setFeedback(messageId, value).catchError((_) {}));
  }

  /// Loads an older page of history (keyset pagination).
  Future<void> loadOlder() async {
    final current = state.valueOrNull;
    if (current == null || current.loadingOlder || current.nextCursor == null) {
      return;
    }
    state = AsyncData(current.copyWith(loadingOlder: true));
    try {
      final page = await _repo.fetchOlder(cursor: current.nextCursor!);
      await _repo.cache(page.messages);
      final ids = page.messages.map((m) => m.id).toSet();
      final merged = [
        ...page.messages,
        ...current.messages.where((m) => !ids.contains(m.id)),
      ];
      state = AsyncData(
        current.copyWith(
          messages: merged,
          loadingOlder: false,
          nextCursor: page.nextCursor,
          clearCursor: page.nextCursor == null,
        ),
      );
    } on AppException {
      state = AsyncData(current.copyWith(loadingOlder: false));
    }
  }

  /// Records a tap on a fertilizer card (internal analytics). Fire-and-forget:
  /// a failed analytics ping must never disrupt the user opening the product.
  void recordFertilizerTap(String slug) {
    if (slug.isEmpty) {
      return;
    }
    unawaited(_repo.recordFertilizerTap(slug).catchError((_) {}));
  }

  /// Deletes a message (server + cache). Throws [AppException] on failure.
  Future<void> deleteMessage(String id) async {
    await _repo.deleteMessage(id);
    final current = state.valueOrNull;
    if (current == null) {
      return;
    }
    state = AsyncData(
      current.copyWith(
        messages: current.messages.where((m) => m.id != id).toList(),
      ),
    );
  }

  Future<void> _refreshFromServer() async {
    try {
      final page = await _repo.fetchConversation();
      await _repo.cache(page.messages);
      final current = state.valueOrNull;
      state = AsyncData(
        ChatState(
          messages: page.messages,
          nextCursor: page.nextCursor,
          sending: false,
          loadingOlder: current?.loadingOlder ?? false,
          feedback: current?.feedback ?? const {},
        ),
      );
    } on AppException {
      // Offline / transient: keep the local state, just clear the sending flag.
      final current = state.valueOrNull;
      if (current != null) {
        state = AsyncData(current.copyWith(sending: false));
      }
    }
  }

  void _finishStream(
    String localAssistantId,
    String partial,
    List<ContentBlock> cards,
    MessageStatus status,
    String? errorCode,
  ) {
    final current = state.valueOrNull;
    if (current == null) {
      return;
    }
    final content = _composeContent(partial, cards);
    state = AsyncData(
      current.copyWith(
        sending: false,
        messages: [
          for (final m in current.messages)
            if (m.id == localAssistantId)
              m.copyWith(
                status: status,
                streaming: false,
                content: content,
                errorCode: errorCode,
              )
            else
              m,
        ],
      ),
    );
  }

  void _patchAssistant(
    String localAssistantId,
    ChatMessage Function(ChatMessage) update,
  ) {
    final current = state.valueOrNull;
    if (current == null) {
      return;
    }
    state = AsyncData(
      current.copyWith(
        messages: [
          for (final m in current.messages)
            if (m.id == localAssistantId) update(m) else m,
        ],
      ),
    );
  }

  void _patchUser(
    String localUserId,
    ChatMessage Function(ChatMessage) update,
  ) {
    final current = state.valueOrNull;
    if (current == null) {
      return;
    }
    state = AsyncData(
      current.copyWith(
        messages: [
          for (final m in current.messages)
            if (m.id == localUserId) update(m) else m,
        ],
      ),
    );
  }

  String _localId(String prefix) =>
      'local-$prefix-${DateTime.now().microsecondsSinceEpoch}-${_seq++}';
}

final chatControllerProvider = AsyncNotifierProvider<ChatController, ChatState>(
  ChatController.new,
);
