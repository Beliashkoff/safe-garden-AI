import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_ru.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'generated/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations? of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations);
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('ru'),
  ];

  /// No description provided for @appTitle.
  ///
  /// In ru, this message translates to:
  /// **'ИИ Агроном'**
  String get appTitle;

  /// No description provided for @loginTitle.
  ///
  /// In ru, this message translates to:
  /// **'Войти'**
  String get loginTitle;

  /// No description provided for @loginSubtitle.
  ///
  /// In ru, this message translates to:
  /// **'Диагностика растений с помощью AI'**
  String get loginSubtitle;

  /// No description provided for @loginButtonVk.
  ///
  /// In ru, this message translates to:
  /// **'Войти с VK ID'**
  String get loginButtonVk;

  /// No description provided for @loginButtonYandex.
  ///
  /// In ru, this message translates to:
  /// **'Войти с Яндекс ID'**
  String get loginButtonYandex;

  /// No description provided for @loginButtonEmail.
  ///
  /// In ru, this message translates to:
  /// **'Войти по email'**
  String get loginButtonEmail;

  /// No description provided for @consentPrefix.
  ///
  /// In ru, this message translates to:
  /// **'Продолжая, вы принимаете'**
  String get consentPrefix;

  /// No description provided for @consentPrivacy.
  ///
  /// In ru, this message translates to:
  /// **'Политику конфиденциальности'**
  String get consentPrivacy;

  /// No description provided for @consentAnd.
  ///
  /// In ru, this message translates to:
  /// **'и'**
  String get consentAnd;

  /// No description provided for @consentTerms.
  ///
  /// In ru, this message translates to:
  /// **'Условия использования'**
  String get consentTerms;

  /// No description provided for @onboardingTitle1.
  ///
  /// In ru, this message translates to:
  /// **'Сфотографируйте растение'**
  String get onboardingTitle1;

  /// No description provided for @onboardingBody1.
  ///
  /// In ru, this message translates to:
  /// **'Сделайте снимок проблемного места — ИИ-агроном определит, что не так.'**
  String get onboardingBody1;

  /// No description provided for @onboardingTitle2.
  ///
  /// In ru, this message translates to:
  /// **'Опишите голосом или текстом'**
  String get onboardingTitle2;

  /// No description provided for @onboardingBody2.
  ///
  /// In ru, this message translates to:
  /// **'Расскажите о симптомах удобным способом — поддерживаются текст и голос.'**
  String get onboardingBody2;

  /// No description provided for @onboardingTitle3.
  ///
  /// In ru, this message translates to:
  /// **'Получите рекомендации'**
  String get onboardingTitle3;

  /// No description provided for @onboardingBody3.
  ///
  /// In ru, this message translates to:
  /// **'Точный диагноз и подбор удобрений под ваше растение.'**
  String get onboardingBody3;

  /// No description provided for @onboardingSkip.
  ///
  /// In ru, this message translates to:
  /// **'Пропустить'**
  String get onboardingSkip;

  /// No description provided for @onboardingNext.
  ///
  /// In ru, this message translates to:
  /// **'Далее'**
  String get onboardingNext;

  /// No description provided for @onboardingStart.
  ///
  /// In ru, this message translates to:
  /// **'Начать'**
  String get onboardingStart;

  /// No description provided for @emailRequestTitle.
  ///
  /// In ru, this message translates to:
  /// **'Вход по email'**
  String get emailRequestTitle;

  /// No description provided for @emailRequestHint.
  ///
  /// In ru, this message translates to:
  /// **'Введите email — мы пришлём код для входа'**
  String get emailRequestHint;

  /// No description provided for @emailFieldLabel.
  ///
  /// In ru, this message translates to:
  /// **'Email'**
  String get emailFieldLabel;

  /// No description provided for @emailRequestCta.
  ///
  /// In ru, this message translates to:
  /// **'Получить код'**
  String get emailRequestCta;

  /// No description provided for @emailVerifyTitle.
  ///
  /// In ru, this message translates to:
  /// **'Введите код'**
  String get emailVerifyTitle;

  /// No description provided for @emailVerifyHint.
  ///
  /// In ru, this message translates to:
  /// **'Код отправлен на {email}'**
  String emailVerifyHint(String email);

  /// No description provided for @codeFieldLabel.
  ///
  /// In ru, this message translates to:
  /// **'Код из 6 цифр'**
  String get codeFieldLabel;

  /// No description provided for @emailVerifyCta.
  ///
  /// In ru, this message translates to:
  /// **'Войти'**
  String get emailVerifyCta;

  /// No description provided for @resendCode.
  ///
  /// In ru, this message translates to:
  /// **'Отправить код повторно'**
  String get resendCode;

  /// No description provided for @errorInvalidEmail.
  ///
  /// In ru, this message translates to:
  /// **'Некорректный email'**
  String get errorInvalidEmail;

  /// No description provided for @errorInvalidCode.
  ///
  /// In ru, this message translates to:
  /// **'Неверный или истёкший код'**
  String get errorInvalidCode;

  /// No description provided for @errorTooManyAttempts.
  ///
  /// In ru, this message translates to:
  /// **'Слишком много попыток. Запросите новый код.'**
  String get errorTooManyAttempts;

  /// No description provided for @errorRateLimited.
  ///
  /// In ru, this message translates to:
  /// **'Слишком часто. Попробуйте позже.'**
  String get errorRateLimited;

  /// No description provided for @errorNetwork.
  ///
  /// In ru, this message translates to:
  /// **'Нет соединения. Проверьте интернет.'**
  String get errorNetwork;

  /// No description provided for @errorGeneric.
  ///
  /// In ru, this message translates to:
  /// **'Что-то пошло не так. Попробуйте ещё раз.'**
  String get errorGeneric;

  /// No description provided for @chatTitle.
  ///
  /// In ru, this message translates to:
  /// **'Чат'**
  String get chatTitle;

  /// No description provided for @chatEmptyHint.
  ///
  /// In ru, this message translates to:
  /// **'Сфотографируйте растение или опишите проблему'**
  String get chatEmptyHint;

  /// No description provided for @chatInputPlaceholder.
  ///
  /// In ru, this message translates to:
  /// **'Сообщение'**
  String get chatInputPlaceholder;

  /// No description provided for @chatLogout.
  ///
  /// In ru, this message translates to:
  /// **'Выйти'**
  String get chatLogout;

  /// No description provided for @chatDeleteAccount.
  ///
  /// In ru, this message translates to:
  /// **'Удалить аккаунт'**
  String get chatDeleteAccount;

  /// No description provided for @chatSend.
  ///
  /// In ru, this message translates to:
  /// **'Отправить'**
  String get chatSend;

  /// No description provided for @chatStop.
  ///
  /// In ru, this message translates to:
  /// **'Стоп'**
  String get chatStop;

  /// No description provided for @chatRetry.
  ///
  /// In ru, this message translates to:
  /// **'Повторить'**
  String get chatRetry;

  /// No description provided for @chatCancelledNote.
  ///
  /// In ru, this message translates to:
  /// **'Ответ остановлен'**
  String get chatCancelledNote;

  /// No description provided for @chatDeleteMessageConfirm.
  ///
  /// In ru, this message translates to:
  /// **'Удалить это сообщение?'**
  String get chatDeleteMessageConfirm;

  /// No description provided for @chatErrorNetwork.
  ///
  /// In ru, this message translates to:
  /// **'Нет соединения. Проверьте интернет.'**
  String get chatErrorNetwork;

  /// No description provided for @chatErrorRateLimited.
  ///
  /// In ru, this message translates to:
  /// **'Слишком много сообщений. Подождите немного.'**
  String get chatErrorRateLimited;

  /// No description provided for @chatErrorUnsupported.
  ///
  /// In ru, this message translates to:
  /// **'Этот тип содержимого пока не поддерживается.'**
  String get chatErrorUnsupported;

  /// No description provided for @chatErrorTooLarge.
  ///
  /// In ru, this message translates to:
  /// **'Сообщение слишком длинное.'**
  String get chatErrorTooLarge;

  /// No description provided for @chatErrorGeneric.
  ///
  /// In ru, this message translates to:
  /// **'Не удалось получить ответ. Попробуйте ещё раз.'**
  String get chatErrorGeneric;

  /// No description provided for @chatAttach.
  ///
  /// In ru, this message translates to:
  /// **'Прикрепить фото'**
  String get chatAttach;

  /// No description provided for @chatAttachCamera.
  ///
  /// In ru, this message translates to:
  /// **'Камера'**
  String get chatAttachCamera;

  /// No description provided for @chatAttachGallery.
  ///
  /// In ru, this message translates to:
  /// **'Галерея'**
  String get chatAttachGallery;

  /// No description provided for @chatPermissionCameraTitle.
  ///
  /// In ru, this message translates to:
  /// **'Нужен доступ к камере'**
  String get chatPermissionCameraTitle;

  /// No description provided for @chatPermissionCameraBody.
  ///
  /// In ru, this message translates to:
  /// **'Разрешите доступ к камере в настройках, чтобы сфотографировать растение.'**
  String get chatPermissionCameraBody;

  /// No description provided for @chatPermissionPhotosTitle.
  ///
  /// In ru, this message translates to:
  /// **'Нужен доступ к фото'**
  String get chatPermissionPhotosTitle;

  /// No description provided for @chatPermissionPhotosBody.
  ///
  /// In ru, this message translates to:
  /// **'Разрешите доступ к фото в настройках, чтобы прикрепить снимок.'**
  String get chatPermissionPhotosBody;

  /// No description provided for @chatOpenSettings.
  ///
  /// In ru, this message translates to:
  /// **'Открыть настройки'**
  String get chatOpenSettings;

  /// No description provided for @chatMaxPhotos.
  ///
  /// In ru, this message translates to:
  /// **'Можно прикрепить до {max} фото'**
  String chatMaxPhotos(int max);

  /// No description provided for @chatUploadFailed.
  ///
  /// In ru, this message translates to:
  /// **'Не удалось загрузить фото. Попробуйте ещё раз.'**
  String get chatUploadFailed;

  /// No description provided for @chatRemovePhoto.
  ///
  /// In ru, this message translates to:
  /// **'Удалить фото'**
  String get chatRemovePhoto;

  /// No description provided for @voiceHoldToRecord.
  ///
  /// In ru, this message translates to:
  /// **'Удерживайте кнопку для записи'**
  String get voiceHoldToRecord;

  /// No description provided for @voiceSlideToCancel.
  ///
  /// In ru, this message translates to:
  /// **'Влево — отмена'**
  String get voiceSlideToCancel;

  /// No description provided for @voiceReleaseToCancel.
  ///
  /// In ru, this message translates to:
  /// **'Отпустите для отмены'**
  String get voiceReleaseToCancel;

  /// No description provided for @voiceTranscribing.
  ///
  /// In ru, this message translates to:
  /// **'Расшифровка…'**
  String get voiceTranscribing;

  /// No description provided for @voicePermissionTitle.
  ///
  /// In ru, this message translates to:
  /// **'Нужен доступ к микрофону'**
  String get voicePermissionTitle;

  /// No description provided for @voicePermissionBody.
  ///
  /// In ru, this message translates to:
  /// **'Разрешите доступ к микрофону в настройках, чтобы записать голосовое сообщение.'**
  String get voicePermissionBody;

  /// No description provided for @voiceUploadFailed.
  ///
  /// In ru, this message translates to:
  /// **'Не удалось отправить голосовое. Попробуйте ещё раз.'**
  String get voiceUploadFailed;

  /// No description provided for @fertilizerCardMore.
  ///
  /// In ru, this message translates to:
  /// **'Подробнее'**
  String get fertilizerCardMore;

  /// No description provided for @deleteAccountConfirmTitle.
  ///
  /// In ru, this message translates to:
  /// **'Удалить аккаунт?'**
  String get deleteAccountConfirmTitle;

  /// No description provided for @deleteAccountConfirmBody.
  ///
  /// In ru, this message translates to:
  /// **'Это действие необратимо. Все данные будут удалены.'**
  String get deleteAccountConfirmBody;

  /// No description provided for @commonCancel.
  ///
  /// In ru, this message translates to:
  /// **'Отмена'**
  String get commonCancel;

  /// No description provided for @commonDelete.
  ///
  /// In ru, this message translates to:
  /// **'Удалить'**
  String get commonDelete;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'ru'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'ru':
      return AppLocalizationsRu();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
