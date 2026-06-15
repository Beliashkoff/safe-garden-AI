import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';

import '../../../../app/theme.dart';

/// Highlights the actionable "Что делать" section of an assistant answer as an
/// inset card: a heading row with a checklist glyph, then each step on its own
/// row behind a soft green number badge. Step text keeps its inline Markdown
/// (dosages in bold, nested sub-lists, etc.). The card uses the [AppPalette.soft]
/// fill with a [AppPalette.borderStrong] outline so it reads as a distinct,
/// highlighted block against the white chat surface while staying flat (no
/// elevation), in keeping with the loft look.
class ActionStepsCard extends StatelessWidget {
  const ActionStepsCard({required this.title, required this.steps, super.key});

  final String title;
  final List<String> steps;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = theme.palette;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 16),
      decoration: BoxDecoration(
        color: p.soft,
        borderRadius: BorderRadius.circular(AppRadius.card),
        border: Border.all(color: p.borderStrong),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          _header(theme, p),
          const SizedBox(height: 14),
          for (var n = 0; n < steps.length; n++) ...[
            if (n > 0) const SizedBox(height: 12),
            _step(theme, p, n + 1, steps[n]),
          ],
        ],
      ),
    );
  }

  Widget _header(ThemeData theme, AppPalette p) {
    return Row(
      children: [
        Icon(Icons.checklist_rounded, size: 18, color: p.brandGreen),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            title,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: p.textMuted,
              fontWeight: FontWeight.w600,
              letterSpacing: -0.1,
            ),
          ),
        ),
      ],
    );
  }

  Widget _step(ThemeData theme, AppPalette p, int number, String text) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _badge(theme, p, number),
        const SizedBox(width: 12),
        Expanded(
          child: Padding(
            padding: const EdgeInsets.only(top: 1),
            child: MarkdownBody(
              data: text,
              selectable: true,
              styleSheet: MarkdownStyleSheet.fromTheme(theme).copyWith(
                p: theme.textTheme.bodyLarge,
                pPadding: EdgeInsets.zero,
                blockSpacing: 4,
              ),
            ),
          ),
        ),
      ],
    );
  }

  Widget _badge(ThemeData theme, AppPalette p, int number) {
    return Container(
      width: 24,
      height: 24,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: p.brandGreenSoft,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        '$number',
        style: theme.textTheme.bodySmall?.copyWith(
          color: p.brandGreen,
          fontWeight: FontWeight.w700,
          height: 1,
        ),
      ),
    );
  }
}
