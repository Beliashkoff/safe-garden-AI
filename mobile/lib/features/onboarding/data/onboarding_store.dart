import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Persists whether the user has completed the intro flow. Backed by the same
/// secure store as tokens (no extra dependency); the value is not sensitive but
/// keeping one storage mechanism is simpler.
abstract interface class OnboardingStore {
  Future<bool> hasSeen();
  Future<void> markSeen();
}

class SecureOnboardingStore implements OnboardingStore {
  SecureOnboardingStore(this._storage);

  static const _key = 'onboarding_seen';

  final FlutterSecureStorage _storage;

  @override
  Future<bool> hasSeen() async => (await _storage.read(key: _key)) == 'true';

  @override
  Future<void> markSeen() => _storage.write(key: _key, value: 'true');
}

/// Override in tests with an in-memory fake.
final onboardingStoreProvider = Provider<OnboardingStore>((ref) {
  return SecureOnboardingStore(
    const FlutterSecureStorage(
      aOptions: AndroidOptions(encryptedSharedPreferences: true),
    ),
  );
});
