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
  String get emailRequestTitle => 'Вход по email';

  @override
  String get emailRequestHint => 'Введите email — мы пришлём код для входа';

  @override
  String get emailFieldLabel => 'Email';

  @override
  String get emailRequestCta => 'Получить код';

  @override
  String get emailVerifyTitle => 'Введите код';

  @override
  String emailVerifyHint(String email) {
    return 'Код отправлен на $email';
  }

  @override
  String get codeFieldLabel => 'Код из 6 цифр';

  @override
  String get emailVerifyCta => 'Войти';

  @override
  String get resendCode => 'Отправить код повторно';

  @override
  String get errorInvalidEmail => 'Некорректный email';

  @override
  String get errorInvalidCode => 'Неверный или истёкший код';

  @override
  String get errorTooManyAttempts =>
      'Слишком много попыток. Запросите новый код.';

  @override
  String get errorRateLimited => 'Слишком часто. Попробуйте позже.';

  @override
  String get errorNetwork => 'Нет соединения. Проверьте интернет.';

  @override
  String get errorGeneric => 'Что-то пошло не так. Попробуйте ещё раз.';

  @override
  String get chatTitle => 'Чат';

  @override
  String get chatEmptyHint => 'Сфотографируйте растение или опишите проблему';

  @override
  String get chatInputPlaceholder => 'Сообщение';

  @override
  String get chatLogout => 'Выйти';

  @override
  String get chatDeleteAccount => 'Удалить аккаунт';

  @override
  String get deleteAccountConfirmTitle => 'Удалить аккаунт?';

  @override
  String get deleteAccountConfirmBody =>
      'Это действие необратимо. Все данные будут удалены.';

  @override
  String get commonCancel => 'Отмена';

  @override
  String get commonDelete => 'Удалить';
}
