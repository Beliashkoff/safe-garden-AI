import 'package:agronom_ai/features/chat/presentation/chat_screen.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'helpers/fakes.dart';

Widget _wrap(MockAuthRepository repo) {
  return ProviderScope(
    overrides: authTestOverrides(repo),
    child: const MaterialApp(
      locale: Locale('ru'),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: ChatScreen(),
    ),
  );
}

void main() {
  testWidgets('shows AppBar title, empty hint, and disabled input', (
    tester,
  ) async {
    final repo = MockAuthRepository();
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => null);

    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    expect(find.text('Чат'), findsOneWidget);
    expect(
      find.text('Сфотографируйте растение или опишите проблему'),
      findsOneWidget,
    );

    final textField = tester.widget<TextField>(find.byType(TextField));
    expect(textField.enabled, isFalse);
  });
}
