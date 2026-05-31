import 'dart:convert';

import 'package:agronom_ai/features/chat/data/sse_parser.dart';
import 'package:agronom_ai/features/chat/domain/chat_models.dart';
import 'package:agronom_ai/features/chat/presentation/widgets/fertilizer_card.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Stream<List<int>> _bytes(String s) async* {
  yield utf8.encode(s);
}

Widget _wrap(Widget child) {
  return MaterialApp(
    locale: const Locale('ru'),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(body: child),
  );
}

void main() {
  group('sse_parser fertilizer_card', () {
    test('parses products into SseFertilizerCard', () async {
      const raw =
          'event: fertilizer_card\n'
          'data: {"products":[{"id":"1","slug":"k-boost","name":"Калий-Буст","short_desc":"Подкормка","image_url":"https://img","deeplink_url":"https://shop/k"}]}\n\n';

      final events = await parseSse(_bytes(raw)).toList();

      expect(events, hasLength(1));
      final card = events.first as SseFertilizerCard;
      expect(card.products, hasLength(1));
      final p = card.products.first;
      expect(p.slug, 'k-boost');
      expect(p.name, 'Калий-Буст');
      expect(p.shortDesc, 'Подкормка');
      expect(p.imageUrl, 'https://img');
      expect(p.deeplinkUrl, 'https://shop/k');
    });

    test('empty products yields an empty card list', () async {
      const raw = 'event: fertilizer_card\ndata: {"products":[]}\n\n';
      final events = await parseSse(_bytes(raw)).toList();
      expect(events, hasLength(1));
      expect((events.first as SseFertilizerCard).products, isEmpty);
    });
  });

  group('FertilizerCardList widget', () {
    testWidgets('renders one card per product (vertical stack)', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const FertilizerCardList(
            products: [
              FertilizerProduct(slug: 'a', name: 'Удобрение А', shortDesc: 'Описание А'),
              FertilizerProduct(slug: 'b', name: 'Удобрение Б', shortDesc: 'Описание Б'),
            ],
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Удобрение А'), findsOneWidget);
      expect(find.text('Удобрение Б'), findsOneWidget);
      expect(find.text('Описание А'), findsOneWidget);
      // One "Подробнее" button per product.
      expect(find.text('Подробнее'), findsNWidgets(2));
    });

    testWidgets('tapping Подробнее reports the product via onOpen', (tester) async {
      FertilizerProduct? tapped;
      await tester.pumpWidget(
        _wrap(
          FertilizerCardList(
            // Empty deeplink → onOpen fires, url_launcher is skipped (no platform).
            products: const [
              FertilizerProduct(slug: 'k-boost', name: 'Калий-Буст', shortDesc: 'd'),
            ],
            onOpen: (p) => tapped = p,
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Подробнее'));
      await tester.pumpAndSettle();

      expect(tapped, isNotNull);
      expect(tapped!.slug, 'k-boost');
    });
  });
}
