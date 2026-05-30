import 'dart:convert';

import 'package:agronom_ai/features/chat/data/sse_parser.dart';
import 'package:agronom_ai/features/chat/domain/chat_models.dart';
import 'package:flutter_test/flutter_test.dart';

Stream<List<int>> _bytes(List<String> chunks) =>
    Stream.fromIterable(chunks.map(utf8.encode));

void main() {
  test('parses a full message_started/delta/done sequence', () async {
    const raw =
        'event: message_started\ndata: {"message_id":"m1"}\n\n'
        'event: delta\ndata: {"text":"Hello "}\n\n'
        'event: delta\ndata: {"text":"world"}\n\n'
        'event: done\ndata: {"message_id":"m1","tokens_used":{"in":10,"out":20}}\n\n';

    final events = await parseSse(_bytes([raw])).toList();

    expect(events, hasLength(4));
    expect((events[0] as SseMessageStarted).messageId, 'm1');
    expect((events[1] as SseDelta).text, 'Hello ');
    expect((events[2] as SseDelta).text, 'world');
    final done = events[3] as SseDone;
    expect(done.messageId, 'm1');
    expect(done.tokensIn, 10);
    expect(done.tokensOut, 20);
  });

  test('reassembles an event split across byte chunks', () async {
    final events = await parseSse(
      _bytes(['event: del', 'ta\ndata: {"te', 'xt":"Hi"}\n', '\n']),
    ).toList();

    expect(events, hasLength(1));
    expect((events.single as SseDelta).text, 'Hi');
  });

  test('maps an error event', () async {
    final events = await parseSse(
      _bytes([
        'event: error\ndata: {"code":"upstream_error","message":"down"}\n\n',
      ]),
    ).toList();

    final err = events.single as SseError;
    expect(err.code, 'upstream_error');
    expect(err.message, 'down');
  });

  test('skips unknown events and malformed data', () async {
    final events = await parseSse(
      _bytes([
        'event: ping\ndata: {}\n\n'
            'event: delta\ndata: not-json\n\n'
            'event: delta\ndata: {"text":"ok"}\n\n',
      ]),
    ).toList();

    expect(events, hasLength(1));
    expect((events.single as SseDelta).text, 'ok');
  });

  test('decodes multibyte characters split across chunks', () async {
    final full = utf8.encode('event: delta\ndata: {"text":"Привет"}\n\n');
    final mid = full.length ~/ 2;
    final events = await parseSse(
      Stream.fromIterable([full.sublist(0, mid), full.sublist(mid)]),
    ).toList();

    expect((events.single as SseDelta).text, 'Привет');
  });
}
