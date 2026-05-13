import 'package:flutter/material.dart';

import '../../../l10n/generated/app_localizations.dart';

class ChatScreen extends StatelessWidget {
  const ChatScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.chatTitle),
      ),
      body: SafeArea(
        child: Column(
          children: [
            Expanded(
              child: Center(
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 32),
                  child: Text(
                    l10n.chatEmptyHint,
                    textAlign: TextAlign.center,
                    style: theme.textTheme.bodyLarge,
                  ),
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(12),
              child: TextField(
                enabled: false,
                decoration: InputDecoration(
                  hintText: l10n.chatInputPlaceholder,
                  border: const OutlineInputBorder(),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
