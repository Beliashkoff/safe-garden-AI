import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../l10n/generated/app_localizations.dart';
import '../application/auth_controller.dart';
import 'auth_error_message.dart';

class EmailRequestScreen extends ConsumerStatefulWidget {
  const EmailRequestScreen({super.key});

  @override
  ConsumerState<EmailRequestScreen> createState() => _EmailRequestScreenState();
}

class _EmailRequestScreenState extends ConsumerState<EmailRequestScreen> {
  final _controllerText = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _busy = false;

  @override
  void dispose() {
    _controllerText.dispose();
    super.dispose();
  }

  static final _emailRegex = RegExp(r'^[^@\s]+@[^@\s]+\.[^@\s]+$');

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    final email = _controllerText.text.trim();
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    final router = GoRouter.of(context);
    setState(() => _busy = true);
    try {
      await ref.read(authControllerProvider.notifier).requestEmailCode(email);
      router.go('/login/email/verify', extra: email);
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

    return Scaffold(
      appBar: AppBar(title: Text(l10n.emailRequestTitle)),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 24),
          child: Form(
            key: _formKey,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(l10n.emailRequestHint, style: theme.textTheme.bodyLarge),
                const SizedBox(height: 24),
                TextFormField(
                  controller: _controllerText,
                  enabled: !_busy,
                  keyboardType: TextInputType.emailAddress,
                  autofillHints: const [AutofillHints.email],
                  decoration: InputDecoration(
                    labelText: l10n.emailFieldLabel,
                    border: const OutlineInputBorder(),
                  ),
                  validator: (value) {
                    final v = value?.trim() ?? '';
                    return _emailRegex.hasMatch(v)
                        ? null
                        : l10n.errorInvalidEmail;
                  },
                  onFieldSubmitted: (_) => _busy ? null : _submit(),
                ),
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: _busy ? null : _submit,
                  child: _busy
                      ? const SizedBox(
                          height: 20,
                          width: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Text(l10n.emailRequestCta),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
