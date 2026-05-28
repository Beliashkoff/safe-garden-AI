import '../../../core/network/api_exception.dart';
import '../../../l10n/generated/app_localizations.dart';

/// Maps an [AppException] to a localized, user-facing message. Backend error
/// codes (§4.7) drive the mapping; anything unrecognized falls back to a
/// generic message.
String authErrorMessage(AppLocalizations l10n, Object error) {
  if (error is NetworkException) {
    return l10n.errorNetwork;
  }
  if (error is ApiException) {
    switch (error.code) {
      case 'validation_failed':
        return l10n.errorInvalidEmail;
      case 'unauthorized':
        return l10n.errorInvalidCode;
      case 'rate_limited':
        return l10n.errorRateLimited;
      default:
        return l10n.errorGeneric;
    }
  }
  return l10n.errorGeneric;
}
