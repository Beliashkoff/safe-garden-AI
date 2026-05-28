import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/api_exception.dart';
import '../../../core/storage/secure_token_store.dart';
import '../domain/auth_models.dart';
import 'auth_api.dart';
import 'oauth_providers.dart';

/// Coordinates the auth API, token storage, and native OAuth flows. The
/// controller layer talks only to this.
class AuthRepository {
  AuthRepository({
    required AuthApi api,
    required TokenStore store,
    required OAuthProvider oauth,
  }) : _api = api,
       _store = store,
       _oauth = oauth;

  final AuthApi _api;
  final TokenStore _store;
  final OAuthProvider _oauth;

  Future<AppUser> signInWithApple() async {
    final credential = await _oauth.getAppleCredential();
    final response = await _api.signInApple(
      idToken: credential.identityToken,
      nonce: credential.rawNonce,
    );
    await _persist(response);
    return response.user;
  }

  Future<AppUser> signInWithGoogle() async {
    final idToken = await _oauth.getGoogleIdToken();
    final response = await _api.signInGoogle(idToken: idToken);
    await _persist(response);
    return response.user;
  }

  Future<void> requestEmailCode(String email) => _api.requestEmailCode(email);

  Future<AppUser> verifyEmailCode({
    required String email,
    required String code,
  }) async {
    final response = await _api.verifyEmailCode(email: email, code: code);
    await _persist(response);
    return response.user;
  }

  /// Attempts to restore a session at startup. Returns the user if a stored
  /// refresh token still yields a valid account, null if there is no session
  /// (or it is no longer valid). Network errors are rethrown for the caller to
  /// surface/retry.
  Future<AppUser?> tryRestoreSession() async {
    final refresh = await _store.readRefreshToken();
    if (refresh == null || refresh.isEmpty) {
      return null;
    }
    try {
      return await _api.getAccount();
    } on NetworkException {
      rethrow;
    } on ApiException {
      // Unauthorized (refresh failed/revoked) or account gone → logged out.
      await _store.clear();
      return null;
    }
  }

  Future<void> logout() async {
    final refresh = await _store.readRefreshToken();
    if (refresh != null && refresh.isNotEmpty) {
      try {
        await _api.logout(refresh);
      } on AppException {
        // Best-effort server revoke; always clear locally.
      }
    }
    await _store.clear();
  }

  Future<void> deleteAccount() async {
    await _api.deleteAccount();
    await _store.clear();
  }

  Future<void> _persist(SignInResponse response) => _store.writeTokens(
    accessToken: response.accessToken,
    refreshToken: response.refreshToken,
  );
}

final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepository(
    api: ref.watch(authApiProvider),
    store: ref.watch(secureTokenStoreProvider),
    oauth: ref.watch(oauthProvider),
  );
});
