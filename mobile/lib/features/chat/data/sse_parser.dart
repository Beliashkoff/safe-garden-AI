import 'dart:convert';

import '../domain/chat_models.dart';

/// Frames a raw SSE byte stream into typed [SseEvent]s.
///
/// The backend writes each event as `event: <name>\ndata: <json>\n\n`. The byte
/// stream is decoded as UTF-8 and split into lines (both transforms buffer
/// across chunk boundaries), `event:`/`data:` lines are accumulated, and on a
/// blank line the buffered event is dispatched. Unknown event names and
/// malformed data are skipped rather than throwing, so one bad event never
/// kills the stream.
Stream<SseEvent> parseSse(Stream<List<int>> bytes) async* {
  final lines = bytes.transform(utf8.decoder).transform(const LineSplitter());
  String? eventName;
  final dataBuffer = StringBuffer();

  await for (final line in lines) {
    if (line.isEmpty) {
      final event = _dispatch(eventName, dataBuffer.toString());
      eventName = null;
      dataBuffer.clear();
      if (event != null) {
        yield event;
      }
      continue;
    }
    if (line.startsWith('event:')) {
      eventName = line.substring(6).trim();
    } else if (line.startsWith('data:')) {
      // Per the SSE spec multiple data: lines join with '\n'. The backend emits
      // a single data line, but handle the general case.
      if (dataBuffer.isNotEmpty) {
        dataBuffer.write('\n');
      }
      dataBuffer.write(line.substring(5).trimLeft());
    }
    // Other fields (id:, retry:, ':' comments) are ignored.
  }

  // Flush a trailing event that lacked a final blank line (defensive).
  final tail = _dispatch(eventName, dataBuffer.toString());
  if (tail != null) {
    yield tail;
  }
}

SseEvent? _dispatch(String? eventName, String data) {
  if (eventName == null || eventName.isEmpty) {
    return null;
  }
  Map<String, dynamic> json;
  try {
    final dynamic decoded = data.isEmpty
        ? const <String, dynamic>{}
        : jsonDecode(data);
    json = decoded is Map ? decoded.cast<String, dynamic>() : const {};
  } on FormatException {
    return null;
  }

  switch (eventName) {
    case 'message_started':
      return SseMessageStarted((json['message_id'] as String?) ?? '');
    case 'delta':
      return SseDelta((json['text'] as String?) ?? '');
    case 'tool_use':
      return SseToolUse(
        tool: (json['tool'] as String?) ?? '',
        args: (json['args'] as Map?)?.cast<String, dynamic>() ?? const {},
      );
    case 'fertilizer_card':
      return SseFertilizerCard(json);
    case 'error':
      return SseError(
        code: (json['code'] as String?) ?? 'upstream_error',
        message: (json['message'] as String?) ?? 'the assistant failed',
      );
    case 'done':
      final tokens = (json['tokens_used'] as Map?)?.cast<String, dynamic>();
      return SseDone(
        messageId: (json['message_id'] as String?) ?? '',
        tokensIn: (tokens?['in'] as num?)?.toInt() ?? 0,
        tokensOut: (tokens?['out'] as num?)?.toInt() ?? 0,
      );
    default:
      return null;
  }
}
