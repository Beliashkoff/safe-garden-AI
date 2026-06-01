import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../data/onboarding_store.dart';

/// Synchronous projection for the router redirect, mirroring AuthStatus:
/// `unknown` while the flag loads, then `seen` / `notSeen`.
enum OnboardingStatus { unknown, notSeen, seen }

class OnboardingController extends AsyncNotifier<bool> {
  OnboardingStore get _store => ref.read(onboardingStoreProvider);

  @override
  Future<bool> build() => _store.hasSeen();

  /// Marks onboarding complete; the router redirect then sends the user on to
  /// the login subtree.
  Future<void> complete() async {
    await _store.markSeen();
    state = const AsyncData(true);
  }
}

final onboardingControllerProvider =
    AsyncNotifierProvider<OnboardingController, bool>(OnboardingController.new);

final onboardingStatusProvider = Provider<OnboardingStatus>((ref) {
  return ref
      .watch(onboardingControllerProvider)
      .maybeWhen(
        data: (seen) => seen ? OnboardingStatus.seen : OnboardingStatus.notSeen,
        orElse: () => OnboardingStatus.unknown,
      );
});
