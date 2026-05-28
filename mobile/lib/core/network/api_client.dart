import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../config/app_config.dart';
import '../storage/secure_token_store.dart';

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
