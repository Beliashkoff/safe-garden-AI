import 'dart:io';

import 'package:agronom_ai/features/chat/data/audio_ports.dart';
import 'package:agronom_ai/features/chat/data/media_cache.dart';
import 'package:agronom_ai/features/chat/domain/chat_models.dart';
import 'package:agronom_ai/features/chat/presentation/message_bubble.dart';
import 'package:agronom_ai/features/chat/presentation/widgets/voice_message_player.dart';
import 'package:agronom_ai/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakePlayer implements AudioPlayerPort {
  @override
  Future<Duration?> setFilePath(String path) async => const Duration(seconds: 4);
  @override
  Future<void> play() async {}
  @override
  Future<void> pause() async {}
  @override
  Future<void> stop() async {}
  @override
  Future<void> seek(Duration position) async {}
  @override
  Stream<Duration> position() => const Stream.empty();
  @override
  Stream<bool> playing() => const Stream.empty();
  @override
  Duration? get duration => const Duration(seconds: 4);
  @override
  Future<void> dispose() async {}
}

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

  testWidgets('a transcription block renders its text', (tester) async {
    await tester.pumpWidget(
      _wrap(
        MessageBubble(
          message: ChatMessage(
            id: 'v',
            role: MessageRole.user,
            status: MessageStatus.complete,
            createdAt: DateTime.utc(2026, 1, 1),
            content: const [
              ContentBlock(
                type: 'transcription',
                text: 'вянут помидоры',
                durationMs: 4200,
              ),
            ],
          ),
        ),
      ),
    );

    expect(find.text('вянут помидоры'), findsOneWidget);
  });

  testWidgets('a voice message renders a player and the transcription', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          audioPlayerFactoryProvider.overrideWithValue(_FakePlayer.new),
          mediaFileProvider('u/a/audio/x.m4a').overrideWith(
            (ref) => File('${Directory.systemTemp.path}/test_voice.m4a'),
          ),
        ],
        child: _wrap(
          MessageBubble(
            message: ChatMessage(
              id: 'v',
              role: MessageRole.user,
              status: MessageStatus.complete,
              createdAt: DateTime.utc(2026, 1, 1),
              content: const [
                ContentBlock(
                  type: 'audio',
                  storageKey: 'u/a/audio/x.m4a',
                  durationMs: 4200,
                ),
                ContentBlock(
                  type: 'transcription',
                  text: 'вянут помидоры',
                  durationMs: 4200,
                ),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byType(VoiceMessagePlayer), findsOneWidget);
    expect(find.text('вянут помидоры'), findsOneWidget);
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
