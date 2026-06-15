import 'package:agronom_ai/features/chat/presentation/answer_segments.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('parseAnswerSegments', () {
    test('plain text without an action section stays a single markdown segment', () {
      final segments = parseAnswerSegments(
        'Это просто ответ\n\nБез шагов.',
      );

      expect(segments, hasLength(1));
      expect(segments.single, isA<MarkdownSegment>());
    });

    test('lifts a bold "Что делать" heading with a numbered list into a card', () {
      const text = '''
**Что я вижу** — листья желтеют.

**Возможные причины**
- нехватка магния (высокий)

**Что делать**
1. Опрыскать листья сульфатом магния (1.5%)
2. Полить под корень раствором с магнием
3. Сократить калийные подкормки''';

      final segments = parseAnswerSegments(text);

      final steps = segments.whereType<StepsSegment>().toList();
      expect(steps, hasLength(1));
      expect(steps.single.title, 'Что делать');
      expect(steps.single.steps, hasLength(3));
      expect(steps.single.steps.first, contains('сульфатом магния'));

      // Prose before the steps survives as its own markdown segment, and it
      // still carries "Возможные причины" (not cardified).
      final before = segments.first as MarkdownSegment;
      expect(before.text, contains('Возможные причины'));
      expect(before.text, isNot(contains('Опрыскать')));
    });

    test('detects an ATX heading variant', () {
      const text = '### Что делать сейчас\n1. Первый шаг\n2. Второй шаг';

      final steps = parseAnswerSegments(text).whereType<StepsSegment>().single;

      expect(steps.title, 'Что делать сейчас');
      expect(steps.steps, ['Первый шаг', 'Второй шаг']);
    });

    test('keeps prose after the list as a trailing markdown segment', () {
      const text = '''
**Что делать**
1. Шаг один
2. Шаг два

Я подобрал для вас подходящее удобрение ниже.''';

      final segments = parseAnswerSegments(text);

      expect(segments[0], isA<StepsSegment>());
      expect(segments[1], isA<MarkdownSegment>());
      expect((segments[1] as MarkdownSegment).text, contains('удобрение'));
    });

    test('folds wrapped continuation lines into the step', () {
      const text = '''
**Что делать**
1. Опрыскать листья раствором,
   повторить через 10 дней
2. Сократить полив''';

      final steps = parseAnswerSegments(text).whereType<StepsSegment>().single;

      expect(steps.steps, hasLength(2));
      expect(steps.steps.first, 'Опрыскать листья раствором, повторить через 10 дней');
    });

    test('handles blank lines between list items', () {
      const text = '''
**Что делать**

1. Первый

2. Второй''';

      final steps = parseAnswerSegments(text).whereType<StepsSegment>().single;

      expect(steps.steps, ['Первый', 'Второй']);
    });

    test('does not cardify a non-action heading like "Возможные причины"', () {
      const text = '''
**Возможные причины**
1. Нехватка азота
2. Перелив''';

      final segments = parseAnswerSegments(text);

      expect(segments.whereType<StepsSegment>(), isEmpty);
      expect(segments.single, isA<MarkdownSegment>());
    });

    test('a heading with no following list stays plain markdown', () {
      const text = '**Что делать** дальше — расскажу позже.';

      final segments = parseAnswerSegments(text);

      expect(segments.whereType<StepsSegment>(), isEmpty);
    });

    test('accepts an unordered list under the action heading', () {
      const text = '''
**Что делать**
- Убрать поражённые листья
- Обработать фунгицидом''';

      final steps = parseAnswerSegments(text).whereType<StepsSegment>().single;

      expect(steps.steps, ['Убрать поражённые листья', 'Обработать фунгицидом']);
    });

    test('tolerates a trailing emoji on the bold heading', () {
      const text = '**Что делать** 🌱\n1. Полить\n2. Подкормить';

      final steps = parseAnswerSegments(text).whereType<StepsSegment>().single;

      expect(steps.steps, ['Полить', 'Подкормить']);
    });

    test('splits a list written inline on the heading line', () {
      const text = '**Что делать:** 1. Полить 2. Подкормить 3. Прорыхлить';

      final steps = parseAnswerSegments(text).whereType<StepsSegment>().single;

      expect(steps.title, 'Что делать');
      expect(steps.steps, ['Полить', 'Подкормить', 'Прорыхлить']);
    });

    test('folds nested sub-bullets into their parent step', () {
      const text = '''
**Что делать**
1. Подготовьте раствор:
   - 10 г на 10 л
   - тёплая вода
2. Опрыскайте листья''';

      final steps = parseAnswerSegments(text).whereType<StepsSegment>().single;

      expect(steps.steps, hasLength(2));
      expect(steps.steps.first, contains('10 г на 10 л'));
      expect(steps.steps.first, contains('Подготовьте раствор'));
      expect(steps.steps.last, 'Опрыскайте листья');
    });

    test('a bold note between steps ends the list instead of being swallowed', () {
      const text = '''
**Что делать**
1. Сделайте раствор.
**Важно:** не превышайте дозу.
2. Полейте''';

      final segments = parseAnswerSegments(text);
      final steps = segments.whereType<StepsSegment>().single;

      expect(steps.steps, ['Сделайте раствор.']);
      final after = segments.whereType<MarkdownSegment>().last;
      expect(after.text, contains('Важно'));
    });

    test('a non-indented paragraph between items ends the list', () {
      const text = '''
**Что делать**
1. Полить
Это важно сделать утром.
2. Подкормить''';

      final segments = parseAnswerSegments(text);
      final steps = segments.whereType<StepsSegment>().single;

      expect(steps.steps, ['Полить']);
      expect(
        segments.whereType<MarkdownSegment>().last.text,
        contains('важно сделать утром'),
      );
    });
  });
}
