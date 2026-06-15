import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:share_plus/share_plus.dart';

import '../../../app/theme.dart';
import '../../../app/widgets/brand_logo.dart';
import '../../../l10n/generated/app_localizations.dart';
import '../domain/chat_models.dart';
import 'chat_error_message.dart';
import 'widgets/fertilizer_card.dart';
import 'widgets/message_photos.dart';
import 'widgets/voice_message_player.dart';

/// A single chat turn. User turns render photos (rounded, with an "analysed"
/// badge once answered) and a soft right-aligned bubble; assistant turns render
/// bubble-less markdown on the left under a brand author row, with an action
/// footer (copy / feedback / regenerate / share) on the finished answer.
class MessageBubble extends StatelessWidget {
  const MessageBubble({
    required this.message,
    this.photoAnalysed = false,
    this.reviewedPhoto = false,
    this.showFooter = false,
    this.feedbackValue,
    this.onFeedback,
    this.onRegenerate,
    this.onRetry,
    this.onFertilizerTap,
    super.key,
  });

  final ChatMessage message;
  final bool photoAnalysed;
  final bool reviewedPhoto;
  final bool showFooter;
  final String? feedbackValue;
  final void Function(String? value)? onFeedback;
  final VoidCallback? onRegenerate;
  final VoidCallback? onRetry;
  final void Function(FertilizerProduct product)? onFertilizerTap;

  bool get _isUser => message.role == MessageRole.user;

  /// The plain-text body of the message (for copy / share).
  String get _plainText => message.content
      .where((b) => (b.type == 'text' || b.type == 'transcription'))
      .map((b) => b.text)
      .where((t) => t.isNotEmpty)
      .join('\n\n');

  @override
  Widget build(BuildContext context) {
    return _isUser ? _userTurn(context) : _assistantTurn(context);
  }

