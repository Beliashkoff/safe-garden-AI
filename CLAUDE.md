# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Что за проект

**Safe Garden AI** — мобильное приложение (Flutter) + бэкенд (Go) для диагностики проблем растений с помощью Claude Opus и нативной рекомендации удобрений. Целевая аудитория — РФ/СНГ, продакшн с первой версии.

Подробности — в `SPEC.md` (продукт), `ARCHITECTURE.md` (техническая архитектура), `ROADMAP.md` (этапы реализации). **Перед нетривиальными решениями обязательно сверяйся с этими документами.**

## Состояние

Сейчас в репозитории **только планировочные документы**. Код будет создаваться по этапам из `ROADMAP.md`. Не создавай каталоги `backend/`, `mobile/`, `infra/` без явного запроса — они появятся в Этапе 0.

## Команды

После старта реализации (Этап 0+) будут доступны команды ниже. До этого — только просмотр документации.

```bash
# Backend (после Этапа 0)
cd backend
make dev              # docker-compose: postgres + redis + minio + air (cmd/api)
make worker-dev       # запуск llm-worker'а локально на :8081 (cmd/llmworker)
make test             # go test ./...
make test-integration # с testcontainer Postgres
make lint             # golangci-lint
make migrate-up       # goose миграции
make sqlc-gen         # перегенерация sqlc

# Mobile (после Этапа 0)
cd mobile
flutter pub get
flutter run
flutter test
flutter test integration_test/
dart run build_runner build --delete-conflicting-outputs   # для freezed/json_serializable
```

## Архитектурные инварианты

Эти правила вытекают из `ARCHITECTURE.md` и продуктовых ограничений. Нарушать без явной просьбы пользователя — нельзя.

1. **Один чат на пользователя в v1.** Уникальный индекс `conversations(user_id)`. Не вводить `chat_id` в API.
2. **Все приватные эндпоинты под `RequireAuth`.** Каждый запрос к ресурсу должен проверять `user_id`-владельца.
3. **Никогда не логировать содержимое:** текст сообщений, фото-байты, аудио, email, OTP, токены. `slog` Replacer фильтрует поля; добавь новое чувствительное поле в фильтр.
4. **Медиа-файлы загружаются через presigned PUT напрямую в Object Storage**, бэк не проксирует загрузку.
5. **Anthropic недоступен из РФ напрямую и через AWS Bedrock.** Все вызовы Claude идут через `llm-worker` на Hetzner Frankfurt (вне РФ). РФ-бэкенд обращается только к worker'у через mTLS (`internal/llm/worker_client.go`). В worker НЕ передавать email, реальный UUID, refresh-токены — только `uid_hash = sha256(uid + UID_HASH_PEPPER)`. `anthropic-sdk-go` импортируется только в `internal/llmworker/`, не в основном бэкенде.
6. **Refresh-токены — opaque и хэшируются** (sha256). JWT — только access. Ротация при каждом refresh.
7. **Apple Sign-In обязателен** при наличии Google (App Store §4.8).
8. **Удаление аккаунта** — каскад в БД + асинхронное удаление префикса `u/{user_id}/` в Object Storage. Без этого Apple/Google ревью не пропустит.
9. **Стриминг ответа Claude — через SSE**, не WebSocket.
10. **PII (email, имя) — в РФ.** Postgres в Yandex Cloud. Anthropic вызывается из иностранного юрлица, в Anthropic не уходит email пользователя — только хэш user_id в `metadata`.

## Стиль кода

### Backend (Go)
- Идиоматичный Go: маленькие интерфейсы определяются на стороне потребителя (`internal/usecase`), не на стороне реализации.
- Слои: `transport/http` → `usecase` → `domain` (интерфейсы) ← `storage` (реализации). Зависимости только сверху вниз.
- SQL только через sqlc. Никаких `database/sql` руками в `usecase`.
- Ошибки оборачиваем через `fmt.Errorf("...: %w", err)`. На границе HTTP — мапим в `error_response.go`.
- `context.Context` — первым аргументом во всех функциях, делающих I/O.
- Никаких глобальных переменных, кроме `slog.Default()`.
- Тесты: `testify` + табличные. Моки — только на внешние границы (Anthropic, S3, SMTP, OIDC).

