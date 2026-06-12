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
  String get loginButtonVk => 'Войти с VK ID';

  @override
  String get loginButtonYandex => 'Войти с Яндекс ID';

  @override
  String get loginButtonEmail => 'Войти по email';

  @override
  String get consentPrefix => 'Продолжая, вы принимаете';

  @override
  String get consentPrivacy => 'Политику конфиденциальности';

  @override
  String get consentAnd => 'и';

  @override
  String get consentTerms => 'Условия использования';

  @override
  String get onboardingTitle1 => 'Сфотографируйте растение';

  @override
  String get onboardingBody1 =>
      'Сделайте снимок проблемного места — ИИ-агроном определит, что не так.';

  @override
  String get onboardingTitle2 => 'Опишите голосом или текстом';

  @override
  String get onboardingBody2 =>
      'Расскажите о симптомах удобным способом — поддерживаются текст и голос.';

  @override
  String get onboardingTitle3 => 'Получите рекомендации';

  @override
  String get onboardingBody3 =>
      'Точный диагноз и подбор удобрений под ваше растение.';

  @override
  String get onboardingSkip => 'Пропустить';

  @override
  String get onboardingNext => 'Далее';

  @override
  String get onboardingStart => 'Начать';

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
  String get chatSend => 'Отправить';

  @override
  String get chatStop => 'Стоп';

  @override
  String get chatRetry => 'Повторить';

  @override
  String get chatCancelledNote => 'Ответ остановлен';

  @override
  String get chatDeleteMessageConfirm => 'Удалить это сообщение?';

  @override
  String get chatErrorNetwork => 'Нет соединения. Проверьте интернет.';

  @override
  String get chatErrorRateLimited =>
      'Слишком много сообщений. Подождите немного.';

  @override
  String get chatErrorUnsupported =>
      'Этот тип содержимого пока не поддерживается.';

  @override
  String get chatErrorTooLarge => 'Сообщение слишком длинное.';

  @override
  String get chatErrorGeneric =>
      'Не удалось получить ответ. Попробуйте ещё раз.';

  @override
  String get chatAttach => 'Прикрепить фото';

  @override
  String get chatAttachCamera => 'Камера';

  @override
  String get chatAttachGallery => 'Галерея';

  @override
  String get chatPermissionCameraTitle => 'Нужен доступ к камере';

  @override
  String get chatPermissionCameraBody =>
      'Разрешите доступ к камере в настройках, чтобы сфотографировать растение.';

  @override
  String get chatPermissionPhotosTitle => 'Нужен доступ к фото';

  @override
  String get chatPermissionPhotosBody =>
      'Разрешите доступ к фото в настройках, чтобы прикрепить снимок.';

  @override
  String get chatOpenSettings => 'Открыть настройки';

  @override
  String chatMaxPhotos(int max) {
    return 'Можно прикрепить до $max фото';
  }

  @override
  String get chatUploadFailed =>
      'Не удалось загрузить фото. Попробуйте ещё раз.';

  @override
  String get chatRemovePhoto => 'Удалить фото';

  @override
  String get voiceHoldToRecord => 'Удерживайте кнопку для записи';

  @override
  String get voiceSlideToCancel => 'Влево — отмена';

  @override
  String get voiceReleaseToCancel => 'Отпустите для отмены';

  @override
  String get voiceTranscribing => 'Расшифровка…';

  @override
  String get voicePermissionTitle => 'Нужен доступ к микрофону';

  @override
  String get voicePermissionBody =>
      'Разрешите доступ к микрофону в настройках, чтобы записать голосовое сообщение.';

  @override
  String get voiceUploadFailed =>
      'Не удалось отправить голосовое. Попробуйте ещё раз.';

  @override
  String get fertilizerCardMore => 'Подробнее';

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
