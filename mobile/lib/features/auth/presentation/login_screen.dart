import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../l10n/generated/app_localizations.dart';
import '../application/auth_controller.dart';
import 'auth_error_message.dart';

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  bool _busy = false;

  Future<void> _runOAuth(Future<void> Function() action) async {
    setState(() => _busy = true);
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await action();
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(authErrorMessage(l10n, e))),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final controller = ref.read(authControllerProvider.notifier);
    final showAppleButton =
        theme.platform == TargetPlatform.iOS ||
        theme.platform == TargetPlatform.macOS;

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
                  onPressed: _busy
                      ? null
                      : () => _runOAuth(controller.signInWithApple),
                  child: Text(l10n.loginButtonApple),
                ),
                const SizedBox(height: 12),
              ],
              FilledButton.tonal(
                onPressed: _busy
                    ? null
                    : () => _runOAuth(controller.signInWithGoogle),
                child: Text(l10n.loginButtonGoogle),
              ),
              const SizedBox(height: 12),
              OutlinedButton(
                onPressed: _busy ? null : () => context.go('/login/email'),
                child: Text(l10n.loginButtonEmail),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
