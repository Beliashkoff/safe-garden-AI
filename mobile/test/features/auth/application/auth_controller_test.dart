import 'package:agronom_ai/core/network/api_client.dart';
import 'package:agronom_ai/core/network/api_exception.dart';
import 'package:agronom_ai/features/auth/application/auth_controller.dart';
import 'package:agronom_ai/features/auth/data/auth_repository.dart';
import 'package:agronom_ai/features/auth/domain/auth_models.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../../../helpers/fakes.dart';

class MockAuthRepository extends Mock implements AuthRepository {}

const _user = AppUser(
  id: 'user-1',
  email: 'u@example.com',
  emailVerified: true,
);

ProviderContainer _container(MockAuthRepository repo) {
  final container = ProviderContainer(
    overrides: [
      authRepositoryProvider.overrideWithValue(repo),
      apiClientProvider.overrideWithValue(
        ApiClient(dio: Dio(), refreshDio: Dio(), store: FakeTokenStore()),
      ),
    ],
  );
  addTearDown(container.dispose);
  return container;
}

void main() {
  late MockAuthRepository repo;

  setUp(() => repo = MockAuthRepository());

  test('bootstrap with no session → unauthenticated', () async {
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => null);
    final c = _container(repo);

    final snap = await c.read(authControllerProvider.future);

    expect(snap, isA<Unauthenticated>());
    expect(c.read(authStatusProvider), AuthStatus.unauthenticated);
  });

  test('bootstrap with stored session → authenticated', () async {
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => _user);
    final c = _container(repo);

    final snap = await c.read(authControllerProvider.future);

    expect(snap, isA<Authenticated>());
    expect(c.read(authStatusProvider), AuthStatus.authenticated);
  });

  test('bootstrap offline → unauthenticated (no crash)', () async {
    when(() => repo.tryRestoreSession()).thenThrow(const NetworkException());
    final c = _container(repo);

    final snap = await c.read(authControllerProvider.future);
    expect(snap, isA<Unauthenticated>());
  });

  test('verifyEmailCode success → authenticated', () async {
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => null);
    when(
      () => repo.verifyEmailCode(
        email: any(named: 'email'),
        code: any(named: 'code'),
      ),
    ).thenAnswer((_) async => _user);
    final c = _container(repo);
    await c.read(authControllerProvider.future);

    await c
        .read(authControllerProvider.notifier)
        .verifyEmailCode(email: 'u@example.com', code: '123456');

    expect(c.read(authStatusProvider), AuthStatus.authenticated);
  });

  test('verifyEmailCode error propagates and stays unauthenticated', () async {
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => null);
    when(
      () => repo.verifyEmailCode(
        email: any(named: 'email'),
        code: any(named: 'code'),
      ),
    ).thenThrow(const ApiException(code: 'unauthorized', message: 'bad'));
    final c = _container(repo);
    await c.read(authControllerProvider.future);

    await expectLater(
      c
          .read(authControllerProvider.notifier)
          .verifyEmailCode(email: 'u@example.com', code: '000000'),
      throwsA(isA<ApiException>()),
    );
    expect(c.read(authStatusProvider), AuthStatus.unauthenticated);
  });

  test('cancelled Apple sign-in is silent', () async {
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => null);
    when(
      () => repo.signInWithApple(),
    ).thenThrow(const OAuthCanceledException());
    final c = _container(repo);
    await c.read(authControllerProvider.future);

    await c.read(authControllerProvider.notifier).signInWithApple();

    expect(c.read(authStatusProvider), AuthStatus.unauthenticated);
  });

  test('logout → unauthenticated', () async {
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => _user);
    when(() => repo.logout()).thenAnswer((_) async {});
    final c = _container(repo);
    await c.read(authControllerProvider.future);

    await c.read(authControllerProvider.notifier).logout();

    expect(c.read(authStatusProvider), AuthStatus.unauthenticated);
  });

  test('deleteAccount → unauthenticated', () async {
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => _user);
    when(() => repo.deleteAccount()).thenAnswer((_) async {});
    final c = _container(repo);
    await c.read(authControllerProvider.future);

    await c.read(authControllerProvider.notifier).deleteAccount();

    expect(c.read(authStatusProvider), AuthStatus.unauthenticated);
  });
}
