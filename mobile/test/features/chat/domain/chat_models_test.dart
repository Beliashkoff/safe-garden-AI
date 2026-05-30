import 'package:agronom_ai/features/chat/domain/chat_models.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('ChatMessage.fromJson parses role, status, created_at, content', () {
    final m = ChatMessage.fromJson({
      'id': 'm1',
      'role': 'assistant',
      'status': 'complete',
      'created_at': '2026-05-30T12:34:56.789Z',
      'content': [
        {'type': 'text', 'text': 'Hi'},
      ],
    });

    expect(m.id, 'm1');
    expect(m.role, MessageRole.assistant);
    expect(m.status, MessageStatus.complete);
    expect(m.createdAt.toUtc().year, 2026);
    expect(m.content.single.text, 'Hi');
    expect(m.streaming, isFalse);
    expect(m.errorCode, isNull);
  });

  test('every wire status maps to an enum', () {
    ChatMessage parseStatus(String s) => ChatMessage.fromJson({
      'id': 'x',
      'role': 'assistant',
      'status': s,
      'created_at': '2026-01-01T00:00:00Z',
      'content': const <Object>[],
    });

    expect(parseStatus('pending').status, MessageStatus.pending);
    expect(parseStatus('complete').status, MessageStatus.complete);
    expect(parseStatus('cancelled').status, MessageStatus.cancelled);
    expect(parseStatus('failed').status, MessageStatus.failed);
  });

  test('ConversationPage.fromJson reads messages and next_cursor', () {
    final page = ConversationPage.fromJson({
      'messages': [
        {
          'id': 'a',
          'role': 'user',
          'status': 'complete',
          'created_at': '2026-01-01T00:00:00Z',
          'content': const <Object>[],
        },
      ],
      'next_cursor': 'CURSOR',
    });

    expect(page.messages, hasLength(1));
    expect(page.messages.single.role, MessageRole.user);
    expect(page.nextCursor, 'CURSOR');
  });

  test('ConversationPage.fromJson tolerates a missing next_cursor', () {
    final page = ConversationPage.fromJson({'messages': const <Object>[]});

    expect(page.messages, isEmpty);
    expect(page.nextCursor, isNull);
  });
}
