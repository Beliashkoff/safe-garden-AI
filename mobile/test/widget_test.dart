import 'package:agronom_ai/app/app.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AgronomApp renders and lands on login screen', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(child: AgronomApp()),
    );
    await tester.pumpAndSettle();

    expect(find.text('Войти'), findsOneWidget);
    expect(find.text('Диагностика растений с помощью AI'), findsOneWidget);
  });
}
