import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../domain/chat_models.dart';
import 'chat_api.dart';
import 'local/chat_database.dart';
import 'media_cache.dart';

/// Coordinates the chat API and the local drift cache. The controller talks
/// only to this. Server history is the source of truth; the cache mirrors it.
class ChatRepository {
  ChatRepository({
    required ChatApi api,
    required ChatDatabase db,
    required MediaCache media,
  }) : _api = api,
       _db = db,
       _media = media;

  final ChatApi _api;
  final ChatDatabase _db;
  final MediaCache _media;

  /// Cached messages for instant/offline display at startup.
  Future<List<ChatMessage>> loadCached() => _db.loadMessages();

  /// Latest page from the server. Caller mirrors the result into the cache.
  Future<ConversationPage> fetchConversation({int? limit}) =>
      _api.getConversation(limit: limit);

  /// An older keyset page.
  Future<ConversationPage> fetchOlder({required String cursor, int? limit}) =>
      _api.listMessages(cursor: cursor, limit: limit);

  /// Mirrors messages into the cache (insert-or-replace).
  Future<void> cache(List<ChatMessage> messages) =>
      _db.upsertMessages(messages);

  /// Streams the assistant reply to a message of text and/or image refs.
  Stream<SseEvent> sendMessage({
    required String text,
    List<String> imageStorageKeys = const [],
    CancelToken? cancelToken,
  }) => _api.sendMessage(
    text: text,
    imageStorageKeys: imageStorageKeys,
    cancelToken: cancelToken,
  );

  /// Deletes a message on the server, then from the cache.
  Future<void> deleteMessage(String id) async {
    await _api.deleteMessage(id);
    await _db.deleteMessage(id);
  }

  /// Drops the local cache (logout / account deletion): both the message cache
  /// and the on-device photo files.
  Future<void> clearCache() async {
    await _db.clear();
    await _media.clear();
  }
}

final chatRepositoryProvider = Provider<ChatRepository>((ref) {
  return ChatRepository(
    api: ref.watch(chatApiProvider),
    db: ref.watch(chatDatabaseProvider),
    media: ref.watch(mediaCacheProvider),
  );
});