  // ── user ──────────────────────────────────────────────────────────────
  Widget _userTurn(BuildContext context) {
    final theme = Theme.of(context);
    final p = theme.palette;
    final children = <Widget>[];

    final imageKeys = [
      for (final b in message.content)
        if (b.type == 'image' && b.storageKey.isNotEmpty) b.storageKey,
    ];
    if (imageKeys.isNotEmpty) {
      children.add(
        MessagePhotos(storageKeys: imageKeys, analysed: photoAnalysed),
      );
    }

    for (final b in message.content) {
      if (b.type == 'audio' && b.storageKey.isNotEmpty) {
        children.add(
          _bubble(
            p,
            VoiceMessagePlayer(
              storageKey: b.storageKey,
              hintDurationMs: b.durationMs,
            ),
          ),
        );
      } else if (b.type == 'transcription' && b.text.isNotEmpty) {
        children.add(
          _bubble(
            p,
            Text(
              b.text,
              style: theme.textTheme.bodyLarge?.copyWith(
                fontStyle: FontStyle.italic,
              ),
            ),
          ),
        );
      } else if (b.type == 'text' && b.text.isNotEmpty) {
        children.add(
          _bubble(p, SelectableText(b.text, style: theme.textTheme.bodyLarge)),
        );
      }
    }

    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 6, 14, 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          for (final c in children) ...[
            ConstrainedBox(
              constraints: BoxConstraints(
                maxWidth: MediaQuery.of(context).size.width * 0.78,
              ),
              child: c,
            ),
            const SizedBox(height: 6),
          ],
        ],
      ),
    );
  }

  Widget _bubble(AppPalette p, Widget child) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: p.bubble,
        borderRadius: BorderRadius.circular(AppRadius.bubble),
      ),
      child: child,
    );
  }

  // ── assistant ─────────────────────────────────────────────────────────
  Widget _assistantTurn(BuildContext context) {
    final theme = Theme.of(context);
    final p = theme.palette;
    final l10n = AppLocalizations.of(context)!;
    final hasText = message.content.any(
      (b) => b.type == 'text' && b.text.isNotEmpty,
    );

    final body = <Widget>[
      _authorRow(theme, p, l10n),
      const SizedBox(height: 10),
    ];

    for (final b in message.content) {
      if (b.type == 'text' && b.text.isNotEmpty) {
        body.add(
          MarkdownBody(
            data: b.text,
            selectable: true,
            styleSheet: MarkdownStyleSheet.fromTheme(theme).copyWith(
              p: theme.textTheme.bodyLarge,
              listBullet: theme.textTheme.bodyLarge,
            ),
          ),
        );
      } else if (b.type == 'fertilizer_card' && b.products.isNotEmpty) {
        body.add(
          Padding(
            padding: const EdgeInsets.only(top: 10),
            child: FertilizerCardList(
              products: b.products,
              onOpen: onFertilizerTap,
            ),
          ),
        );
      }
    }

    if (message.streaming && !hasText) {
      body.add(_TypingDots(key: const Key('ai-typing'), color: p.textSubtle));
    }

    if (message.status == MessageStatus.cancelled) {
      body.add(_note(theme, l10n.chatCancelledNote, p.textMuted));
    } else if (message.status == MessageStatus.failed) {
      body.add(
        _note(
          theme,
          chatErrorMessageForCode(l10n, message.errorCode ?? 'internal_error'),
          p.danger,
        ),
      );
      if (onRetry != null) {
        body.add(
          Align(
            alignment: Alignment.centerLeft,
            child: TextButton(onPressed: onRetry, child: Text(l10n.chatRetry)),
          ),
        );
      }
    }

    if (showFooter) {
      body.add(_footer(context, theme, p, l10n));
    }

    return Padding(
      padding: const EdgeInsets.fromLTRB(18, 14, 18, 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: body,
      ),
    );
  }

  Widget _authorRow(ThemeData theme, AppPalette p, AppLocalizations l10n) {
    return Row(
      children: [
        const BrandLogo(size: 22),
        const SizedBox(width: 7),
        Text(
          l10n.appTitle,
          style: theme.textTheme.bodySmall?.copyWith(
            color: p.textMuted,
            fontWeight: FontWeight.w500,
          ),
        ),
        if (reviewedPhoto) ...[
          const SizedBox(width: 6),
          Text(
            '· ${l10n.aiReviewedPhoto}',
            style: theme.textTheme.bodySmall?.copyWith(color: p.textSubtle),
          ),
        ],
      ],
    );
  }

  Widget _footer(
    BuildContext context,
    ThemeData theme,
    AppPalette p,
    AppLocalizations l10n,
  ) {
    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Row(
        children: [
          _FooterButton(
            icon: Icons.copy_rounded,
            tooltip: l10n.actionCopy,
            color: p.textMuted,
            onTap: () => _copy(context, l10n),
          ),
          _FooterButton(
            icon: feedbackValue == 'up'
                ? Icons.thumb_up
                : Icons.thumb_up_outlined,
            tooltip: l10n.actionGoodAnswer,
            color: feedbackValue == 'up' ? p.brandGreen : p.textMuted,
            onTap: () => onFeedback?.call(feedbackValue == 'up' ? null : 'up'),
          ),
          _FooterButton(
            icon: feedbackValue == 'down'
                ? Icons.thumb_down
                : Icons.thumb_down_outlined,
            tooltip: l10n.actionBadAnswer,
            color: feedbackValue == 'down' ? p.danger : p.textMuted,
            onTap: () =>
                onFeedback?.call(feedbackValue == 'down' ? null : 'down'),
          ),
          if (onRegenerate != null)
            _FooterButton(
              icon: Icons.refresh_rounded,
              tooltip: l10n.actionRegenerate,
              color: p.textMuted,
              onTap: onRegenerate!,
            ),
          _FooterButton(
            icon: Icons.ios_share_rounded,
            tooltip: l10n.actionShare,
            color: p.textMuted,
            onTap: _share,
          ),
        ],
      ),
    );
  }

  Future<void> _copy(BuildContext context, AppLocalizations l10n) async {
    final messenger = ScaffoldMessenger.of(context);
    await Clipboard.setData(ClipboardData(text: _plainText));
    messenger.showSnackBar(
      SnackBar(
        content: Text(l10n.actionCopied),
        duration: const Duration(seconds: 1),
      ),
    );
  }

  Future<void> _share() async {
    final text = _plainText;
    if (text.isEmpty) {
      return;
    }
    await SharePlus.instance.share(ShareParams(text: text));
  }

  Widget _note(ThemeData theme, String text, Color color) {
    return Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Text(
        text,
        style: theme.textTheme.bodySmall?.copyWith(color: color),
      ),
    );
  }
}

class _FooterButton extends StatelessWidget {
  const _FooterButton({
    required this.icon,
    required this.tooltip,
    required this.color,
    required this.onTap,
  });

  final IconData icon;
  final String tooltip;
  final Color color;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.all(6),
          child: Icon(icon, size: 18, color: color),
        ),
      ),
    );
  }
}

/// Three pulsing dots shown while the assistant reply is still streaming and no
/// text has arrived yet.
class _TypingDots extends StatefulWidget {
  const _TypingDots({super.key, required this.color});

  final Color color;

  @override
  State<_TypingDots> createState() => _TypingDotsState();
}

class _TypingDotsState extends State<_TypingDots>
    with SingleTickerProviderStateMixin {
  late final AnimationController _c = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1100),
  )..repeat();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: AnimatedBuilder(
        animation: _c,
        builder: (context, _) {
          return Row(
            mainAxisSize: MainAxisSize.min,
            children: List.generate(3, (i) {
              final t = (_c.value + i * 0.2) % 1.0;
              final scale =
                  0.6 + 0.4 * (1 - (t - 0.5).abs() * 2).clamp(0.0, 1.0);
              return Padding(
                padding: const EdgeInsets.only(right: 5),
                child: Transform.scale(
                  scale: scale,
                  child: Container(
                    width: 7,
                    height: 7,
                    decoration: BoxDecoration(
                      color: widget.color,
                      shape: BoxShape.circle,
                    ),
                  ),
                ),
              );
            }),
          );
        },
      ),
    );
  }
}
