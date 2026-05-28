import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_client.dart';
import '../../../core/network/api_exception.dart';
import '../data/auth_repository.dart';
import '../domain/auth_models.dart';

/// Session state. The async wrapper's "loading" is used only during startup
/// bootstrap; in-flight sign-in actions surface errors to the calling screen
/// and never flip the app back to the splash.
sealed class AuthSnapshot {
  const AuthSnapshot();
}

class Authenticated extends AuthSnapshot {
  const Authenticated(this.user);
  final AppUser user;
}

class Unauthenticated extends AuthSnapshot {
  const Unauthenticated();
}

enum AuthStatus { unknown, authenticated, unauthenticated }

class AuthController extends AsyncNotifier<AuthSnapshot> {
  AuthRepository get _repo => ref.read(authRepositoryProvider);

  @override
  Future<AuthSnapshot> build() async {
    // Refresh failures inside the HTTP layer end the session app-wide.
    ref.read(apiClientProvider).onSessionExpired = () {
      state = const AsyncData(Unauthenticated());
    };
    try {
      final user = await _repo.tryRestoreSession();
      return user == null ? const Unauthenticated() : Authenticated(user);
    } on NetworkException {
      // Offline at startup: keep stored tokens, require sign-in for now.
      return const Unauthenticated();
    }
  }

  Future<void> signInWithApple() async {
    try {
      state = AsyncData(Authenticated(await _repo.signInWithApple()));
    } on OAuthCanceledException {
      // User dismissed the sheet — stay where we are.
    }
  }

  Future<void> signInWithGoogle() async {
    try {
      state = AsyncData(Authenticated(await _repo.signInWithGoogle()));
    } on OAuthCanceledException {
      // User dismissed the sheet — stay where we are.
    }
  }

  /// Not a session transition — screen manages its own loading/error.
  Future<void> requestEmailCode(String email) => _repo.requestEmailCode(email);

  Future<void> verifyEmailCode({
    required String email,
    required String code,
  }) async {
    state = AsyncData(
      Authenticated(await _repo.verifyEmailCode(email: email, code: code)),
    );
  }

  Future<void> logout() async {
    await _repo.logout();
    state = const AsyncData(Unauthenticated());
  }

  Future<void> deleteAccount() async {
    await _repo.deleteAccount();
    state = const AsyncData(Unauthenticated());
  }
}

final authControllerProvider =
    AsyncNotifierProvider<AuthController, AuthSnapshot>(AuthController.new);

/// Synchronous projection for the router redirect.
final authStatusProvider = Provider<AuthStatus>((ref) {
  return ref
      .watch(authControllerProvider)
      .maybeWhen(
        data: (snapshot) => switch (snapshot) {
          Authenticated() => AuthStatus.authenticated,
          Unauthenticated() => AuthStatus.unauthenticated,
        },
        orElse: () => AuthStatus.unknown,
      );
});
