/// App-wide configuration resolved at build time via --dart-define.
///
/// The default base URL targets a locally running backend reachable from the
/// Android emulator (10.0.2.2 is the host loopback). Override per environment:
/// `flutter run --dart-define=API_BASE_URL=https://api.agronomai.site/v1`.
class AppConfig {
  const AppConfig._();

  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://10.0.2.2:8080/v1',
  );

  /// OAuth web/server client ID (Google). Used as `serverClientId` so the
  /// returned id_token's `aud` matches the backend allowlist. Supplied per
  /// environment once Google Cloud OAuth clients exist (Stage 0.6).
  static const String googleServerClientId = String.fromEnvironment(
    'GOOGLE_SERVER_CLIENT_ID',
    defaultValue: '',
  );

  /// Sentry DSN. Empty by default so dev/CI builds run with Sentry disabled
  /// (the SDK is a no-op without a DSN). Supplied for release builds via
  /// `--dart-define=SENTRY_DSN=...`; never hardcoded (CLAUDE.md "никаких
  /// секретов в коде").
  static const String sentryDsn = String.fromEnvironment(
    'SENTRY_DSN',
    defaultValue: '',
  );

  /// Sentry environment tag (e.g. `prod`, `dev`).
  static const String sentryEnv = String.fromEnvironment(
    'SENTRY_ENV',
    defaultValue: 'dev',
  );

  /// Legal document URLs shown at the consent point (ARCH §6.3 / App Store
  /// §5.1.1). The documents themselves are published by the operator; only the
  /// URLs are configurable here, never hardcoded in widgets.
  static const String privacyPolicyUrl = String.fromEnvironment(
    'PRIVACY_POLICY_URL',
    defaultValue: 'https://agronomai.site/privacy',
  );

  static const String termsOfServiceUrl = String.fromEnvironment(
    'TERMS_OF_SERVICE_URL',
    defaultValue: 'https://agronomai.site/terms',
  );
}
