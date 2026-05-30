import 'package:freezed_annotation/freezed_annotation.dart';

part 'chat_models.freezed.dart';
part 'chat_models.g.dart';

/// Author of a message. Mirrors the backend `role` field.
enum MessageRole {
  @JsonValue('user')
  user,
  @JsonValue('assistant')
  assistant,
  @JsonValue('system')
  system,
}

/// Lifecycle of a message. Assistant turns move pending → complete, or land on
/// cancelled (connection closed mid-stream) / failed (upstream error). User
/// turns are always complete.
enum MessageStatus {
  @JsonValue('pending')
  pending,
  @JsonValue('complete')
  complete,
  @JsonValue('cancelled')
  cancelled,
  @JsonValue('failed')
  failed,
}

/// A single content block of a message. `type` is `text` or `image` (stage 3.3;
/// `audio` follows in stage 4). For an image block, [storageKey] is the
/// owner-scoped object key the bubble resolves to a file via the media cache;
/// [text] carries the text body otherwise.
@freezed
class ContentBlock with _$ContentBlock {
  const factory ContentBlock({
    required String type,
    @Default('') String text,
    @JsonKey(name: 'storage_key') @Default('') String storageKey,
  }) = _ContentBlock;

  factory ContentBlock.fromJson(Map<String, dynamic> json) =>
      _$ContentBlockFromJson(json);
}

/// A chat message as shown in the UI. [streaming] is a transient client-only
/// flag (the assistant bubble is currently receiving deltas); it is never part
/// of the server JSON.
@freezed
class ChatMessage with _$ChatMessage {
  const factory ChatMessage({
    required String id,
    required MessageRole role,
    required MessageStatus status,
    @JsonKey(name: 'created_at') required DateTime createdAt,
    @Default(<ContentBlock>[]) List<ContentBlock> content,
    @JsonKey(includeFromJson: false, includeToJson: false)
    @Default(false)
    bool streaming,
    @JsonKey(includeFromJson: false, includeToJson: false) String? errorCode,
  }) = _ChatMessage;

  factory ChatMessage.fromJson(Map<String, dynamic> json) =>
      _$ChatMessageFromJson(json);
}

/// A page of conversation history. [nextCursor] is the opaque keyset cursor for
/// the next (older) page, or null when there is no more history. The backend's
/// conversation `id` is ignored — there is one conversation per user.
@freezed
class ConversationPage with _$ConversationPage {
  const factory ConversationPage({
    @Default(<ChatMessage>[]) List<ChatMessage> messages,
    @JsonKey(name: 'next_cursor') String? nextCursor,
  }) = _ConversationPage;

  factory ConversationPage.fromJson(Map<String, dynamic> json) =>
      _$ConversationPageFromJson(json);
}

/// A parsed server-sent event from `POST /v1/messages`. Constructed by the SSE
/// parser, not deserialized directly — hence a plain sealed hierarchy rather
/// than a json-serializable union.
sealed class SseEvent {
  const SseEvent();
}

/// The assistant message was created (status pending); carries its stable id.
class SseMessageStarted extends SseEvent {
  const SseMessageStarted(this.messageId);

  final String messageId;
}

/// A chunk of assistant text to append.
class SseDelta extends SseEvent {
  const SseDelta(this.text);

  final String text;
}

/// A tool invocation passed through from the worker (stage 5; ignored for now).
class SseToolUse extends SseEvent {
  const SseToolUse({required this.tool, required this.args});

  final String tool;
  final Map<String, dynamic> args;
}

/// A fertilizer recommendation card (stage 5; ignored for now).
class SseFertilizerCard extends SseEvent {
  const SseFertilizerCard(this.data);

  final Map<String, dynamic> data;
}

/// A terminal error event — the assistant turn failed.
class SseError extends SseEvent {
  const SseError({required this.code, required this.message});

  final String code;
  final String message;
}

/// The terminal success event — carries the final message id and token usage.
class SseDone extends SseEvent {
  const SseDone({
    required this.messageId,
    required this.tokensIn,
    required this.tokensOut,
  });

  final String messageId;
  final int tokensIn;
  final int tokensOut;
}
