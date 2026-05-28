import 'dart:convert';
import 'dart:math';

import 'package:crypto/crypto.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'package:sign_in_with_apple/sign_in_with_apple.dart';

import '../../../core/config/app_config.dart';
import '../../../core/network/api_exception.dart';

/// Apple credential to forward to the backend: the id_token plus the *raw*
/// nonce (the backend re-hashes and compares against the id_token's claim).
class AppleCredential {
  const AppleCredential({required this.identityToken, required this.rawNonce});

  final String identityToken;
  final String rawNonce;
}

/// Native OAuth flows. An interface so the repository can be unit-tested with a
/// fake (the real plugins require a device).
abstract interface class OAuthProvider {
  Future<AppleCredential> getAppleCredential();
  Future<String> getGoogleIdToken();
}

class PlatformOAuthProvider implements OAuthProvider {
  bool _googleInitialized = false;

  @override
  Future<AppleCredential> getAppleCredential() async {
    final rawNonce = _generateNonce();
    final hashedNonce = sha256.convert(utf8.encode(rawNonce)).toString();
    try {
      final credential = await SignInWithApple.getAppleIDCredential(
        scopes: const [
          AppleIDAuthorizationScopes.email,
          AppleIDAuthorizationScopes.fullName,
        ],
        nonce: hashedNonce,
      );
      final idToken = credential.identityToken;
      if (idToken == null) {
        throw const ApiException(
          code: 'internal_error',
          message: 'Apple did not return an identity token',
        );
      }
      return AppleCredential(identityToken: idToken, rawNonce: rawNonce);
    } on SignInWithAppleAuthorizationException catch (e) {
      if (e.code == AuthorizationErrorCode.canceled) {
        throw const OAuthCanceledException();
      }
      throw ApiException(code: 'internal_error', message: e.message);
    }
  }

  @override
  Future<String> getGoogleIdToken() async {
    final signIn = GoogleSignIn.instance;
    if (!_googleInitialized) {
      await signIn.initialize(
        serverClientId: AppConfig.googleServerClientId.isEmpty
            ? null
            : AppConfig.googleServerClientId,
      );
      _googleInitialized = true;
    }
    try {
      final account = await signIn.authenticate(scopeHint: const ['email']);
      final idToken = account.authentication.idToken;
      if (idToken == null) {
        throw const ApiException(
          code: 'internal_error',
          message: 'Google did not return an id token',
        );
      }
      return idToken;
    } on GoogleSignInException catch (e) {
      if (e.code == GoogleSignInExceptionCode.canceled) {
        throw const OAuthCanceledException();
      }
      throw ApiException(
        code: 'internal_error',
        message: e.description ?? 'Google sign-in failed',
      );
    }
  }

  static String _generateNonce([int length = 32]) {
    final rand = Random.secure();
    final bytes = List<int>.generate(length, (_) => rand.nextInt(256));
    return base64UrlEncode(bytes);
  }
}

final oauthProvider = Provider<OAuthProvider>((ref) => PlatformOAuthProvider());
