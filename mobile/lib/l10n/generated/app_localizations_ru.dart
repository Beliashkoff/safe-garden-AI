// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Russian (`ru`).
class AppLocalizationsRu extends AppLocalizations {
  AppLocalizationsRu([String locale = 'ru']) : super(locale);

  @override
  String get appTitle => 'ИИ Агроном';

  @override
  String get loginTitle => 'Войти';

  @override
  String get loginSubtitle => 'Диагностика растений с помощью AI';

  @override
  String get loginButtonApple => 'Войти с Apple';

  @override
  String get loginButtonGoogle => 'Войти с Google';

  @override
  String get loginButtonEmail => 'Войти по email';

  @override
  String get loginComingSoon => 'Скоро будет доступно';

  @override
  String get chatTitle => 'Чат';

  @override
  String get chatEmptyHint => 'Сфотографируйте растение или опишите проблему';

  @override
  String get chatInputPlaceholder => 'Сообщение';
}
