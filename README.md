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

- **Go** 1.23+
- **Flutter** 3.24+ / Dart 3.5+
- **Docker** + **Docker Compose**
- **Make**
- **Git** — на Windows установить `core.autocrlf=input`, чтобы `.editorconfig` (`end_of_line = lf`) работал корректно:
  ```
  git config --global core.autocrlf input
  ```

iOS-сборка дополнительно требует macOS + Xcode 15+ с CocoaPods.

## Быстрый старт

> **Сейчас репозиторий на этапе 0.1** — только структура. Реальные команды появятся после 0.2 (`make dev` для бэка) и 0.3 (`flutter run` для мобилки). См. `ROADMAP.md`.

После завершения Этапа 0:

```bash
# Backend (локальное окружение через docker-compose)
cd backend
make dev              # postgres + redis + minio + mailhog + air (live-reload)

# Mobile
cd mobile
flutter pub get
flutter run
```

Подробности — в `backend/README.md` и `mobile/README.md` (появятся в 0.5).

## Статус

Этап **0.1 — Структура репозитория** ✅ завершён. В работе — **0.2 Backend skeleton** (см. `ROADMAP.md`).

## Открытые блокеры по этапам

См. `ROADMAP.md` «Критические зависимости». Краткий обзор:

- **Этап 0.6:** Hetzner Cloud аккаунт + Anthropic API-ключ (зарубежная карта/юрлицо), Yandex Cloud организация, Yandex 360 для домена, регистрация домена и DNS.
- **Этап 4:** Yandex SpeechKit ключ.
- **Этап 5:** Каталог удобрений (CSV от заказчика).
- **Этап 7:** Apple Developer Account, Google Play Console, Privacy Policy / ToS на двух языках.
