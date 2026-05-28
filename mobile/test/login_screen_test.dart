import 'package:agronom_ai/features/auth/presentation/login_screen.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:mocktail/mocktail.dart';

import 'helpers/fakes.dart';

Widget _wrap(
  MockAuthRepository repo, {
  TargetPlatform platform = TargetPlatform.android,
}) {
  final router = GoRouter(
    initialLocation: '/login',
    routes: [
      GoRoute(
        path: '/login',
        builder: (_, _) => const LoginScreen(),
        routes: [
          GoRoute(
            path: 'email',
            builder: (_, _) =>
                const Scaffold(body: Center(child: Text('EMAIL_STUB'))),
          ),
        ],
      ),
    ],
  );

  return ProviderScope(
    overrides: authTestOverrides(repo),
    child: MaterialApp.router(
      theme: ThemeData(platform: platform, useMaterial3: true),
      locale: const Locale('ru'),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      routerConfig: router,
    ),
  );
}

void main() {
  late MockAuthRepository repo;

  setUp(() {
    repo = MockAuthRepository();
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => null);
  });

  testWidgets('shows title and subtitle', (tester) async {
    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    expect(find.text('Войти'), findsOneWidget);
    expect(find.text('Диагностика растений с помощью AI'), findsOneWidget);
  });

  testWidgets('Android: Google and Email, no Apple', (tester) async {
    await tester.pumpWidget(_wrap(repo, platform: TargetPlatform.android));
    await tester.pumpAndSettle();

    expect(find.text('Войти с Google'), findsOneWidget);
    expect(find.text('Войти по email'), findsOneWidget);
    expect(find.text('Войти с Apple'), findsNothing);
  });

  testWidgets('iOS: Apple, Google and Email', (tester) async {
    await tester.pumpWidget(_wrap(repo, platform: TargetPlatform.iOS));
    await tester.pumpAndSettle();

    expect(find.text('Войти с Apple'), findsOneWidget);
    expect(find.text('Войти с Google'), findsOneWidget);
    expect(find.text('Войти по email'), findsOneWidget);
  });

  testWidgets('Email button navigates to the email screen', (tester) async {
    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Войти по email'));
    await tester.pumpAndSettle();

    expect(find.text('EMAIL_STUB'), findsOneWidget);
  });
}
