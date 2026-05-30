import 'package:agronom_ai/core/network/api_exception.dart';
import 'package:agronom_ai/features/auth/application/auth_controller.dart';
import 'package:agronom_ai/features/chat/data/chat_repository.dart';
import 'package:agronom_ai/features/chat/domain/chat_models.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Scriptable [ChatRepository] for controller/widget tests — never touches the
/// network or drift. Implementing the concrete repository lets us override
/// [chatRepositoryProvider] directly, bypassing the api/database layers.
class FakeChatRepository implements ChatRepository {
  List<ChatMessage> cached = const [];

  /// When non-null, [fetchConversation]/[fetchOlder] throw this. Defaults to an
  /// offline error so `build()` (which catches [AppException]) falls back to the
  /// cache without hitting the network.
  Object? fetchError = const NetworkException();
  ConversationPage fetchResult = const ConversationPage();

  /// Produces the SSE stream returned by [sendMessage].
  Stream<SseEvent> Function()? scriptedStream;

  final List<String> deleted = [];
  int clearCacheCount = 0;

  @override
  Future<List<ChatMessage>> loadCached() async => cached;

  @override
  Future<ConversationPage> fetchConversation({int? limit}) async {
    final error = fetchError;
    if (error != null) {
      throw error;
    }
    return fetchResult;
  }

  @override
  Future<ConversationPage> fetchOlder({
    required String cursor,
    int? limit,
  }) async {
    final error = fetchError;
    if (error != null) {
      throw error;
    }
    return fetchResult;
  }

  @override
  Future<void> cache(List<ChatMessage> messages) async {}

  @override
  Stream<SseEvent> sendMessage({
    required String text,
    CancelToken? cancelToken,
  }) {
    final build = scriptedStream;
    return build == null ? const Stream<SseEvent>.empty() : build();
  }

  @override
  Future<void> deleteMessage(String id) async => deleted.add(id);

  @override
  Future<void> clearCache() async => clearCacheCount++;
}

/// Provider overrides for chat tests: the fake repository plus a fixed auth
/// status (so the chat controller does not build the real auth graph).
List<Override> chatTestOverrides(
  FakeChatRepository repo, {
  AuthStatus authStatus = AuthStatus.authenticated,
}) {
  return [
    chatRepositoryProvider.overrideWithValue(repo),
    authStatusProvider.overrideWithValue(authStatus),
  ];
}

ChatMessage textMessage({
  required String id,
  required MessageRole role,
  MessageStatus status = MessageStatus.complete,
  String text = 'hello',
  DateTime? createdAt,
}) {
  return ChatMessage(
    id: id,
    role: role,
    status: status,
    createdAt: createdAt ?? DateTime.utc(2026, 1, 1),
    content: [ContentBlock(type: 'text', text: text)],
  );
}
