import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/config/app_config.dart';
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

  Future<void> _openUrl(String url) async {
    final uri = Uri.parse(url);
    if (await canLaunchUrl(uri)) {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    }
  }

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
              // Provider buttons use the providers' fixed brand colors — an
              // explicit exception to the theme-only rule: VK ID and Yandex ID
              // design guidelines forbid recoloring their sign-in buttons.
              FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: _vkBrandBlue,
                  foregroundColor: _brandWhite,
                ),
                onPressed: _busy
                    ? null
                    : () => _runOAuth(controller.signInWithVk),
                child: Text(l10n.loginButtonVk),
              ),
              const SizedBox(height: 12),
              FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: _yandexBrandBlack,
                  foregroundColor: _brandWhite,
                ),
                onPressed: _busy
                    ? null
                    : () => _runOAuth(controller.signInWithYandex),
                child: Text(l10n.loginButtonYandex),
              ),
              const SizedBox(height: 12),
              OutlinedButton(
                onPressed: _busy ? null : () => context.go('/login/email'),
                child: Text(l10n.loginButtonEmail),
              ),
              const SizedBox(height: 32),
              _ConsentNotice(onOpen: _openUrl),
            ],
          ),
        ),
      ),
    );
  }
}

// Brand palette mandated by the providers' button guidelines (VK ID
// design-rules, Yandex ID buttons-design): the colors may not be altered.
const _vkBrandBlue = Color(0xFF0077FF);
const _yandexBrandBlack = Color(0xFF000000);
const _brandWhite = Color(0xFFFFFFFF);

/// Informational consent shown at the registration point (ARCH §6.3): by
/// continuing, the user accepts the Privacy Policy and Terms of Service, both
/// tappable. Does not block sign-in; the action itself constitutes consent.
class _ConsentNotice extends StatelessWidget {
  const _ConsentNotice({required this.onOpen});

  final Future<void> Function(String url) onOpen;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final base = theme.textTheme.bodySmall;
    final link = base?.copyWith(
      color: theme.colorScheme.primary,
      decoration: TextDecoration.underline,
    );

    return Wrap(
      alignment: WrapAlignment.center,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        Text('${l10n.consentPrefix} ', style: base),
        InkWell(
          onTap: () => onOpen(AppConfig.privacyPolicyUrl),
          child: Text(l10n.consentPrivacy, style: link),
        ),
        Text(' ${l10n.consentAnd} ', style: base),
        InkWell(
          onTap: () => onOpen(AppConfig.termsOfServiceUrl),
          child: Text(l10n.consentTerms, style: link),
        ),
      ],
    );
  }
}
