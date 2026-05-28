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

  /// No description provided for @loginButtonApple.
  ///
  /// In ru, this message translates to:
  /// **'Войти с Apple'**
  String get loginButtonApple;

  /// No description provided for @loginButtonGoogle.
  ///
  /// In ru, this message translates to:
  /// **'Войти с Google'**
  String get loginButtonGoogle;

  /// No description provided for @loginButtonEmail.
  ///
  /// In ru, this message translates to:
  /// **'Войти по email'**
  String get loginButtonEmail;

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
