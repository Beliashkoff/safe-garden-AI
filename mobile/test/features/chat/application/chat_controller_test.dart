import 'package:agronom_ai/core/network/api_exception.dart';
import 'package:agronom_ai/features/chat/application/chat_controller.dart';
import 'package:agronom_ai/features/chat/domain/chat_models.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../chat_test_helpers.dart';

Future<ChatController> _boot(ProviderContainer container) async {
  await container.read(chatControllerProvider.future);
  return container.read(chatControllerProvider.notifier);
}

ChatState _state(ProviderContainer container) =>
    container.read(chatControllerProvider).requireValue;

void main() {
  test('streams deltas into the assistant bubble and completes', () async {
    final repo = FakeChatRepository()
      ..scriptedStream = () => Stream.fromIterable([
        const SseMessageStarted('s1'),
        const SseDelta('Hello '),
        const SseDelta('world'),
        const SseDone(messageId: 's1', tokensIn: 1, tokensOut: 2),
      ]);
    final container = ProviderContainer(overrides: chatTestOverrides(repo));
    addTearDown(container.dispose);
    final notifier = await _boot(container);

    await notifier.sendMessage('hi');
    await pumpEventQueue();

    final state = _state(container);
    expect(state.sending, isFalse);
    expect(state.messages, hasLength(2));
    expect(state.messages[0].role, MessageRole.user);
    expect(state.messages[0].content.single.text, 'hi');
    final assistant = state.messages[1];
    expect(assistant.role, MessageRole.assistant);
    expect(assistant.status, MessageStatus.complete);
    expect(assistant.streaming, isFalse);
    expect(assistant.content.single.text, 'Hello world');
  });

  test(
    'cancellation marks the assistant cancelled with partial text',
    () async {
      final repo = FakeChatRepository()
        ..scriptedStream = () async* {
          yield const SseMessageStarted('s1');
          yield const SseDelta('partial');
          throw DioException(
            requestOptions: RequestOptions(path: '/messages'),
            type: DioExceptionType.cancel,
          );
        };
      final container = ProviderContainer(overrides: chatTestOverrides(repo));
      addTearDown(container.dispose);
      final notifier = await _boot(container);

      await notifier.sendMessage('hi');
      await pumpEventQueue();

      final assistant = _state(container).messages.last;
      expect(assistant.status, MessageStatus.cancelled);
      expect(assistant.streaming, isFalse);
      expect(assistant.content.single.text, 'partial');
    },
  );

  test(
    'a pre-stream rejection marks the assistant failed with its code',
    () async {
      final repo = FakeChatRepository()
        ..scriptedStream = () async* {
          throw const ApiException(code: 'rate_limited', message: 'slow');
        };
      final container = ProviderContainer(overrides: chatTestOverrides(repo));
      addTearDown(container.dispose);
      final notifier = await _boot(container);

      await notifier.sendMessage('hi');
      await pumpEventQueue();

      final assistant = _state(container).messages.last;
      expect(assistant.status, MessageStatus.failed);
      expect(assistant.errorCode, 'rate_limited');
      expect(_state(container).sending, isFalse);
    },
  );

  test('an error event marks the assistant failed', () async {
    final repo = FakeChatRepository()
      ..scriptedStream = () => Stream.fromIterable([
        const SseMessageStarted('s1'),
        const SseError(code: 'upstream_error', message: 'down'),
      ]);
    final container = ProviderContainer(overrides: chatTestOverrides(repo));
    addTearDown(container.dispose);
    final notifier = await _boot(container);

    await notifier.sendMessage('hi');
    await pumpEventQueue();

    final assistant = _state(container).messages.last;
    expect(assistant.status, MessageStatus.failed);
    expect(assistant.errorCode, 'upstream_error');
  });

  test('deleteMessage removes the message and calls the repository', () async {
    final repo = FakeChatRepository()
      ..fetchError = null
      ..fetchResult = ConversationPage(
        messages: [
          textMessage(id: 'u1', role: MessageRole.user),
          textMessage(id: 'a1', role: MessageRole.assistant),
        ],
      );
    final container = ProviderContainer(overrides: chatTestOverrides(repo));
    addTearDown(container.dispose);
    final notifier = await _boot(container);

    await notifier.deleteMessage('a1');

    expect(repo.deleted, contains('a1'));
    expect(_state(container).messages.map((m) => m.id), isNot(contains('a1')));
  });

  test('sendMessage with photos adds image blocks and forwards keys', () async {
    final repo = FakeChatRepository()
      ..scriptedStream = () => Stream.fromIterable([
        const SseMessageStarted('s1'),
        const SseDelta('ok'),
        const SseDone(messageId: 's1', tokensIn: 1, tokensOut: 1),
      ]);
    final container = ProviderContainer(overrides: chatTestOverrides(repo));
    addTearDown(container.dispose);
    final notifier = await _boot(container);

    await notifier.sendMessage(
      'look',
      imageStorageKeys: const ['u/a/img/1.jpg', 'u/a/img/2.jpg'],
    );
    await pumpEventQueue();

    expect(repo.lastSendText, 'look');
    expect(repo.lastSendImageKeys, ['u/a/img/1.jpg', 'u/a/img/2.jpg']);
    final user = _state(container).messages.first;
    expect(user.role, MessageRole.user);
    expect(user.content.where((b) => b.type == 'text').single.text, 'look');
    expect(
      user.content.where((b) => b.type == 'image').map((b) => b.storageKey),
      ['u/a/img/1.jpg', 'u/a/img/2.jpg'],
    );
  });

  test('a photo-only message (no text) is allowed', () async {
    final repo = FakeChatRepository()
      ..scriptedStream = () => Stream.fromIterable([
        const SseMessageStarted('s1'),
        const SseDone(messageId: 's1', tokensIn: 0, tokensOut: 0),
      ]);
    final container = ProviderContainer(overrides: chatTestOverrides(repo));
    addTearDown(container.dispose);
    final notifier = await _boot(container);

    await notifier.sendMessage('', imageStorageKeys: const ['u/a/img/1.jpg']);
    await pumpEventQueue();

    expect(repo.lastSendImageKeys, ['u/a/img/1.jpg']);
    final user = _state(container).messages.first;
    expect(user.content, hasLength(1));
    expect(user.content.single.type, 'image');
    expect(_state(container).messages, hasLength(2));
  });

  test('retry resends the text and photos of the last user message', () async {
    final repo = FakeChatRepository()
      ..scriptedStream = () async* {
        throw const ApiException(code: 'network', message: 'x');
      };
    final container = ProviderContainer(overrides: chatTestOverrides(repo));
    addTearDown(container.dispose);
    final notifier = await _boot(container);

    await notifier.sendMessage('look', imageStorageKeys: const ['u/a/img/1.jpg']);
    await pumpEventQueue();
    expect(_state(container).messages.last.status, MessageStatus.failed);

    repo.scriptedStream = () => Stream.fromIterable([
      const SseMessageStarted('s2'),
      const SseDone(messageId: 's2', tokensIn: 0, tokensOut: 0),
    ]);
    await notifier.retry();
    await pumpEventQueue();

    expect(repo.lastSendText, 'look');
    expect(repo.lastSendImageKeys, ['u/a/img/1.jpg']);
  });
}
