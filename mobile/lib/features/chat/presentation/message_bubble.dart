import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';

import '../../../l10n/generated/app_localizations.dart';
import '../domain/chat_models.dart';
import 'chat_error_message.dart';
import 'widgets/message_photos.dart';

/// A single chat bubble. User messages render photos (if any) and plain
/// selectable text on the right; assistant messages render markdown on the
/// left, with status notes (streaming spinner, cancelled, failed + retry).
class MessageBubble extends StatelessWidget {
  const MessageBubble({required this.message, this.onRetry, super.key});

  final ChatMessage message;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final isUser = message.role == MessageRole.user;

    final background = isUser
        ? theme.colorScheme.primaryContainer
        : theme.colorScheme.surfaceContainerHighest;
    final foreground = isUser
        ? theme.colorScheme.onPrimaryContainer
        : theme.colorScheme.onSurface;

    return Align(
      alignment: isUser ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        constraints: BoxConstraints(
          maxWidth: MediaQuery.of(context).size.width * 0.78,
        ),
        margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          color: background,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: _content(theme, l10n, isUser, foreground),
        ),
      ),
    );
  }

  List<Widget> _content(
    ThemeData theme,
    AppLocalizations l10n,
    bool isUser,
    Color foreground,
  ) {
    final children = <Widget>[];
    final hasText = message.content.any(
      (b) => b.type == 'text' && b.text.isNotEmpty,
    );

    // Render blocks in order, grouping consecutive image blocks into one grid.
    final imageRun = <String>[];
    void flushImages() {
      if (imageRun.isEmpty) {
        return;
      }
      children.add(
        Padding(
          padding: const EdgeInsets.only(bottom: 6),
          child: MessagePhotos(storageKeys: List.of(imageRun)),
        ),
      );
      imageRun.clear();
    }

    for (final block in message.content) {
      if (block.type == 'image') {
        if (block.storageKey.isNotEmpty) {
          imageRun.add(block.storageKey);
        }
        continue;
      }
      if (block.type == 'text' && block.text.isNotEmpty) {
        flushImages();
        children.add(
          isUser
              ? SelectableText(
                  block.text,
                  style: theme.textTheme.bodyLarge?.copyWith(color: foreground),
                )
              : MarkdownBody(
                  data: block.text,
                  selectable: true,
                  styleSheet: MarkdownStyleSheet.fromTheme(theme).copyWith(
                    p: theme.textTheme.bodyLarge?.copyWith(color: foreground),
                  ),
                ),
        );
      }
    }
    flushImages();

    if (message.streaming && !hasText) {
      children.add(
        const Padding(
          padding: EdgeInsets.symmetric(vertical: 4),
          child: SizedBox(
            height: 16,
            width: 16,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
        ),
      );
    }

    if (message.status == MessageStatus.cancelled) {
      children.add(_note(theme, l10n.chatCancelledNote));
    } else if (message.status == MessageStatus.failed) {
      children.add(
        _note(
          theme,
          chatErrorMessageForCode(l10n, message.errorCode ?? 'internal_error'),
        ),
      );
      if (onRetry != null) {
        children.add(
          Align(
            alignment: Alignment.centerLeft,
            child: TextButton(onPressed: onRetry, child: Text(l10n.chatRetry)),
          ),
        );
      }
    }

    return children;
  }

  Widget _note(ThemeData theme, String text) {
    return Padding(
      padding: const EdgeInsets.only(top: 4),
      child: Text(
        text,
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.error,
        ),
      ),
    );
  }
}
