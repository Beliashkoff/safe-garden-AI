import 'package:agronom_ai/core/network/api_client.dart';
import 'package:agronom_ai/core/storage/secure_token_store.dart';
import 'package:agronom_ai/features/auth/data/auth_repository.dart';
import 'package:agronom_ai/features/auth/data/oauth_providers.dart';
import 'package:agronom_ai/features/onboarding/data/onboarding_store.dart';
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
  YandexAuthCode yandexCode = const YandexAuthCode(code: 'ya-code');
  VkAuthCode vkCode = const VkAuthCode(code: 'vk-code', deviceId: 'device-1');
  Object? yandexError;
  Object? vkError;

  String? lastYandexAuthUrl;
  String? lastYandexExpectedState;
  String? lastVkState;
  String? lastVkCodeChallenge;

  @override
  Future<YandexAuthCode> getYandexAuthCode({
    required String authUrl,
    required String expectedState,
  }) async {
    lastYandexAuthUrl = authUrl;
    lastYandexExpectedState = expectedState;
    if (yandexError != null) throw yandexError!;
    return yandexCode;
  }

  @override
  Future<VkAuthCode> getVkAuthCode({
    required String state,
    required String codeChallenge,
  }) async {
    lastVkState = state;
    lastVkCodeChallenge = codeChallenge;
    if (vkError != null) throw vkError!;
    return vkCode;
  }
}

/// In-memory [OnboardingStore] for tests. Defaults to "seen" so widget tests
/// exercise the auth/login flow without the intro gating it.
class FakeOnboardingStore implements OnboardingStore {
  FakeOnboardingStore({this.seen = true});

  bool seen;

  @override
  Future<bool> hasSeen() async => seen;

  @override
  Future<void> markSeen() async => seen = true;
}

/// Mock repository for controller/widget tests.
class MockAuthRepository extends Mock implements AuthRepository {}

/// Common provider overrides for widget tests: an in-memory token store and a
/// no-network ApiClient, plus the supplied (mock) repository. With the default
/// mock, the auth controller bootstraps to "unauthenticated"; onboarding is
/// pre-marked seen so it does not gate the login flow.
List<Override> authTestOverrides(MockAuthRepository repo) {
  return [
    secureTokenStoreProvider.overrideWithValue(FakeTokenStore()),
    apiClientProvider.overrideWithValue(
      ApiClient(dio: Dio(), refreshDio: Dio(), store: FakeTokenStore()),
    ),
    authRepositoryProvider.overrideWithValue(repo),
    onboardingStoreProvider.overrideWithValue(FakeOnboardingStore()),
  ];
}