### Mobile (Flutter / Dart)
- Иммутабельные модели через `freezed`.
- State management — `riverpod`. Не использовать `setState` в фичах (только во внутренних виджетах).
- Все строки UI — через `intl` ARB. Никакого хардкода.
- Цвета и типографика — только из `Theme`, не литералы.
- HTTP — только через единый `ApiClient` (dio + интерсепторы). Ни один экран не использует dio напрямую.
- Запрещены `print()` и `unawaited futures` без явной причины (linter настроен).

### Общее
- Без эмодзи в коде/комментариях, если пользователь не попросил.
- Комментарии — только когда «почему» неочевидно. «Что делает код» уже видно из имени.
- Никаких бэккомпат-шимов без необходимости — мы только начинаем.

## Что обязательно сверять через ctx7

Анти-паттерн — писать код на основе памяти модели. Перед использованием любой библиотеки **получай актуальную документацию через `ctx7` CLI** (см. глобальные правила в `~/.claude/rules/context7.md`). Особенно критично для:

- `anthropic-sdk-go` — модельный ID (Opus 4.x), формат streaming events, prompt caching, tool use multi-turn loop. SDK активно меняется. Используется только в `internal/llmworker/`.
- `sign_in_with_apple`, `google_sign_in` — версии и breaking changes частые.
- `sqlc` — конфиг и pgx-плагин.
- `pgx/v5` — pool config, transactions.
- `riverpod` — современный API (2.x с Notifier vs StateNotifier).
- `dio` — SSE/streaming через ResponseType.
- `Yandex SpeechKit v3` — sync recognize API, форматы (LPCM/OggOpus/MP3, не m4a), коды ошибок, лимит 30s sync / streaming для длиннее.
- `gomail` (или альтернатива) — SMTP через Yandex 360 (`smtp.yandex.ru:465` SSL, AUTH LOGIN).
- `ffmpeg` — параметры кодека Opus для конвертации m4a→OggOpus 16kHz mono.

## Безопасность — короткий чек-лист перед PR

- [ ] Новый эндпоинт под `RequireAuth` (если не auth/health).
- [ ] Если работает с `storage_key` или `user_data` — проверена `user_id` ownership.
- [ ] Никаких секретов в коде, только env.
- [ ] Логи не содержат PII/контент сообщений.
- [ ] Лимиты payload и rate limit учтены.
- [ ] Миграция имеет корректный `down`-блок (но в проде применяется только `up`).
- [ ] Если меняется API — обновлены и mobile, и тесты.

## Полезные ссылки внутри репо

- `SPEC.md` §4 — границы (что делаем, что нет, никогда).
- `ARCHITECTURE.md` §4 — API контракты.
- `ARCHITECTURE.md` §6 — схема БД.
- `ARCHITECTURE.md` §8 — безопасность.
- `ARCHITECTURE.md` §11 — доступ к Anthropic из РФ.
- `ROADMAP.md` — текущий этап и DoD.

## Язык

- **Документация** (SPEC, ARCHITECTURE, ROADMAP, README) — русский.
- **Код, имена файлов, идентификаторы, коммиты** — английский.
- **Сообщения коммитов** — Conventional Commits (`feat:`, `fix:`, `chore:` ...) на английском.
- **UI-строки приложения** — русский (через `intl`).

## Важно

- Не создавать новые `*.md` без явной просьбы пользователя.
- Не предлагать MVP-упрощения, не согласованные с заказчиком — `SPEC.md` §4.3 фиксирует, что просим перед изменениями скоупа.
- При сомнении в технической детали — открыть соответствующую секцию `ARCHITECTURE.md`. Если там нет ответа — задать вопрос пользователю, а не додумывать.
