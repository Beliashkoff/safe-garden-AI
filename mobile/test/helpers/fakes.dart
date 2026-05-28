import 'package:agronom_ai/core/network/api_client.dart';
import 'package:agronom_ai/core/storage/secure_token_store.dart';
import 'package:agronom_ai/features/auth/data/auth_repository.dart';
import 'package:agronom_ai/features/auth/data/oauth_providers.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:mocktail/mocktail.dart';

/// In-memory [TokenStore] for tests.
class FakeTokenStore implements TokenStore {
  FakeTokenStore({this.access, this.refresh});

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

/// Scriptable [OAuthProvider] for tests (no native plugins).
class FakeOAuthProvider implements OAuthProvider {
  AppleCredential appleCredential = const AppleCredential(
    identityToken: 'apple-id-token',
    rawNonce: 'raw-nonce',
  );
  String googleIdToken = 'google-id-token';
  Object? appleError;
  Object? googleError;

  @override
  Future<AppleCredential> getAppleCredential() async {
    if (appleError != null) throw appleError!;
    return appleCredential;
  }

  @override
  Future<String> getGoogleIdToken() async {
    if (googleError != null) throw googleError!;
    return googleIdToken;
  }
}

/// Mock repository for controller/widget tests.
class MockAuthRepository extends Mock implements AuthRepository {}

/// Common provider overrides for widget tests: an in-memory token store and a
/// no-network ApiClient, plus the supplied (mock) repository. With the default
/// mock, the auth controller bootstraps to "unauthenticated".
List<Override> authTestOverrides(MockAuthRepository repo) {
  return [
    secureTokenStoreProvider.overrideWithValue(FakeTokenStore()),
    apiClientProvider.overrideWithValue(
      ApiClient(dio: Dio(), refreshDio: Dio(), store: FakeTokenStore()),
    ),
    authRepositoryProvider.overrideWithValue(repo),
  ];
}
