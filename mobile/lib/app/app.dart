import 'package:flutter/material.dart';

import '../l10n/generated/app_localizations.dart';
import 'router.dart';
import 'theme.dart';

class AgronomApp extends StatelessWidget {
  const AgronomApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      onGenerateTitle: (ctx) => AppLocalizations.of(ctx)!.appTitle,
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      themeMode: ThemeMode.system,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: const Locale('ru'),
      routerConfig: appRouter,
      debugShowCheckedModeBanner: false,
    );
  }
}
