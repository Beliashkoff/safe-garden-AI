import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../config/app_config.dart';
import '../storage/secure_token_store.dart';
import 'api_exception.dart';

/// The single HTTP entry point for the app. Wraps a [Dio] configured with:
///  - a base URL,
///  - a request interceptor adding the bearer token (for non-/auth paths) and
///    an Accept-Language header,
///  - a response-error interceptor that transparently refreshes the access
///    token on 401 and retries the original request once.
///
/// No screen or repository talks to [Dio] directly; they use [dio] from here.
class ApiClient {
  ApiClient({
    required Dio dio,
    required Dio refreshDio,
    required TokenStore store,
    this.onSessionExpired,
    String localeCode = 'ru',
  }) : _dio = dio,
       _refreshDio = refreshDio,
       _store = store,
       _localeCode = localeCode {
    _dio.options.baseUrl = AppConfig.apiBaseUrl;
    _refreshDio.options.baseUrl = AppConfig.apiBaseUrl;
    _dio.interceptors.add(
      InterceptorsWrapper(onRequest: _onRequest, onError: _onError),
    );
  }

  final Dio _dio;
  final Dio _refreshDio;
  final TokenStore _store;

  /// Invoked when a refresh attempt fails — the session is unrecoverable and
  /// the user must sign in again. Set by the auth controller.
  void Function()? onSessionExpired;

  String _localeCode;

  /// The configured Dio for repositories to issue requests through.
  Dio get dio => _dio;

  /// Updates the Accept-Language sent with subsequent requests (e.g. so the OTP
  /// email matches the UI language).
  void setLocale(String localeCode) => _localeCode = localeCode;

  // De-duplicates concurrent refreshes: the first 401 starts a refresh, the
  // rest await the same future.
  Future<bool>? _refreshing;

  Future<void> _onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    options.headers['Accept-Language'] = _localeCode;
    if (!_isAuthPath(options.path)) {
      final access = await _store.readAccessToken();
      if (access != null && access.isNotEmpty) {
        options.headers['Authorization'] = 'Bearer $access';
      }
    }
    handler.next(options);
  }

  Future<void> _onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    final options = err.requestOptions;
    final isUnauthorized = err.response?.statusCode == 401;
    final alreadyRetried = options.extra['__retried'] == true;

    if (!isUnauthorized || _isAuthPath(options.path) || alreadyRetried) {
      handler.next(err);
      return;
    }

    final refreshed = await _ensureRefreshed();
    if (!refreshed) {
      onSessionExpired?.call();
      handler.next(err);
      return;
    }

    final access = await _store.readAccessToken();
    options.extra['__retried'] = true;
    options.headers['Authorization'] = 'Bearer $access';
    try {
      final response = await _dio.fetch<dynamic>(options);
      handler.resolve(response);
    } on DioException catch (retryErr) {
      handler.next(retryErr);
    }
  }

  Future<bool> _ensureRefreshed() {
    return _refreshing ??= _doRefresh().whenComplete(() => _refreshing = null);
  }

  Future<bool> _doRefresh() async {
    final refresh = await _store.readRefreshToken();
    if (refresh == null || refresh.isEmpty) {
      return false;
    }
    try {
      final response = await _refreshDio.post<dynamic>(
        '/auth/refresh',
        data: {'refresh_token': refresh},
      );
      final data = response.data as Map;
      await _store.writeTokens(
        accessToken: data['access_token'] as String,
        refreshToken: data['refresh_token'] as String,
      );
      return true;
    } on DioException {
      // Refresh rejected (expired / reuse-detected family revoke). Drop tokens.
      await _store.clear();
      return false;
    }
  }

  /// Opens a server-sent-events POST to [path] with JSON [body] and returns the
  /// raw byte stream of the 200 response; the caller frames the SSE events.
  ///
  /// A streaming request uses a permissive validateStatus so Dio never throws —
  /// which also means the [_onError] refresh-on-401 interceptor does not run for
  /// it. A 401 at open time is therefore handled here: refresh once and retry.
  /// Any non-200 response has its body drained and decoded into an
  /// [ApiException] (the same §4.7 envelope [mapDioException] parses).
  Future<Stream<List<int>>> openEventStream({
    required String path,
    required Object body,
    CancelToken? cancelToken,
  }) async {
    Future<Response<ResponseBody>> open() => _dio.post<ResponseBody>(
      path,
      data: body,
      cancelToken: cancelToken,
      options: Options(
        responseType: ResponseType.stream,
        headers: {'Accept': 'text/event-stream'},
        validateStatus: (_) => true,
      ),
    );

    var response = await open();
    if (response.statusCode == 401) {
      final refreshed = await _ensureRefreshed();
      if (!refreshed) {
        onSessionExpired?.call();
        throw const ApiException(
          code: 'unauthorized',
          message: 'session expired',
          statusCode: 401,
        );
      }
      response = await open();
    }

    final status = response.statusCode ?? 0;
    if (status != 200) {
      if (status == 401) {
        onSessionExpired?.call();
      }
      throw await _apiExceptionFromBody(response);
    }
    return response.data!.stream;
  }

  /// Drains a non-200 streamed response and decodes the §4.7 error envelope,
  /// falling back to a generic error if the body is missing or malformed.
  Future<ApiException> _apiExceptionFromBody(
    Response<ResponseBody> response,
  ) async {
    final status = response.statusCode;
    try {
      final chunks = <int>[];
      await for (final chunk in response.data!.stream) {
        chunks.addAll(chunk);
      }
      final decoded = jsonDecode(utf8.decode(chunks, allowMalformed: true));
      if (decoded is Map) {
        final error = decoded['error'];
        if (error is Map) {
          return ApiException(
            code: (error['code'] as String?) ?? 'internal_error',
            message: (error['message'] as String?) ?? 'Unexpected error',
            details: (error['details'] as Map?)?.cast<String, dynamic>(),
            requestId: decoded['request_id'] as String?,
            statusCode: status,
          );
        }
      }
    } on Object {
      // Malformed / empty body — fall through to the generic error.
    }
    return ApiException(
      code: 'internal_error',
      message: 'Unexpected error',
      statusCode: status,
    );
  }

  static bool _isAuthPath(String path) => path.contains('/auth/');
}

/// Built once and shared. The auth controller sets [ApiClient.onSessionExpired].
final apiClientProvider = Provider<ApiClient>((ref) {
  return ApiClient(
    dio: Dio(),
    refreshDio: Dio(),
    store: ref.watch(secureTokenStoreProvider),
  );
});
