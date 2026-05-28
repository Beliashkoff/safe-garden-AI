import 'package:agronom_ai/app/app.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'helpers/fakes.dart';

void main() {
  testWidgets('boots to the login screen when no session is stored', (
    tester,
  ) async {
    final repo = MockAuthRepository();
    when(() => repo.tryRestoreSession()).thenAnswer((_) async => null);

    await tester.pumpWidget(
      ProviderScope(
        overrides: authTestOverrides(repo),
        child: const AgronomApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Войти'), findsOneWidget);
    expect(find.text('Диагностика растений с помощью AI'), findsOneWidget);
  });
}
