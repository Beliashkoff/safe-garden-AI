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

## Статус

Стадия планирования. Код будет создаваться согласно `ROADMAP.md`, начиная с **Этапа 0 — Фундамент**.

## Открытые блокеры до старта Этапа 0

См. `ROADMAP.md` «Критические зависимости». Активные:
- Hetzner Cloud аккаунт + Anthropic API-ключ (зарубежная карта/юрлицо заказчика).
- Yandex Cloud организация и биллинг.
- Yandex 360 для домена.
- Регистрация домена и DNS.

Отложенные (ко вторым этапам):
- Каталог удобрений (Этап 5).
- Privacy Policy / ToS (Этап 7).
- Apple Developer Account и Google Play Console (Этап 7).
