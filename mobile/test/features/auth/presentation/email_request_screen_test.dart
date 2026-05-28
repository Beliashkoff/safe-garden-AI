import 'package:agronom_ai/features/auth/presentation/email_request_screen.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:mocktail/mocktail.dart';

import '../../../helpers/fakes.dart';

Widget _wrap(MockAuthRepository repo) {
  final router = GoRouter(
    initialLocation: '/login/email',
    routes: [
      GoRoute(
        path: '/login/email',
        builder: (_, _) => const EmailRequestScreen(),
      ),
      GoRoute(
        path: '/login/email/verify',
        builder: (_, _) =>
            const Scaffold(body: Center(child: Text('VERIFY_STUB'))),
      ),
    ],
  );
  return ProviderScope(
    overrides: authTestOverrides(repo),
    child: MaterialApp.router(
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

  testWidgets('rejects an invalid email', (tester) async {
    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextFormField), 'not-an-email');
    await tester.tap(find.text('Получить код'));
    await tester.pumpAndSettle();

    expect(find.text('Некорректный email'), findsOneWidget);
    verifyNever(() => repo.requestEmailCode(any()));
  });

  testWidgets('requests a code and navigates to verify', (tester) async {
    when(() => repo.requestEmailCode(any())).thenAnswer((_) async {});

    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextFormField), 'user@example.com');
    await tester.tap(find.text('Получить код'));
    await tester.pumpAndSettle();

    verify(() => repo.requestEmailCode('user@example.com')).called(1);
    expect(find.text('VERIFY_STUB'), findsOneWidget);
  });
}
