import 'package:freezed_annotation/freezed_annotation.dart';

part 'auth_models.freezed.dart';
part 'auth_models.g.dart';

/// Which sign-in methods are linked to the account. Mirrors the backend
/// `user.providers` object.
@freezed
class AuthProviders with _$AuthProviders {
  const factory AuthProviders({
    @Default(false) bool apple,
    @Default(false) bool google,
    @Default(false) bool email,
  }) = _AuthProviders;

  factory AuthProviders.fromJson(Map<String, dynamic> json) =>
      _$AuthProvidersFromJson(json);
}

/// The authenticated user profile (backend `user` object).
@freezed
class AppUser with _$AppUser {
  const factory AppUser({
    required String id,
    String? email,
    @JsonKey(name: 'display_name') String? displayName,
    @JsonKey(name: 'email_verified') @Default(false) bool emailVerified,
    @Default(AuthProviders()) AuthProviders providers,
  }) = _AppUser;

  factory AppUser.fromJson(Map<String, dynamic> json) =>
      _$AppUserFromJson(json);
}

/// Access + refresh token pair.
@freezed
class AuthTokens with _$AuthTokens {
  const factory AuthTokens({
    @JsonKey(name: 'access_token') required String accessToken,
    @JsonKey(name: 'refresh_token') required String refreshToken,
  }) = _AuthTokens;

  factory AuthTokens.fromJson(Map<String, dynamic> json) =>
      _$AuthTokensFromJson(json);
}

/// Response of the sign-in endpoints (Apple/Google/email-verify).
@freezed
class SignInResponse with _$SignInResponse {
  const factory SignInResponse({
    @JsonKey(name: 'access_token') required String accessToken,
    @JsonKey(name: 'refresh_token') required String refreshToken,
    required AppUser user,
  }) = _SignInResponse;

  factory SignInResponse.fromJson(Map<String, dynamic> json) =>
      _$SignInResponseFromJson(json);

  const SignInResponse._();

  AuthTokens get tokens =>
      AuthTokens(accessToken: accessToken, refreshToken: refreshToken);
}

/// Response of POST /auth/refresh.
@freezed
class RefreshResponse with _$RefreshResponse {
  const factory RefreshResponse({
    @JsonKey(name: 'access_token') required String accessToken,
    @JsonKey(name: 'refresh_token') required String refreshToken,
  }) = _RefreshResponse;

  factory RefreshResponse.fromJson(Map<String, dynamic> json) =>
      _$RefreshResponseFromJson(json);
}
