import 'package:agronom_ai/core/network/api_exception.dart';
import 'package:agronom_ai/features/auth/domain/auth_models.dart';
import 'package:agronom_ai/features/auth/presentation/email_verify_screen.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import '../../../helpers/fakes.dart';

Widget _wrap(MockAuthRepository repo) {
  return ProviderScope(
    overrides: authTestOverrides(repo),
    child: const MaterialApp(
      locale: Locale('ru'),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: EmailVerifyScreen(email: 'user@example.com'),
    ),
  );
}

void main() {
  late MockAuthRepository repo;

  setUp(() {
    repo = MockAuthRepository();
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => null);
  });

  testWidgets('shows the destination email', (tester) async {
    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();
    expect(find.textContaining('user@example.com'), findsOneWidget);
  });

  testWidgets('submitting 6 digits verifies the code', (tester) async {
    when(
      () => repo.verifyEmailCode(
        email: any(named: 'email'),
        code: any(named: 'code'),
      ),
    ).thenAnswer((_) async => const AppUser(id: 'u1', emailVerified: true));

    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), '123456');
    await tester.pumpAndSettle();

    verify(
      () => repo.verifyEmailCode(email: 'user@example.com', code: '123456'),
    ).called(1);
  });

  testWidgets('shows error on invalid code', (tester) async {
    when(
      () => repo.verifyEmailCode(
        email: any(named: 'email'),
        code: any(named: 'code'),
      ),
    ).thenThrow(const ApiException(code: 'unauthorized', message: 'bad'));

    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), '000000');
    await tester.pumpAndSettle();

    expect(find.text('Неверный или истёкший код'), findsOneWidget);
  });
}
