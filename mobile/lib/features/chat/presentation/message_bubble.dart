import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';

import '../../../l10n/generated/app_localizations.dart';
import '../domain/chat_models.dart';
import 'chat_error_message.dart';

/// A single chat bubble. User messages render as plain selectable text on the
/// right; assistant messages render markdown on the left, with status notes
/// (streaming spinner, cancelled, failed + retry).
class MessageBubble extends StatelessWidget {
  const MessageBubble({required this.message, this.onRetry, super.key});

  final ChatMessage message;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final isUser = message.role == MessageRole.user;
    final text = message.content.isNotEmpty ? message.content.first.text : '';

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
          children: _content(theme, l10n, isUser, text, foreground),
        ),
      ),
    );
  }

  List<Widget> _content(
    ThemeData theme,
    AppLocalizations l10n,
    bool isUser,
    String text,
    Color foreground,
  ) {
    final children = <Widget>[];

    if (isUser) {
      children.add(
        SelectableText(
          text,
          style: theme.textTheme.bodyLarge?.copyWith(color: foreground),
        ),
      );
    } else if (text.isNotEmpty) {
      children.add(
        MarkdownBody(
          data: text,
          selectable: true,
          styleSheet: MarkdownStyleSheet.fromTheme(
            theme,
          ).copyWith(p: theme.textTheme.bodyLarge?.copyWith(color: foreground)),
        ),
      );
    }

    if (message.streaming && text.isEmpty) {
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
