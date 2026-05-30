// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'AI Agronomist';

  @override
  String get loginTitle => 'Sign in';

  @override
  String get loginSubtitle => 'Plant diagnosis with AI';

  @override
  String get loginButtonApple => 'Sign in with Apple';

  @override
  String get loginButtonGoogle => 'Sign in with Google';

  @override
  String get loginButtonEmail => 'Sign in with email';

  @override
  String get emailRequestTitle => 'Sign in with email';

  @override
  String get emailRequestHint =>
      'Enter your email — we\'ll send you a sign-in code';

  @override
  String get emailFieldLabel => 'Email';

  @override
  String get emailRequestCta => 'Send code';

  @override
  String get emailVerifyTitle => 'Enter code';

  @override
  String emailVerifyHint(String email) {
    return 'Code sent to $email';
  }

  @override
  String get codeFieldLabel => '6-digit code';

  @override
  String get emailVerifyCta => 'Sign in';

  @override
  String get resendCode => 'Resend code';

  @override
  String get errorInvalidEmail => 'Invalid email';

  @override
  String get errorInvalidCode => 'Invalid or expired code';

  @override
  String get errorTooManyAttempts => 'Too many attempts. Request a new code.';

  @override
  String get errorRateLimited => 'Too many requests. Try again later.';

  @override
  String get errorNetwork => 'No connection. Check your internet.';

  @override
  String get errorGeneric => 'Something went wrong. Please try again.';

  @override
  String get chatTitle => 'Chat';

  @override
  String get chatEmptyHint =>
      'Take a photo of the plant or describe the problem';

  @override
  String get chatInputPlaceholder => 'Message';

  @override
  String get chatLogout => 'Sign out';

  @override
  String get chatDeleteAccount => 'Delete account';

  @override
  String get chatSend => 'Send';

  @override
  String get chatStop => 'Stop';

  @override
  String get chatRetry => 'Retry';

  @override
  String get chatCancelledNote => 'Response stopped';

  @override
  String get chatDeleteMessageConfirm => 'Delete this message?';

  @override
  String get chatErrorNetwork => 'No connection. Check your internet.';

  @override
  String get chatErrorRateLimited => 'Too many messages. Please wait a moment.';

  @override
  String get chatErrorUnsupported => 'This content type is not supported yet.';

  @override
  String get chatErrorTooLarge => 'The message is too long.';

  @override
  String get chatErrorGeneric => 'Couldn\'t get a response. Please try again.';

  @override
  String get deleteAccountConfirmTitle => 'Delete account?';

  @override
  String get deleteAccountConfirmBody =>
      'This cannot be undone. All your data will be deleted.';

  @override
  String get commonCancel => 'Cancel';

  @override
  String get commonDelete => 'Delete';
}
