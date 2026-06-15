import 'package:flutter/material.dart';

import '../../../../app/theme.dart';
import '../../../../app/widgets/brand_logo.dart';
import '../../../../l10n/generated/app_localizations.dart';

/// The three starter prompts shown on the empty chat.
enum ChatSuggestion { disease, deficiency, plan }

/// Shown when the conversation has no messages yet: a brand hero, a greeting,
/// and three tappable suggestion chips (ChatGPT-style first run).
class ChatEmptyState extends StatelessWidget {
  const ChatEmptyState({required this.onSuggestion, super.key});

  final void Function(ChatSuggestion) onSuggestion;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = theme.palette;
    final l10n = AppLocalizations.of(context)!;

    final items = <(IconData, String, ChatSuggestion)>[
      (Icons.image_outlined, l10n.chatSuggestDisease, ChatSuggestion.disease),
      (
        Icons.eco_outlined,
        l10n.chatSuggestDeficiency,
        ChatSuggestion.deficiency,
      ),
      (Icons.auto_awesome_outlined, l10n.chatSuggestPlan, ChatSuggestion.plan),
    ];

    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(24, 24, 24, 24),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(18),
                boxShadow: [
                  BoxShadow(
                    color: p.brandGreen.withValues(alpha: 0.25),
                    blurRadius: 32,
                    offset: const Offset(0, 12),
                  ),
                ],
              ),
              child: const BrandLogo(size: 64),
            ),
            const SizedBox(height: 28),
            Text(
              l10n.chatGreeting,
              style: theme.textTheme.headlineMedium,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              l10n.chatGreetingSubtitle,
              style: theme.textTheme.bodyLarge?.copyWith(color: p.textMuted),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 28),
            for (final (icon, label, kind) in items) ...[
              _SuggestionChip(
                icon: icon,
                label: label,
                onTap: () => onSuggestion(kind),
              ),
              const SizedBox(height: 8),
            ],
          ],
        ),
      ),
    );
  }
}

class _SuggestionChip extends StatelessWidget {
  const _SuggestionChip({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = theme.palette;
    return Material(
      color: p.soft,
      borderRadius: BorderRadius.circular(AppRadius.chip),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(AppRadius.chip),
        child: Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(AppRadius.chip),
            border: Border.all(color: p.border),
          ),
          child: Row(
            children: [
              Icon(icon, size: 20, color: p.textMuted),
              const SizedBox(width: 12),
              Expanded(child: Text(label, style: theme.textTheme.bodyLarge)),
            ],
          ),
        ),
      ),
    );
  }
}
