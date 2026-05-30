import '../../../core/network/api_exception.dart';
import '../../../l10n/generated/app_localizations.dart';

/// Maps a chat failure to a localized, user-facing message. Accepts either an
/// [AppException] or a raw backend error code.
String chatErrorMessage(AppLocalizations l10n, Object error) {
  final code = error is AppException ? error.code : error.toString();
  return chatErrorMessageForCode(l10n, code);
}

/// Maps a backend error code (§4.7) to a localized message.
String chatErrorMessageForCode(AppLocalizations l10n, String code) {
  switch (code) {
    case 'network':
      return l10n.chatErrorNetwork;
    case 'rate_limited':
      return l10n.chatErrorRateLimited;
    case 'unsupported_media_type':
      return l10n.chatErrorUnsupported;
    case 'payload_too_large':
      return l10n.chatErrorTooLarge;
    default:
      return l10n.chatErrorGeneric;
  }
}
