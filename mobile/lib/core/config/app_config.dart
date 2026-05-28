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
}
