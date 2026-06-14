import 'package:agronom_ai/features/chat/domain/chat_models.dart';
import 'package:agronom_ai/features/chat/presentation/chat_screen.dart';
import 'package:agronom_ai/features/chat/presentation/message_bubble.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'features/chat/chat_test_helpers.dart';

Widget _wrap(FakeChatRepository repo) => ProviderScope(
  overrides: chatTestOverrides(repo),
  child: const MaterialApp(
    locale: Locale('ru'),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: ChatScreen(),
  ),
);

void main() {
  testWidgets('shows the title, empty hint, and an enabled input', (
    tester,
  ) async {
    await tester.pumpWidget(_wrap(FakeChatRepository()));
    await tester.pumpAndSettle();

    // Header shows the brand title; the empty state shows the greeting.
    expect(find.text('ИИ Агроном'), findsOneWidget);
    expect(find.text('Здравствуйте'), findsOneWidget);
    final field = tester.widget<TextField>(find.byType(TextField));
    expect(field.enabled, isNot(false));
  });

  testWidgets('renders the loaded conversation as bubbles', (tester) async {
    final repo = FakeChatRepository()
      ..fetchError = null
      ..fetchResult = ConversationPage(
        messages: [
          textMessage(id: 'u1', role: MessageRole.user, text: 'привет'),
          textMessage(id: 'a1', role: MessageRole.assistant, text: 'ответ'),
        ],
      );

    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    expect(find.byType(MessageBubble), findsNWidgets(2));
    expect(find.text('привет'), findsOneWidget);
  });

  testWidgets('typing and sending streams the assistant reply', (tester) async {
    final repo = FakeChatRepository()
      ..scriptedStream = () => Stream.fromIterable([
        const SseMessageStarted('s1'),
        const SseDelta('ответ'),
        const SseDone(messageId: 's1', tokensIn: 1, tokensOut: 1),
      ]);

    await tester.pumpWidget(_wrap(repo));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), 'привет');
    await tester.pump(); // let the trailing button switch from mic to send
    await tester.tap(find.byIcon(Icons.arrow_upward_rounded));
    await tester.pumpAndSettle();

    expect(find.text('привет'), findsOneWidget);
    final markdown = tester.widget<MarkdownBody>(find.byType(MarkdownBody));
    expect(markdown.data, 'ответ');
  });
}
