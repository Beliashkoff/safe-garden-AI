import 'package:agronom_ai/features/auth/presentation/login_screen.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

Widget _wrap(Widget child, {TargetPlatform platform = TargetPlatform.android}) {
  final router = GoRouter(
    initialLocation: '/login',
    routes: [
      GoRoute(path: '/login', builder: (context, state) => child),
      GoRoute(
        path: '/chat',
        builder: (context, state) =>
            const Scaffold(body: Center(child: Text('CHAT_STUB'))),
      ),
    ],
  );

  return MaterialApp.router(
    theme: ThemeData(platform: platform, useMaterial3: true),
    locale: const Locale('ru'),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    routerConfig: router,
  );
}

void main() {
  testWidgets('shows title and subtitle', (tester) async {
    await tester.pumpWidget(_wrap(const LoginScreen()));
    await tester.pumpAndSettle();

    expect(find.text('Войти'), findsOneWidget);
    expect(find.text('Диагностика растений с помощью AI'), findsOneWidget);
  });

  testWidgets('Android: shows Google and Email, no Apple button',
      (tester) async {
    await tester.pumpWidget(
      _wrap(const LoginScreen(), platform: TargetPlatform.android),
    );
    await tester.pumpAndSettle();

    expect(find.text('Войти с Google'), findsOneWidget);
    expect(find.text('Войти по email'), findsOneWidget);
    expect(find.text('Войти с Apple'), findsNothing);
  });

  testWidgets('iOS: shows Apple, Google and Email buttons', (tester) async {
    await tester.pumpWidget(
      _wrap(const LoginScreen(), platform: TargetPlatform.iOS),
    );
    await tester.pumpAndSettle();

    expect(find.text('Войти с Apple'), findsOneWidget);
    expect(find.text('Войти с Google'), findsOneWidget);
    expect(find.text('Войти по email'), findsOneWidget);
  });

  testWidgets('Email button navigates to /chat', (tester) async {
    await tester.pumpWidget(_wrap(const LoginScreen()));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Войти по email'));
    await tester.pumpAndSettle();

    expect(find.text('CHAT_STUB'), findsOneWidget);
  });

  testWidgets('Google button shows "Coming soon" snackbar', (tester) async {
    await tester.pumpWidget(_wrap(const LoginScreen()));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Войти с Google'));
    await tester.pump();

    expect(find.text('Скоро будет доступно'), findsOneWidget);
  });
}
