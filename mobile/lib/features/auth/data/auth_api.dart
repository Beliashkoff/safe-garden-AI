import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/api_exception.dart';
import '../domain/auth_models.dart';

/// Thin wrapper over the auth/account HTTP endpoints. Every method translates
/// transport errors into [AppException] so callers never see Dio types.
class AuthApi {
  AuthApi(this._client);

  final ApiClient _client;

  Dio get _dio => _client.dio;

  Future<SignInResponse> signInApple({
    required String idToken,
    required String nonce,
  }) => _signIn('/auth/apple', {'id_token': idToken, 'nonce': nonce});

  Future<SignInResponse> signInGoogle({required String idToken}) =>
      _signIn('/auth/google', {'id_token': idToken});

  Future<SignInResponse> verifyEmailCode({
    required String email,
    required String code,
  }) => _signIn('/auth/email/verify', {'email': email, 'code': code});

  Future<void> requestEmailCode(String email) async {
    try {
      await _dio.post<dynamic>('/auth/email/request', data: {'email': email});
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  Future<void> logout(String refreshToken) async {
    try {
      await _dio.post<dynamic>(
        '/auth/logout',
        data: {'refresh_token': refreshToken},
      );
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  Future<AppUser> getAccount() async {
    try {
      final resp = await _dio.get<dynamic>('/account');
      final data = (resp.data as Map).cast<String, dynamic>();
      return AppUser.fromJson((data['user'] as Map).cast<String, dynamic>());
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  Future<void> deleteAccount() async {
    try {
      await _dio.delete<dynamic>('/account');
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }

  Future<SignInResponse> _signIn(String path, Map<String, dynamic> body) async {
    try {
      final resp = await _dio.post<dynamic>(path, data: body);
      return SignInResponse.fromJson(
        (resp.data as Map).cast<String, dynamic>(),
      );
    } on DioException catch (e) {
      throw mapDioException(e);
    }
  }
}

final authApiProvider = Provider<AuthApi>(
  (ref) => AuthApi(ref.watch(apiClientProvider)),
);
