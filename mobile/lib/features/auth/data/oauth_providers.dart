import 'dart:async';

import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_web_auth_2/flutter_web_auth_2.dart';
import 'package:vkid_flutter_sdk/library_vkid.dart';

import '../../../core/config/app_config.dart';
import '../../../core/network/api_exception.dart';

/// Yandex ID browser-flow result: the authorization code to forward to the
/// backend (which holds the PKCE verifier and the client_secret).
class YandexAuthCode {
  const YandexAuthCode({required this.code});

  final String code;
}

/// VK ID SDK (confidential flow) result: the code plus the device_id VK
/// issued alongside it — both are required by the backend exchange.
class VkAuthCode {
  const VkAuthCode({required this.code, required this.deviceId});

  final String code;
  final String deviceId;
}

/// Native OAuth flows. An interface so the repository can be unit-tested with a
/// fake (the real plugins require a device).
abstract interface class OAuthProvider {
  /// Opens [authUrl] (built by the backend) in the system browser and returns
  /// the authorization code. The redirect's `state` is checked against
  /// [expectedState] before the code is accepted.
  Future<YandexAuthCode> getYandexAuthCode({
    required String authUrl,
    required String expectedState,
  });

  /// Runs the VK ID SDK with backend-issued PKCE material ([codeChallenge])
  /// so the SDK cannot exchange the code itself — only our backend can.
  Future<VkAuthCode> getVkAuthCode({
    required String state,
    required String codeChallenge,
  });
}

class PlatformOAuthProvider implements OAuthProvider {
  @override
  Future<YandexAuthCode> getYandexAuthCode({
    required String authUrl,
    required String expectedState,
  }) async {
    final String callback;
    try {
      callback = await FlutterWebAuth2.authenticate(
        url: authUrl,
        callbackUrlScheme: AppConfig.yandexCallbackScheme,
      );
    } on PlatformException catch (e) {
      if (e.code == 'CANCELED') {
        throw const OAuthCanceledException();
      }
      throw ApiException(
        code: 'internal_error',
        message: e.message ?? 'Yandex sign-in failed',
      );
    }
    final params = Uri.parse(callback).queryParameters;
    final code = params['code'];
    if (params['state'] != expectedState || code == null || code.isEmpty) {
      // A mismatched state means the redirect does not belong to this attempt
      // (CSRF / code injection) — never forward such a code.
      throw const ApiException(
        code: 'unauthorized',
        message: 'Yandex sign-in could not be confirmed',
      );
    }
    return YandexAuthCode(code: code);
  }

  @override
  Future<VkAuthCode> getVkAuthCode({
    required String state,
    required String codeChallenge,
  }) async {
    final vkid = await VKID.getInstance();
    final completer = Completer<VkAuthCode>();
    vkid.authorize(
      params: AuthParamsBuilder()
          .withAuthFlow(ConfidentialFlowData(state, codeChallenge))
          .withLocale(AuthLocale.ru)
          .withScopes({'email'})
          .build(),
      onAuthCode: (AuthCodeData data, bool isCompletion) {
        if (!completer.isCompleted) {
          completer.complete(
            VkAuthCode(code: data.code, deviceId: data.deviceID),
          );
        }
      },
      onError: (AuthError error) {
        if (completer.isCompleted) return;
        completer.completeError(switch (error) {
          AuthCancelledError() => const OAuthCanceledException(),
          AuthOtherError(:final description) => ApiException(
            code: 'internal_error',
            message: description,
          ),
        });
      },
    );
    return completer.future;
  }
}

final oauthProvider = Provider<OAuthProvider>((ref) => PlatformOAuthProvider());
