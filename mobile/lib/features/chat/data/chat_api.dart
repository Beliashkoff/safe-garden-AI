import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/api_exception.dart';
import '../domain/chat_models.dart';
import 'sse_parser.dart';

/// Thin wrapper over the chat HTTP endpoints (ARCH §4.3). JSON calls translate
/// transport errors into [AppException]; [sendMessage] streams typed events.
class ChatApi {
  ChatApi(this._client);

  final ApiClient _client;

  Dio get _dio => _client.dio;

  Future<ConversationPage> getConversation({int? limit}) async {
    try {
      final resp = await _dio.get<dynamic>(
        '/conversation',
        queryParameters: {'limit': ?limit},
      );
      return ConversationPage.fromJson(
        (resp.data as Map).cast<String, dynamic>(),
      );
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  Future<ConversationPage> listMessages({
    required String cursor,
    int? limit,
  }) async {
    try {
      final resp = await _dio.get<dynamic>(
        '/conversation/messages',
        queryParameters: {'cursor': cursor, 'limit': ?limit},
      );
      return ConversationPage.fromJson(
        (resp.data as Map).cast<String, dynamic>(),
      );
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  Future<void> deleteMessage(String id) async {
    try {
      await _dio.delete<dynamic>('/messages/$id');
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  /// Streams the assistant reply to a text message as typed [SseEvent]s. Errors
  /// before the stream opens (validation, rate limit, auth) surface as the
  /// future failing with an [ApiException]; mid-stream failures arrive as an
  /// [SseError] event or as a Dio cancellation when [cancelToken] is cancelled.
  Stream<SseEvent> sendMessage({
    required String text,
    CancelToken? cancelToken,
  }) async* {
    final bytes = await _client.openEventStream(
      path: '/messages',
      body: {
        'content': [
          {'type': 'text', 'text': text},
        ],
      },
      cancelToken: cancelToken,
    );
    yield* parseSse(bytes);
  }
}

final chatApiProvider = Provider<ChatApi>(
  (ref) => ChatApi(ref.watch(apiClientProvider)),
);
