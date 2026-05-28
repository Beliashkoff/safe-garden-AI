// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$AuthProvidersImpl _$$AuthProvidersImplFromJson(Map<String, dynamic> json) =>
    _$AuthProvidersImpl(
      apple: json['apple'] as bool? ?? false,
      google: json['google'] as bool? ?? false,
      email: json['email'] as bool? ?? false,
    );

Map<String, dynamic> _$$AuthProvidersImplToJson(_$AuthProvidersImpl instance) =>
    <String, dynamic>{
      'apple': instance.apple,
      'google': instance.google,
      'email': instance.email,
    };

_$AppUserImpl _$$AppUserImplFromJson(Map<String, dynamic> json) =>
    _$AppUserImpl(
      id: json['id'] as String,
      email: json['email'] as String?,
      displayName: json['display_name'] as String?,
      emailVerified: json['email_verified'] as bool? ?? false,
      providers: json['providers'] == null
          ? const AuthProviders()
          : AuthProviders.fromJson(json['providers'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$$AppUserImplToJson(_$AppUserImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'email': instance.email,
      'display_name': instance.displayName,
      'email_verified': instance.emailVerified,
      'providers': instance.providers,
    };

_$AuthTokensImpl _$$AuthTokensImplFromJson(Map<String, dynamic> json) =>
    _$AuthTokensImpl(
      accessToken: json['access_token'] as String,
      refreshToken: json['refresh_token'] as String,
    );

Map<String, dynamic> _$$AuthTokensImplToJson(_$AuthTokensImpl instance) =>
    <String, dynamic>{
      'access_token': instance.accessToken,
      'refresh_token': instance.refreshToken,
    };

_$SignInResponseImpl _$$SignInResponseImplFromJson(Map<String, dynamic> json) =>
    _$SignInResponseImpl(
      accessToken: json['access_token'] as String,
      refreshToken: json['refresh_token'] as String,
      user: AppUser.fromJson(json['user'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$$SignInResponseImplToJson(
  _$SignInResponseImpl instance,
) => <String, dynamic>{
  'access_token': instance.accessToken,
  'refresh_token': instance.refreshToken,
  'user': instance.user,
};

_$RefreshResponseImpl _$$RefreshResponseImplFromJson(
  Map<String, dynamic> json,
) => _$RefreshResponseImpl(
  accessToken: json['access_token'] as String,
  refreshToken: json['refresh_token'] as String,
);

Map<String, dynamic> _$$RefreshResponseImplToJson(
  _$RefreshResponseImpl instance,
) => <String, dynamic>{
  'access_token': instance.accessToken,
  'refresh_token': instance.refreshToken,
};
