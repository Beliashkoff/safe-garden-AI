# safe-garden-AI

Мобильное приложение для диагностики проблем растений с помощью Claude Opus и нативной рекомендации удобрений компании.

## Документация

- **[SPEC.md](./SPEC.md)** — продуктовая спецификация: цель, фичи, границы, открытые вопросы.
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** — техническая архитектура: стек, API, БД, безопасность, инфраструктура.
- **[ROADMAP.md](./ROADMAP.md)** — поэтапный план реализации с критериями приёмки.
- **[CLAUDE.md](./CLAUDE.md)** — инструкции для Claude Code.

## Стек (кратко)

- **Mobile:** Flutter (Riverpod, dio, freezed), iOS 14+ / Android 8+
- **Backend:** Go (chi, pgx, sqlc), PostgreSQL, Redis, Yandex Object Storage
- **LLM:** Claude Opus 4.x через **отдельный `llm-worker`** на Hetzner Frankfurt (вне РФ; Anthropic блокирует РФ-AS, поэтому прямой вызов и AWS Bedrock не работают)
- **Email:** Yandex 360 SMTP (OTP-коды)
- **Транскрипция:** Yandex SpeechKit v3 (с конвертацией m4a→OggOpus через `ffmpeg`)
- **Облако:** Yandex Cloud (152-ФЗ, РФ-юрисдикция, PII не покидает РФ)
- **Деплой:** Docker Compose на VM (Yandex Compute + Hetzner CX22). Без Kubernetes в v1.
- **Окружения:** dev (локально) + prod. Stage добавим после релиза в сторы.
- **Сторы:** App Store, Google Play

## Структура репозитория

Монорепо. Полная схема — в `ARCHITECTURE.md` §3.

```
safe-garden-AI/
├── backend/              # Go: HTTP API (РФ) + LLM-worker (Hetzner)
├── mobile/               # Flutter: iOS / Android
├── infra/                # Terraform + Docker Compose для prod
├── .github/workflows/    # CI: backend, mobile, release
├── SPEC.md / ARCHITECTURE.md / ROADMAP.md / CLAUDE.md
```

## Требования к окружению

Для разработки нужно:

- **Go** 1.24+ (см. `backend/go.mod`)
- **Flutter** 3.35+ stable / Dart 3.8+ (см. `mobile/pubspec.yaml`)
- **Docker** + **Docker Compose**
- **Make**
- **Git** — на Windows установить `core.autocrlf=input`, чтобы `.editorconfig` (`end_of_line = lf`) работал корректно:
  ```
  git config --global core.autocrlf input
  ```

iOS-сборка дополнительно требует macOS + Xcode 15+ с CocoaPods.

## Быстрый старт

Скелеты backend и mobile уже подняты. Подробности — в [`backend/README.md`](./backend/README.md) и [`mobile/README.md`](./mobile/README.md).

```bash
# Backend (локальное окружение через docker-compose)
cd backend
cp .env.example .env
docker compose up -d   # postgres + redis + minio + mailhog
go run ./cmd/api       # либо `air` для live-reload

# Mobile
cd mobile
flutter pub get
flutter run
```

## Ветка main и CI

`main` защищена правилом branch protection. Прямые пуши запрещены — изменения попадают через PR с зелёным CI.

CI состоит из двух workflow:

- `.github/workflows/backend-ci.yml` — `lint` (golangci-lint + gofmt) и `test` (`go test -race`) на каждом PR/push в main.
- `.github/workflows/mobile-ci.yml` — `analyze` (`dart format` + `flutter analyze`), `test` (`flutter test`), `build-apk-debug` (на PR), `build-apk-release` (на push в main).

### Настройка branch protection (один раз, владелец репо)

1. Закоммитить и запушить оба workflow → открыть тестовый PR → дождаться первого зелёного прогона (тогда GitHub узнает имена required checks).
2. Settings → Branches → Add branch protection rule. Branch name pattern: `main`. Включить:
   - **Require a pull request before merging**
     - Require approvals: `1` (или `0` для одиночной разработки)
     - Dismiss stale pull request approvals when new commits are pushed
   - **Require status checks to pass before merging**
     - Require branches to be up to date before merging
     - Required status checks:
       - `backend-ci / lint`
       - `backend-ci / test`
       - `mobile-ci / analyze`
       - `mobile-ci / test`
       - `mobile-ci / build-apk-debug`
   - **Require linear history**
   - **Do not allow bypassing the above settings**
3. **Allow force pushes** — выключено.
4. **Allow deletions** — выключено.

`mobile-ci / build-apk-release` сознательно не входит в required checks: он запускается только на push в main и нужен как post-merge сигнал, иначе PR будет вечно в pending.

## Статус

- 0.1 Структура репозитория ✅
- 0.2 Backend skeleton ✅
- 0.3 Mobile skeleton ✅
- 0.4 CI/CD bootstrap ✅
- В работе — **0.5 Документация**. Дальнейшие этапы — в `ROADMAP.md`.

## Открытые блокеры по этапам

См. `ROADMAP.md` «Критические зависимости». Краткий обзор:

- **Этап 0.6:** Hetzner Cloud аккаунт + Anthropic API-ключ (зарубежная карта/юрлицо), Yandex Cloud организация, Yandex 360 для домена, регистрация домена и DNS.
- **Этап 4:** Yandex SpeechKit ключ.
- **Этап 5:** Каталог удобрений (CSV от заказчика).
- **Этап 7:** Apple Developer Account, Google Play Console, Privacy Policy / ToS на двух языках.
