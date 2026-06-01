import 'package:agronom_ai/features/onboarding/data/onboarding_store.dart';
import 'package:agronom_ai/features/onboarding/presentation/onboarding_screen.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'helpers/fakes.dart';

Widget _wrap(FakeOnboardingStore store) {
  return ProviderScope(
    overrides: [onboardingStoreProvider.overrideWithValue(store)],
    child: const MaterialApp(
      locale: Locale('ru'),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: OnboardingScreen(),
    ),
  );
}

void main() {
  testWidgets('advances through pages and completes on the last', (
    tester,
  ) async {
    final store = FakeOnboardingStore(seen: false);
    await tester.pumpWidget(_wrap(store));
    await tester.pumpAndSettle();

    expect(find.text('Сфотографируйте растение'), findsOneWidget);

    await tester.tap(find.text('Далее'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Далее'));
    await tester.pumpAndSettle();

    expect(find.text('Начать'), findsOneWidget);
    await tester.tap(find.text('Начать'));
    await tester.pumpAndSettle();

    expect(store.seen, isTrue);
  });

  testWidgets('skip marks onboarding seen immediately', (tester) async {
    final store = FakeOnboardingStore(seen: false);
    await tester.pumpWidget(_wrap(store));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Пропустить'));
    await tester.pumpAndSettle();

    expect(store.seen, isTrue);
  });
}
