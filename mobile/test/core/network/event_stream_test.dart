import 'dart:convert';
import 'dart:typed_data';

import 'package:agronom_ai/core/network/api_client.dart';
import 'package:agronom_ai/core/network/api_exception.dart';
import 'package:agronom_ai/core/storage/secure_token_store.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeTokenStore implements TokenStore {
  _FakeTokenStore({this.access, this.refresh});

  String? access;
  String? refresh;

  @override
  Future<String?> readAccessToken() async => access;

  @override
  Future<String?> readRefreshToken() async => refresh;

  @override
  Future<void> writeTokens({
    required String accessToken,
    required String refreshToken,
  }) async {
    access = accessToken;
    refresh = refreshToken;
  }

  @override
  Future<void> clear() async {
    access = null;
    refresh = null;
  }
}

typedef _Handler = ResponseBody Function(RequestOptions options);

class _FakeAdapter implements HttpClientAdapter {
  final Map<String, _Handler> handlers = {};

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final handler = handlers[options.path];
    return handler == null
        ? ResponseBody.fromString('{}', 404)
        : handler(options);
  }

  @override
  void close({bool force = false}) {}
}

final _jsonHeaders = {
  Headers.contentTypeHeader: ['application/json'],
};

ResponseBody _sse(String body) => ResponseBody.fromString(
  body,
  200,
  headers: {
    Headers.contentTypeHeader: ['text/event-stream'],
  },
);

ResponseBody _jsonError(int status, String code) => ResponseBody.fromString(
  jsonEncode({
    'error': {'code': code, 'message': code},
    'request_id': 'req-1',
  }),
  status,
  headers: _jsonHeaders,
);

ApiClient _client(
  _FakeAdapter adapter,
  TokenStore store, {
  void Function()? onSessionExpired,
}) {
  return ApiClient(
    dio: Dio()..httpClientAdapter = adapter,
    refreshDio: Dio()..httpClientAdapter = adapter,
    store: store,
    onSessionExpired: onSessionExpired,
  );
}

Future<String> _drain(Stream<List<int>> stream) async {
  final bytes = <int>[];
  await for (final chunk in stream) {
    bytes.addAll(chunk);
  }
  return utf8.decode(bytes);
}

void main() {
  test('returns the raw byte stream on 200', () async {
    const sse = 'event: done\ndata: {"message_id":"m1"}\n\n';
    final adapter = _FakeAdapter()..handlers['/messages'] = (_) => _sse(sse);
    final client = _client(adapter, _FakeTokenStore(access: 'a', refresh: 'r'));

    final stream = await client.openEventStream(
      path: '/messages',
      body: {'content': const <Object>[]},
    );

    expect(await _drain(stream), sse);
  });

  test('drains and maps a non-200 error envelope', () async {
    final adapter = _FakeAdapter()
      ..handlers['/messages'] = (_) => _jsonError(429, 'rate_limited');
    final client = _client(adapter, _FakeTokenStore(access: 'a', refresh: 'r'));

    await expectLater(
      client.openEventStream(
        path: '/messages',
        body: {'content': const <Object>[]},
      ),
      throwsA(
        isA<ApiException>().having((e) => e.code, 'code', 'rate_limited'),
      ),
    );
  });

  test('refreshes on 401 then retries the stream with the new token', () async {
    final store = _FakeTokenStore(access: 'old', refresh: 'r1');
    final adapter = _FakeAdapter();
    var messageCalls = 0;
    adapter.handlers['/auth/refresh'] = (_) => ResponseBody.fromString(
      jsonEncode({'access_token': 'new', 'refresh_token': 'r2'}),
      200,
      headers: _jsonHeaders,
    );
    adapter.handlers['/messages'] = (options) {
      messageCalls++;
      if (messageCalls == 1) {
        return _jsonError(401, 'unauthorized');
      }
      expect(options.headers['Authorization'], 'Bearer new');
      return _sse('event: done\ndata: {"message_id":"m"}\n\n');
    };

    var expired = false;
    final client = _client(
      adapter,
      store,
      onSessionExpired: () => expired = true,
    );

    final stream = await client.openEventStream(
      path: '/messages',
      body: {'content': const <Object>[]},
    );

    expect(await _drain(stream), contains('done'));
    expect(messageCalls, 2);
    expect(store.access, 'new');
    expect(expired, isFalse);
  });
}
