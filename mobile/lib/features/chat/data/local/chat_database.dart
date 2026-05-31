import 'dart:convert';
import 'dart:io';

import 'package:drift/drift.dart';
import 'package:drift/native.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';

import '../../domain/chat_models.dart';

part 'chat_database.g.dart';

/// Cached messages. Ordered by [createdAtMs] (epoch millis, UTC); [role] and
/// [status] hold the wire strings (MessageRole/MessageStatus `.name`).
class Messages extends Table {
  TextColumn get id => text()();
  TextColumn get role => text()();
  TextColumn get status => text()();
  IntColumn get createdAtMs => integer()();

  @override
  Set<Column<Object>> get primaryKey => {id};
}

/// Cached content blocks. Owned by a message (deleted alongside it in code —
/// foreign-key enforcement is not relied upon).
class MessageBlocks extends Table {
  TextColumn get messageId => text()();
  IntColumn get orderIndex => integer()();
  TextColumn get type => text()();
  TextColumn get content => text()();

  @override
  Set<Column<Object>> get primaryKey => {messageId, orderIndex};
}

/// Local offline cache for the single conversation. Server history is the source
/// of truth; this mirrors it so the chat opens instantly and works offline.
@DriftDatabase(tables: [Messages, MessageBlocks])
class ChatDatabase extends _$ChatDatabase {
  ChatDatabase([QueryExecutor? executor]) : super(executor ?? _open());

  @override
  int get schemaVersion => 1;

  static QueryExecutor _open() {
    return LazyDatabase(() async {
      final dir = await getApplicationSupportDirectory();
      final file = File('${dir.path}/safe_garden_chat.sqlite');
      return NativeDatabase.createInBackground(file);
    });
  }

  /// All cached messages in chronological order, each with its content blocks.
  Future<List<ChatMessage>> loadMessages() async {
    final rows = await (select(
      messages,
    )..orderBy([(m) => OrderingTerm.asc(m.createdAtMs)])).get();
    final blockRows = await select(messageBlocks).get();

    final byMessage = <String, List<MessageBlock>>{};
    for (final b in blockRows) {
      byMessage.putIfAbsent(b.messageId, () => []).add(b);
    }

    return rows.map((r) {
      final blocks = (byMessage[r.id] ?? <MessageBlock>[])
        ..sort((a, b) => a.orderIndex.compareTo(b.orderIndex));
      return ChatMessage(
        id: r.id,
        role: MessageRole.values.byName(r.role),
        status: MessageStatus.values.byName(r.status),
        createdAt: DateTime.fromMillisecondsSinceEpoch(
          r.createdAtMs,
          isUtc: true,
        ),
        content: blocks.map(_decodeBlock).toList(),
      );
    }).toList();
  }

  /// Inserts or replaces [items] and their blocks (transient `streaming` flag is
  /// not persisted).
  Future<void> upsertMessages(List<ChatMessage> items) async {
    await transaction(() async {
      for (final m in items) {
        await into(messages).insertOnConflictUpdate(
          MessagesCompanion.insert(
            id: m.id,
            role: m.role.name,
            status: m.status.name,
            createdAtMs: m.createdAt.toUtc().millisecondsSinceEpoch,
          ),
        );
        await (delete(
          messageBlocks,
        )..where((b) => b.messageId.equals(m.id))).go();
        for (var i = 0; i < m.content.length; i++) {
          await into(messageBlocks).insert(
            MessageBlocksCompanion.insert(
              messageId: m.id,
              orderIndex: i,
              type: m.content[i].type,
              content: _encodeBlock(m.content[i]),
            ),
          );
        }
      }
    });
  }

  /// Removes a message and its blocks from the cache.
  Future<void> deleteMessage(String id) async {
    await transaction(() async {
      await (delete(messageBlocks)..where((b) => b.messageId.equals(id))).go();
      await (delete(messages)..where((m) => m.id.equals(id))).go();
    });
  }

  /// Drops the entire cache (e.g. on logout / account deletion).
  Future<void> clear() async {
    await transaction(() async {
      await delete(messageBlocks).go();
      await delete(messages).go();
    });
  }
}

/// Serialises a content block to the row's `content` column as JSON, so media
/// keys and durations survive offline (a single text column otherwise loses
/// everything but `text`).
String _encodeBlock(ContentBlock b) => jsonEncode({
  'text': b.text,
  'storage_key': b.storageKey,
  'duration_ms': b.durationMs,
});

/// Rebuilds a content block from a cached row. New rows hold a JSON payload;
/// legacy rows held plain text — both are tolerated.
ContentBlock _decodeBlock(MessageBlock b) {
  try {
    final dynamic decoded = jsonDecode(b.content);
    if (decoded is Map) {
      return ContentBlock(
        type: b.type,
        text: (decoded['text'] as String?) ?? '',
        storageKey: (decoded['storage_key'] as String?) ?? '',
        durationMs: (decoded['duration_ms'] as num?)?.toInt() ?? 0,
      );
    }
  } on FormatException {
    // legacy plain-text row — fall through
  }
  return ContentBlock(type: b.type, text: b.content);
}

final chatDatabaseProvider = Provider<ChatDatabase>((ref) {
  final db = ChatDatabase();
  ref.onDispose(db.close);
  return db;
});
