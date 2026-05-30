import 'package:agronom_ai/features/chat/domain/chat_models.dart';
import 'package:agronom_ai/features/chat/presentation/message_bubble.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Widget _wrap(Widget child) => MaterialApp(
  locale: const Locale('ru'),
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);

ChatMessage _message({
  required MessageRole role,
  required MessageStatus status,
  String text = '',
  bool streaming = false,
  String? errorCode,
}) {
  return ChatMessage(
    id: 'id',
    role: role,
    status: status,
    createdAt: DateTime.utc(2026, 1, 1),
    content: text.isEmpty ? const [] : [ContentBlock(type: 'text', text: text)],
    streaming: streaming,
    errorCode: errorCode,
  );
}

void main() {
  testWidgets('user message renders as selectable text', (tester) async {
    await tester.pumpWidget(
      _wrap(
        MessageBubble(
          message: _message(
            role: MessageRole.user,
            status: MessageStatus.complete,
            text: 'Привет',
          ),
        ),
      ),
    );

    expect(find.text('Привет'), findsOneWidget);
  });

  testWidgets('streaming assistant with no text shows a spinner', (
    tester,
  ) async {
    await tester.pumpWidget(
      _wrap(
        MessageBubble(
          message: _message(
            role: MessageRole.assistant,
            status: MessageStatus.pending,
            streaming: true,
          ),
        ),
      ),
    );

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });

  testWidgets('failed assistant shows the error note and a retry button', (
    tester,
  ) async {
    var retried = false;
    await tester.pumpWidget(
      _wrap(
        MessageBubble(
          message: _message(
            role: MessageRole.assistant,
            status: MessageStatus.failed,
            errorCode: 'rate_limited',
          ),
          onRetry: () => retried = true,
        ),
      ),
    );

    expect(find.text('Повторить'), findsOneWidget);
    await tester.tap(find.text('Повторить'));
    expect(retried, isTrue);
  });

  testWidgets('cancelled assistant shows the cancelled note', (tester) async {
    await tester.pumpWidget(
      _wrap(
        MessageBubble(
          message: _message(
            role: MessageRole.assistant,
            status: MessageStatus.cancelled,
            text: 'partial',
          ),
        ),
      ),
    );

    expect(find.text('Ответ остановлен'), findsOneWidget);
  });
}
