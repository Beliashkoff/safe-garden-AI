import 'package:dio/dio.dart';

/// Base type for all failures surfaced to the app layer. [code] is a stable,
/// machine-readable key the UI maps to a localized message.
sealed class AppException implements Exception {
  const AppException({required this.code, required this.message});

  final String code;
  final String message;

  @override
  String toString() => 'AppException($code): $message';
}

/// A structured error returned by the backend in the §4.7 envelope:
/// `{ "error": { "code", "message", "details" }, "request_id" }`.
class ApiException extends AppException {
  const ApiException({
    required super.code,
    required super.message,
    this.details,
    this.requestId,
    this.statusCode,
  });

  final Map<String, dynamic>? details;
  final String? requestId;
  final int? statusCode;
}

/// Connectivity / timeout failure — no usable HTTP response was received.
class NetworkException extends AppException {
  const NetworkException({
    super.code = 'network',
    super.message = 'network error',
  });
}

/// Raised when the user dismisses a native OAuth sheet. The UI treats this as a
/// no-op rather than an error. Lives here so it shares the sealed hierarchy.
class OAuthCanceledException extends AppException {
  const OAuthCanceledException()
    : super(code: 'oauth_canceled', message: 'cancelled');
}

/// Translates a [DioException] into an [AppException], parsing the backend
/// error envelope when present and falling back to a network error otherwise.
AppException mapDioException(DioException e) {
  final response = e.response;
  if (response == null) {
    return const NetworkException();
  }
  final data = response.data;
  if (data is Map) {
    final error = data['error'];
    if (error is Map) {
      return ApiException(
        code: (error['code'] as String?) ?? 'internal_error',
        message: (error['message'] as String?) ?? 'Unexpected error',
        details: (error['details'] as Map?)?.cast<String, dynamic>(),
        requestId: data['request_id'] as String?,
        statusCode: response.statusCode,
      );
    }
  }
  return ApiException(
    code: 'internal_error',
    message: 'Unexpected error',
    statusCode: response.statusCode,
  );
}
