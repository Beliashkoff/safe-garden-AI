import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/api_exception.dart';

/// Result of `POST /v1/uploads/presign`: a presigned PUT URL plus the
/// owner-scoped storage key to later reference as an `image_ref` block.
class PresignResult {
  const PresignResult({
    required this.url,
    required this.key,
    required this.headers,
  });

  final String url;
  final String key;
  final Map<String, String> headers;
}

/// Result of `POST /v1/uploads/view`: a short-lived presigned GET URL.
class ViewResult {
  const ViewResult(this.url);

  final String url;
}

/// Wraps the upload endpoints and the direct-to-storage transfers. The presign
/// JSON calls go through the authenticated [ApiClient]; the PUT/GET against the
/// presigned URLs use a bare [Dio] — those URLs are absolute and already signed,
/// so the bearer token must NOT be attached (CLAUDE.md #4).
abstract interface class UploadApi {
  Future<PresignResult> presignPut({
    required String contentType,
    required int sizeBytes,
  });

  Future<void> putObject({
    required String url,
    required Map<String, String> headers,
    required List<int> bytes,
    ProgressCallback? onSendProgress,
    CancelToken? cancelToken,
  });

  Future<ViewResult> presignView({required String storageKey});

  Future<List<int>> downloadBytes({
    required String url,
    CancelToken? cancelToken,
  });
}

class HttpUploadApi implements UploadApi {
  HttpUploadApi(this._client, {Dio? rawDio}) : _raw = rawDio ?? Dio();

  final ApiClient _client;
  final Dio _raw;

  Dio get _dio => _client.dio;

  @override
  Future<PresignResult> presignPut({
    required String contentType,
    required int sizeBytes,
  }) async {
    try {
      final resp = await _dio.post<dynamic>(
        '/uploads/presign',
        data: {'content_type': contentType, 'size_bytes': sizeBytes},
      );
      final map = (resp.data as Map).cast<String, dynamic>();
      return PresignResult(
        url: map['url'] as String,
        key: map['key'] as String,
        headers: (map['headers'] as Map?)?.map(
              (k, v) => MapEntry(k.toString(), v.toString()),
            ) ??
            const {},
      );
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  @override
  Future<void> putObject({
    required String url,
    required Map<String, String> headers,
    required List<int> bytes,
    ProgressCallback? onSendProgress,
    CancelToken? cancelToken,
  }) async {
    try {
      await _raw.put<dynamic>(
        url,
        data: Stream<List<int>>.value(bytes),
        options: Options(
          headers: {...headers, Headers.contentLengthHeader: bytes.length},
          contentType: headers['Content-Type'],
          responseType: ResponseType.plain,
          validateStatus: (s) => s != null && s < 400,
        ),
        onSendProgress: onSendProgress,
        cancelToken: cancelToken,
      );
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  @override
  Future<ViewResult> presignView({required String storageKey}) async {
    try {
      final resp = await _dio.post<dynamic>(
        '/uploads/view',
        data: {'storage_key': storageKey},
      );
      final map = (resp.data as Map).cast<String, dynamic>();
      return ViewResult(map['url'] as String);
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  @override
  Future<List<int>> downloadBytes({
    required String url,
    CancelToken? cancelToken,
  }) async {
    try {
      final resp = await _raw.get<List<int>>(
        url,
        options: Options(responseType: ResponseType.bytes),
        cancelToken: cancelToken,
      );
      return resp.data ?? const [];
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }
}

final uploadApiProvider = Provider<UploadApi>(
  (ref) => HttpUploadApi(ref.watch(apiClientProvider)),
);
