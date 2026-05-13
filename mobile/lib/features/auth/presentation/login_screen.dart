import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../l10n/generated/app_localizations.dart';

class LoginScreen extends StatelessWidget {
  const LoginScreen({super.key});

  void _showComingSoon(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(l10n.loginComingSoon)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final platform = theme.platform;
    final showAppleButton =
        platform == TargetPlatform.iOS || platform == TargetPlatform.macOS;

    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                l10n.loginTitle,
                textAlign: TextAlign.center,
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 12),
              Text(
                l10n.loginSubtitle,
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyLarge,
              ),
              const SizedBox(height: 48),
              if (showAppleButton) ...[
                FilledButton(
                  onPressed: () => _showComingSoon(context),
                  child: Text(l10n.loginButtonApple),
                ),
                const SizedBox(height: 12),
              ],
              FilledButton.tonal(
                onPressed: () => _showComingSoon(context),
                child: Text(l10n.loginButtonGoogle),
              ),
              const SizedBox(height: 12),
              OutlinedButton(
                onPressed: () => context.go('/chat'),
                child: Text(l10n.loginButtonEmail),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
