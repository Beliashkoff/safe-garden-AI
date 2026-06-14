import 'package:flutter/material.dart';

import '../../../../app/theme.dart';
import '../../../../app/widgets/brand_logo.dart';
import '../../../../l10n/generated/app_localizations.dart';

/// The side panel opened from the header hamburger: brand identity at the top,
/// account actions (sign out, delete account), and the legal links pinned to
/// the bottom.
class ChatDrawer extends StatelessWidget {
  const ChatDrawer({
    required this.onLogout,
    required this.onDeleteAccount,
    required this.onOpenPrivacy,
    required this.onOpenTerms,
    super.key,
  });

  final VoidCallback onLogout;
  final VoidCallback onDeleteAccount;
  final VoidCallback onOpenPrivacy;
  final VoidCallback onOpenTerms;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = theme.palette;
    final l10n = AppLocalizations.of(context)!;

    return Drawer(
      child: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 24, 20, 20),
              child: Row(
                children: [
                  const BrandLogo(size: 44),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(l10n.appTitle, style: theme.textTheme.titleMedium),
                        const SizedBox(height: 2),
                        Text(
                          l10n.drawerSubtitle,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: p.textSubtle,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            Divider(color: p.border, height: 1),
            const SizedBox(height: 8),
            ListTile(
              leading: const Icon(Icons.logout_rounded),
              title: Text(l10n.chatLogout),
              onTap: onLogout,
            ),
            ListTile(
              leading: Icon(Icons.delete_outline_rounded, color: p.danger),
              title: Text(
                l10n.chatDeleteAccount,
                style: TextStyle(color: p.danger),
              ),
              onTap: onDeleteAccount,
            ),
            const Spacer(),
            Divider(color: p.border, height: 1),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              child: Row(
                children: [
                  TextButton(
                    onPressed: onOpenPrivacy,
                    child: Text(
                      l10n.consentPrivacy,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: p.textMuted,
                      ),
                    ),
                  ),
                  TextButton(
                    onPressed: onOpenTerms,
                    child: Text(
                      l10n.consentTerms,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: p.textMuted,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
