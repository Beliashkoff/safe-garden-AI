# Mobile — Safe Garden AI

Flutter-приложение для iOS 14+ / Android 8+. Пакет в [`pubspec.yaml`](./pubspec.yaml) называется `agronom_ai` — рабочее название, продуктовое наименование указано в [`../SPEC.md`](../SPEC.md).

State management — Riverpod, навигация — `go_router`, HTTP — `dio`, иммутабельные модели — `freezed`, локализация — `intl` ARB. Полная архитектура — в [`../ARCHITECTURE.md`](../ARCHITECTURE.md) §3, §9.

> Текущее состояние — скелет: два экрана (логин, чат) как заглушки, тема Material 3 (light/dark), локализация EN/RU.

## Prerequisites

| Платформа | Что нужно |
| --------- | --------- |
| Любая    | **Flutter 3.41.9** stable (закреплено в CI), Dart 3.8+. Версия из `pubspec.yaml`: `flutter: ">=3.35.0"`, но локально используем ту же, что и CI — реже surprises. |
| Android  | Android Studio (для Device Manager) + Android SDK с платформой API 26+ (Android 8). Java 17 (если собираем APK локально). |
| iOS      | macOS + Xcode 15+ + CocoaPods. На Windows iOS-разработка **невозможна** — только Android и CI. |

После установки — `flutter doctor` должен показать зелёные галочки для нужных платформ.

## Структура `lib/`

```
lib/
├── main.dart                          # точка входа
├── app/
│   ├── app.dart                       # корневой MaterialApp
│   ├── router.dart                    # go_router конфигурация
│   └── theme.dart                     # Material 3, light/dark
├── features/
│   ├── auth/presentation/             # экран логина (заглушка)
│   └── chat/presentation/             # экран чата (заглушка)
└── l10n/
    ├── app_en.arb                     # English (инфраструктурный, v1 — RU only)
    ├── app_ru.arb                     # Русский
    └── generated/app_localizations.dart  # генерируется автоматически
```

Будущие каталоги по `../ROADMAP.md`:

- `lib/core/api/`, `lib/core/storage/` — общий ApiClient (dio) и SecureTokenStore (Этап 1.5).
- `lib/features/auth/data/` — репозиторий + Riverpod-нотифаер (Этап 1.5).
- `lib/features/chat/data/` — SSE-клиент, локальный кэш через `drift` (Этап 2.4–2.5).

## Зависимости

Из [`pubspec.yaml`](./pubspec.yaml):

| Пакет                 | Зачем |
| --------------------- | ----- |
| `flutter_riverpod`    | State management. Не использовать `setState` в фичах. |
| `go_router`           | Декларативная навигация, deep links. |
| `dio`                 | HTTP-клиент. В будущем — через единый `ApiClient` с интерсепторами (см. CLAUDE.md). |
| `freezed_annotation` + `json_annotation` | Иммутабельные модели + JSON-сериализация. Codegen через `build_runner`. |
| `intl` + `flutter_localizations` | i18n. UI-строки — только через ARB, без хардкода. |
| `cupertino_icons`     | Иконки iOS. |
| `flutter_lints` (dev) | Базовые правила, поверх — строгие в `analysis_options.yaml`. |
| `mocktail` (dev)      | Моки для тестов. |
| `build_runner` + `freezed` + `json_serializable` (dev) | Кодогенерация. |

## Команды

```bash
# Зависимости (после клонирования или после правок pubspec.yaml)
flutter pub get

# Запуск приложения
flutter devices                    # список устройств / эмуляторов
flutter run                        # на дефолтном устройстве
flutter run -d <device-id>         # на конкретном

# Тесты
flutter test                       # unit + widget
flutter test --coverage            # с покрытием в coverage/lcov.info
flutter test integration_test/     # integration (появятся позже)

# Анализ и форматирование
flutter analyze --fatal-infos      # как в CI
dart format lib test               # форматирование (как в CI: dart format --set-exit-if-changed lib test)

# Кодогенерация (freezed / json_serializable)
dart run build_runner build --delete-conflicting-outputs
dart run build_runner watch        # для активной разработки

# Сборка релизных артефактов
flutter build apk --release        # Android APK
flutter build appbundle            # Android AAB (Google Play)
flutter build ios                  # iOS (только macOS)
```

## Локализация

- Строки лежат в [`lib/l10n/app_en.arb`](./lib/l10n/app_en.arb) и [`lib/l10n/app_ru.arb`](./lib/l10n/app_ru.arb).
- Генерация `AppLocalizations` запускается автоматически при `flutter pub get` (`flutter.generate: true` в `pubspec.yaml`, конфиг — в `l10n.yaml`).
- В v1 UI — только RU. EN ARB держим, чтобы инфраструктура локализации была проверена и готова к расширению (см. ROADMAP §6.5).
- Не использовать хардкод-строки в UI — всё через `AppLocalizations.of(context).xxx`.

## Симуляторы и эмуляторы

### Android Emulator

