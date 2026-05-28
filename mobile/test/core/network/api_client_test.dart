import 'dart:async';
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
    if (handler == null) {
      return _json(404, {
        'error': {'code': 'not_found', 'message': 'no handler'},
      });
    }
    return handler(options);
  }

  @override
  void close({bool force = false}) {}
}

final _jsonHeaders = {
  Headers.contentTypeHeader: ['application/json'],
};

ResponseBody _json(int status, Object body) =>
    ResponseBody.fromString(jsonEncode(body), status, headers: _jsonHeaders);

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

void main() {
  group('mapDioException', () {
    test('parses the backend error envelope', () {
      final err = DioException(
        requestOptions: RequestOptions(path: '/account'),
        response: Response<dynamic>(
          requestOptions: RequestOptions(path: '/account'),
          statusCode: 429,
          data: {
            'error': {
              'code': 'rate_limited',
              'message': 'slow down',
              'details': {'field': 'email'},
            },
            'request_id': 'req-1',
          },
        ),
      );
      final mapped = mapDioException(err);
      expect(mapped, isA<ApiException>());
      final api = mapped as ApiException;
      expect(api.code, 'rate_limited');
      expect(api.message, 'slow down');
      expect(api.requestId, 'req-1');
      expect(api.statusCode, 429);
      expect(api.details?['field'], 'email');
    });

    test('falls back to NetworkException without a response', () {
      final err = DioException(
        requestOptions: RequestOptions(path: '/account'),
        type: DioExceptionType.connectionTimeout,
      );
      expect(mapDioException(err), isA<NetworkException>());
    });
  });

  group('refresh-on-401', () {
    test('refreshes and retries the original request once', () async {
      final store = _FakeTokenStore(access: 'old', refresh: 'r1');
      final adapter = _FakeAdapter();
      var accountCalls = 0;
      adapter.handlers['/auth/refresh'] = (_) =>
          _json(200, {'access_token': 'new', 'refresh_token': 'r2'});
      adapter.handlers['/account'] = (options) {
        accountCalls++;
        if (accountCalls == 1) {
          return _json(401, {
            'error': {'code': 'unauthorized', 'message': 'expired'},
          });
        }
        expect(options.headers['Authorization'], 'Bearer new');
        return _json(200, {'ok': true});
      };

      var expired = false;
      final client = _client(
        adapter,
        store,
        onSessionExpired: () => expired = true,
      );
      final resp = await client.dio.get<dynamic>('/account');

      expect(resp.statusCode, 200);
      expect(accountCalls, 2);
      expect(store.access, 'new');
      expect(store.refresh, 'r2');
      expect(expired, isFalse);
    });

    test('clears tokens and signals expiry when refresh fails', () async {
      final store = _FakeTokenStore(access: 'old', refresh: 'r1');
      final adapter = _FakeAdapter();
      adapter.handlers['/auth/refresh'] = (_) => _json(401, {
        'error': {'code': 'unauthorized', 'message': 'reuse'},
      });
      adapter.handlers['/account'] = (_) => _json(401, {
        'error': {'code': 'unauthorized', 'message': 'expired'},
      });

      var expired = false;
      final client = _client(
        adapter,
        store,
        onSessionExpired: () => expired = true,
      );

      await expectLater(
        client.dio.get<dynamic>('/account'),
        throwsA(isA<DioException>()),
      );
      expect(expired, isTrue);
      expect(store.access, isNull);
      expect(store.refresh, isNull);
    });
  });
}
