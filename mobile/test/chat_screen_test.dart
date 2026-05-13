import 'package:agronom_ai/features/chat/presentation/chat_screen.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Widget _wrap(Widget child) {
  return MaterialApp(
    locale: const Locale('ru'),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: child,
  );
}

void main() {
  testWidgets('shows AppBar title, empty hint, and disabled input',
      (tester) async {
    await tester.pumpWidget(_wrap(const ChatScreen()));
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
