// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'AI Agronom';

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
  String get loginComingSoon => 'Coming soon';

  @override
  String get chatTitle => 'Chat';

  @override
  String get chatEmptyHint =>
      'Take a photo of the plant or describe the problem';

  @override
  String get chatInputPlaceholder => 'Message';
}
