import 'package:agronom_ai/core/network/api_exception.dart';
import 'package:agronom_ai/features/auth/data/auth_api.dart';
import 'package:agronom_ai/features/auth/data/auth_repository.dart';
import 'package:agronom_ai/features/auth/domain/auth_models.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../../../helpers/fakes.dart';

class MockAuthApi extends Mock implements AuthApi {}

SignInResponse _signIn({bool yandex = false, bool vk = false}) =>
    SignInResponse(
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
      user: AppUser(
        id: 'user-1',
        email: 'u@example.com',
        emailVerified: true,
        providers: AuthProviders(email: !yandex && !vk, yandex: yandex, vk: vk),
      ),
    );

void main() {
  late MockAuthApi api;
  late FakeTokenStore store;
  late FakeOAuthProvider oauth;
  late AuthRepository repo;

  setUp(() {
    api = MockAuthApi();
    store = FakeTokenStore();
    oauth = FakeOAuthProvider();
    repo = AuthRepository(api: api, store: store, oauth: oauth);
  });

  test('signInWithYandex runs start → browser → complete and persists '
      'tokens', () async {
    when(() => api.startYandex()).thenAnswer(
      (_) async => const OAuthStartResponse(
        state: 'state-1',
        authUrl: 'https://oauth.yandex.ru/authorize?x=1',
      ),
    );
    when(
      () => api.completeYandex(
        code: any(named: 'code'),
        state: any(named: 'state'),
      ),
    ).thenAnswer((_) async => _signIn(yandex: true));

    final user = await repo.signInWithYandex();

    expect(user.providers.yandex, isTrue);
    expect(store.access, 'access-1');
    expect(store.refresh, 'refresh-1');
    expect(oauth.lastYandexAuthUrl, 'https://oauth.yandex.ru/authorize?x=1');
    expect(oauth.lastYandexExpectedState, 'state-1');
    verify(
      () => api.completeYandex(code: 'ya-code', state: 'state-1'),
    ).called(1);
  });

  test('signInWithVk passes backend PKCE material to the SDK and sends '
      'code + device_id back', () async {
    when(() => api.startVk()).thenAnswer(
      (_) async => const OAuthStartResponse(
        state: 'state-2',
        codeChallenge: 'challenge-2',
      ),
    );
    when(
      () => api.completeVk(
        code: any(named: 'code'),
        state: any(named: 'state'),
        deviceId: any(named: 'deviceId'),
      ),
    ).thenAnswer((_) async => _signIn(vk: true));

    final user = await repo.signInWithVk();

    expect(user.providers.vk, isTrue);
    expect(store.refresh, 'refresh-1');
    expect(oauth.lastVkState, 'state-2');
    expect(oauth.lastVkCodeChallenge, 'challenge-2');
    verify(
      () => api.completeVk(
        code: 'vk-code',
        state: 'state-2',
        deviceId: 'device-1',
      ),
    ).called(1);
  });

  test('verifyEmailCode persists tokens', () async {
    when(
      () => api.verifyEmailCode(
        email: any(named: 'email'),
        code: any(named: 'code'),
      ),
    ).thenAnswer((_) async => _signIn());

    final user = await repo.verifyEmailCode(
      email: 'u@example.com',
      code: '123456',
    );

    expect(user.id, 'user-1');
    expect(store.access, 'access-1');
  });

  test(
    'verifyEmailCode propagates invalid-code error and does not store',
    () async {
      when(
        () => api.verifyEmailCode(
          email: any(named: 'email'),
          code: any(named: 'code'),
        ),
      ).thenThrow(
        const ApiException(code: 'unauthorized', message: 'bad code'),
      );

      await expectLater(
        repo.verifyEmailCode(email: 'u@example.com', code: '000000'),
        throwsA(isA<ApiException>()),
      );
      expect(store.access, isNull);
    },
  );

  test('tryRestoreSession returns null without a refresh token', () async {
    expect(await repo.tryRestoreSession(), isNull);
    verifyNever(() => api.getAccount());
  });

  test('tryRestoreSession returns user when account fetch succeeds', () async {
    store.refresh = 'refresh-1';
    when(
      () => api.getAccount(),
    ).thenAnswer((_) async => const AppUser(id: 'user-1', emailVerified: true));
    final user = await repo.tryRestoreSession();
    expect(user?.id, 'user-1');
  });

  test('tryRestoreSession clears and returns null on ApiException', () async {
    store
      ..access = 'a'
      ..refresh = 'refresh-1';
    when(
      () => api.getAccount(),
    ).thenThrow(const ApiException(code: 'unauthorized', message: 'x'));

    expect(await repo.tryRestoreSession(), isNull);
    expect(store.refresh, isNull);
  });

  test('tryRestoreSession rethrows network errors', () async {
    store.refresh = 'refresh-1';
    when(() => api.getAccount()).thenThrow(const NetworkException());
    await expectLater(
      repo.tryRestoreSession(),
      throwsA(isA<NetworkException>()),
    );
  });

  test('logout calls server revoke and clears tokens', () async {
    store
      ..access = 'a'
      ..refresh = 'refresh-1';
    when(() => api.logout(any())).thenAnswer((_) async {});

    await repo.logout();

    verify(() => api.logout('refresh-1')).called(1);
    expect(store.access, isNull);
    expect(store.refresh, isNull);
  });

  test('deleteAccount calls api and clears tokens', () async {
    store
      ..access = 'a'
      ..refresh = 'refresh-1';
    when(() => api.deleteAccount()).thenAnswer((_) async {});

    await repo.deleteAccount();

    verify(() => api.deleteAccount()).called(1);
    expect(store.refresh, isNull);
  });
}
