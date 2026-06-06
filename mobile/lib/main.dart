import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import 'app/app.dart';
import 'core/config/app_config.dart';

Future<void> main() async {
  await SentryFlutter.init((options) {
    // Empty DSN → the SDK initializes as a no-op (dev/CI). Set for release
    // builds via --dart-define=SENTRY_DSN=...
    options.dsn = AppConfig.sentryDsn;
    options.environment = AppConfig.sentryEnv;
    // Never attach PII (request bodies, headers, user IP) to events
    // (CLAUDE.md invariant #3) — mirrors the backend SendDefaultPII: false.
    options.sendDefaultPii = false;
    // Errors and crashes only; no performance tracing (matches the backend).
    options.tracesSampleRate = 0.0;
  }, appRunner: () => runApp(const ProviderScope(child: AgronomApp())));
}