1. Запустить Android Studio → **Device Manager** → **Create Device** → выбрать профиль (Pixel 6 и подобные) → system image на **API 26+** (Android 8.0+).
2. Стартовать AVD из IDE или из CLI:
   ```bash
   flutter emulators                       # список
   flutter emulators --launch <id>
   ```
3. Проверить, что эмулятор виден: `flutter devices`.

### Физическое Android-устройство

1. Settings → About → семь тапов по Build Number → **Developer Options**.
2. Включить **USB debugging**.
3. Подключить по USB, разрешить отладку на устройстве, проверить `flutter devices`.

### iOS Simulator (macOS)

```bash
open -a Simulator                  # запустить Simulator.app
flutter run -d ios                 # или выбрать через flutter devices
```

При первом запуске под iOS Flutter попросит `pod install` — выполнится автоматически, либо вручную из `ios/`.

### Физическое iOS-устройство

Apple Developer Account, signing настраивается в Xcode (`open ios/Runner.xcworkspace`). Полный setup — задача Этапа 7.

## CI

См. [`../.github/workflows/mobile-ci.yml`](../.github/workflows/mobile-ci.yml). Workflow запускается на PR и push в `main`:

| Job                  | Триггер           | Что делает |
| -------------------- | ----------------- | ---------- |
| `analyze`            | PR + push         | `dart format --set-exit-if-changed lib test`, `flutter analyze --fatal-infos`. |
| `test`               | PR + push         | `flutter test --coverage` + artifact `coverage/lcov.info` (7 дней). |
| `build-apk-debug`    | Только PR         | `flutter build apk --debug`, artifact `app-debug.apk` (7 дней). |
| `build-apk-release`  | Только push в main| `flutter build apk --release`, artifact `app-release-unsigned.apk` (14 дней). Не входит в required checks — это post-merge сигнал. |

Перед PR локально полезно прогнать:

```bash
flutter pub get
dart format lib test
flutter analyze --fatal-infos
flutter test
```

## Стиль кода

Жёсткие правила из [`analysis_options.yaml`](./analysis_options.yaml):

- `prefer_const_constructors`, `prefer_const_declarations`, `prefer_const_literals_to_create_immutables`.
- `prefer_single_quotes`, `require_trailing_commas`.
- `avoid_print: error`, `unawaited_futures: error`.
- `strict-casts`, `strict-inference`, `strict-raw-types`.
- `prefer_relative_imports`, `sort_constructors_first`, `use_super_parameters`.

Подробнее про style — в [`../CLAUDE.md`](../CLAUDE.md) §«Стиль кода».

## Troubleshooting

### Flutter SDK на Windows

Установка через `winget` упирается в права в `C:\Program Files` при первом `flutter doctor`. Рекомендуется:

1. Склонировать репозиторий Flutter в путь без пробелов: `git clone https://github.com/flutter/flutter.git -b stable <flutter_root>`.
2. Прописать `<flutter_root>\bin` в `PATH`:
   ```powershell
   # PowerShell, текущая сессия
   $env:Path = "<flutter_root>\bin;$env:Path"
   # Постоянно
   setx PATH "<flutter_root>\bin;$env:Path"
   ```
3. Запустить `flutter --version` и `flutter doctor` для проверки.

Желательно использовать ту же версию, что и CI (`3.41.9`): `git -C <flutter_root> checkout 3.41.9`.

### Android SDK не установлен

`flutter doctor` покажет красный крестик у Android toolchain. Варианты:

- Поставить Android Studio (тянет SDK и emulator) — нужно для локальной сборки APK и запуска на эмуляторе.
- Не ставить и работать только через CI для APK: локально остаются `flutter analyze`, `flutter test`, `flutter pub get`. Reasonable для разработчика, у которого основной таргет — iOS или анализ кода.

### Codegen «out of date»

После правок `@freezed` / `@JsonSerializable` запустить:

```bash
dart run build_runner build --delete-conflicting-outputs
```

Если файлы `*.freezed.dart` / `*.g.dart` дают ошибки — обычно лечится тем же. Они в `.gitignore` не попадают — генерируются заново и коммитятся (см. `analysis_options.yaml` — они исключены из анализа).

### CocoaPods на macOS

Если `flutter run -d ios` падает с «CocoaPods not installed»:

```bash
sudo gem install cocoapods
cd ios && pod install && cd ..
```

### `flutter pub get` зависает

В РФ некоторые соединения к `pub.dev` идут через прокладки, иногда тормозят / тайм-аутят. При проблемах — VPN на момент `pub get`. Постоянный mirror в проекте не настроен.

### Несовпадение версии Flutter с CI

Если `flutter analyze` локально проходит, а CI красный — почти всегда дело в версии. Локальная `flutter --version` должна показывать `3.41.9` stable.

## Связанные документы

- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) — архитектура клиента, состояния, локальный кэш.
- [`../ROADMAP.md`](../ROADMAP.md) — этапы и DoD.
- [`../CLAUDE.md`](../CLAUDE.md) — правила работы Claude Code, стиль кода Flutter.
- [`../SPEC.md`](../SPEC.md) — продуктовая спецификация.
