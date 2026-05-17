# ARCHITECTURE — Safe Garden AI

Техническая архитектура. Дополняет `SPEC.md` (продуктовое «что/зачем») и `ROADMAP.md` (последовательность работ).

---

## 1. Высокоуровневая схема

```
                                    ┌──────────────────────┐
                                    │   Apple ID Provider  │
                                    │   Google Identity    │
                                    └──────────┬───────────┘
                                               │ id_token
                                               ▼
┌───────────────────┐    HTTPS/SSE    ┌──────────────────────┐
│  Mobile (Flutter) │ ──────────────▶ │  Go Backend (chi)    │
│  iOS / Android    │ ◀────────────── │  Yandex Cloud (РФ)   │
└───┬───────────────┘                 └──┬─────┬───────┬─────┘
    │ PUT (presigned)                    │     │       │
    ▼                                    ▼     ▼       │ mTLS, JSON
┌──────────────────┐         ┌─────────────┐ ┌─────────────┐
│ Object Storage   │         │ PostgreSQL  │ │   Redis     │
│ (Yandex S3-like) │         │ users, msgs │ │ rate-limit  │
│ photos, audio    │         │   (PII)     │ │   cache     │
└──────────────────┘         └─────────────┘ └─────────────┘
                                                  │
                                                  │ mTLS over HTTPS
                                                  ▼
                                       ┌──────────────────────┐         ┌──────────────────┐
                                       │  llm-worker          │ HTTPS   │  Anthropic API   │
                                       │  HostKey Frankfurt   │────────▶│  Claude Opus 4.x │
                                       │  (вне РФ, без PII)   │◀────────│                  │
                                       └──────────────────────┘         └──────────────────┘
```

> **Важно:** Anthropic API недоступен из РФ напрямую и через AWS Bedrock — см. §11. Все вызовы Claude проксируются через `llm-worker` на VPS вне РФ.

---

## 2. Технологический стек

### 2.1 Mobile (Flutter)

| Слой / задача                | Пакет / инструмент                          |
| ---------------------------- | ------------------------------------------- |
| Фреймворк                    | Flutter (стабильный, ≥ 3.24)                |
| Язык                         | Dart 3.x                                    |
| State management             | `riverpod` (`flutter_riverpod`)             |
| Навигация                    | `go_router`                                 |
| Иммутабельные модели         | `freezed` + `json_serializable`             |
| HTTP                         | `dio` (с интерсепторами auth/refresh)       |
| SSE                          | `dio` `ResponseType.stream` + кастомный парсер |
| Безопасное хранилище         | `flutter_secure_storage`                    |
| Локальный кэш истории        | `drift` (SQLite через типизированный DSL)   |
| Apple Sign-In                | `sign_in_with_apple`                        |
| Google Sign-In               | `google_sign_in`                            |
| Камера / галерея             | `image_picker`                              |
| Сжатие фото                  | `flutter_image_compress`                    |
| Запись аудио                 | `record`                                    |
| Воспроизведение              | `just_audio`                                |
| Разрешения                   | `permission_handler`                        |
| Локализация                  | `intl` + сгенерированные ARB                |
| Логи                         | `logger` (с фильтрацией PII)                |
| Sentry                       | `sentry_flutter`                            |

### 2.2 Backend (Go)

| Слой / задача                | Пакет / инструмент                                   |
| ---------------------------- | ---------------------------------------------------- |
| Версия Go                    | 1.23+                                                |
| HTTP роутер                  | `github.com/go-chi/chi/v5`                           |
| Postgres драйвер             | `github.com/jackc/pgx/v5/pgxpool`                    |
| SQL-генерация                | `sqlc` (`sqlc.yaml`, `internal/storage/postgres/queries`) |
| Миграции                     | `goose` (или `atlas` если нужны декларативные)       |
| Конфиг                       | `github.com/kelseyhightower/envconfig`               |
| Логи                         | `log/slog` (stdlib, JSON-handler в проде)            |
| Валидация                    | `github.com/go-playground/validator/v10`             |
| JWT                          | `github.com/golang-jwt/jwt/v5` (RS256)               |
| OAuth верификация            | `github.com/coreos/go-oidc/v3` для Apple/Google JWKS |
| Anthropic SDK                | `github.com/anthropics/anthropic-sdk-go`             |
| AWS S3 (для Yandex Storage)  | `github.com/aws/aws-sdk-go-v2`                       |
| Redis                        | `github.com/redis/go-redis/v9`                       |
| Rate limit                   | `github.com/go-redis/redis_rate/v10`                 |
| Sentry                       | `github.com/getsentry/sentry-go`                     |
| Метрики                      | `github.com/prometheus/client_golang`                |
| Тесты                        | `github.com/stretchr/testify`, `testcontainers-go`  |
| Live reload (dev)            | `github.com/cosmtrek/air`                            |
| Linter                       | `golangci-lint`                                      |
| OpenAPI генерация            | `github.com/getkin/kin-openapi` (опционально)        |

### 2.3 Инфраструктура

| Компонент                | Решение                                              |
| ------------------------ | ---------------------------------------------------- |
| Облако (РФ)              | Yandex Cloud (РФ-юрисдикция, 152-ФЗ)                 |
| Compute (РФ)             | Yandex Compute Cloud — **одна VM** (s-standard, 2 vCPU / 4 GB на старте) с Docker Compose. Миграция на Managed K8s — после v1, при росте. |
| База данных              | **Yandex Managed PostgreSQL** (managed-сервис, не в Docker — для бэкапов/HA из коробки) |
| Object Storage           | Yandex Object Storage (S3 API)                       |
| Redis                    | **Yandex Managed Redis** (managed-сервис)            |
| Контейнеры               | **Docker + Docker Compose** на каждой VM (dev и prod). Без K8s в v1.   |
| Reverse proxy / TLS      | **Caddy** на каждой VM (auto-HTTPS через Let's Encrypt, минимум конфига) |
| CDN                      | Yandex Cloud CDN (для статики, аватары)              |
| DNS                      | Yandex Cloud DNS                                     |
| CI/CD                    | GitHub Actions → push в Yandex Container Registry → деплой |
| Мониторинг               | Prometheus + Grafana (или Yandex Monitoring)         |
| Логи                     | Yandex Cloud Logging (или Loki self-hosted)          |
| Error tracking           | Sentry self-hosted (или sentry.io по согласованию)   |
| Secrets                  | Yandex Lockbox + переменные окружения                |
| IaC                      | Terraform (`infra/terraform/`)                       |
| Email (OTP)              | **Yandex 360 SMTP** (рабочий ящик `noreply@<domain>`)|
| Транскрипция             | **Yandex SpeechKit v3** (sync recognize)             |
| LLM-worker (вне РФ)      | **HostKey Frankfurt** (vm.v2-nano, Ubuntu 24.04, ~830 ₽/мес). Оплата рублями, юрлицо провайдера — ООО «АЙТИБ» (РФ); принятые риски — см. §11.7. |
| Конвертация аудио        | `ffmpeg` (m4a/AAC → OggOpus 16kHz mono для SpeechKit)|

---

## 3. Структура репозитория

Монорепо в одном Git-репозитории — упрощает координацию изменений API между бэком и мобилкой.

```
safe-garden-AI/
├── SPEC.md                          # продуктовая спецификация
├── ARCHITECTURE.md                  # этот файл
├── ROADMAP.md                       # дорожная карта
├── CLAUDE.md                        # инструкции для Claude Code
├── README.md                        # быстрый старт
├── .editorconfig
├── .github/
│   └── workflows/
│       ├── backend-ci.yml
│       ├── mobile-ci.yml
│       └── release.yml
├── backend/
│   ├── cmd/
│   │   ├── api/                     # HTTP-сервер (РФ, Yandex Cloud)
│   │   ├── llmworker/               # LLM-микросервис (вне РФ, HostKey Frankfurt)
│   │   ├── seed/                    # сидер каталога удобрений
│   │   └── migrate/                 # запуск миграций
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/                  # модели + интерфейсы (порты)
│   │   ├── usecase/                 # бизнес-логика
│   │   │   ├── auth/
│   │   │   ├── chat/
│   │   │   ├── upload/
│   │   │   └── account/
│   │   ├── transport/
│   │   │   └── http/
│   │   │       ├── handlers/
│   │   │       ├── middleware/
│   │   │       └── router.go
│   │   ├── storage/
│   │   │   ├── postgres/            # репозитории + sqlc-сгенерированный код
│   │   │   │   ├── queries/         # *.sql файлы для sqlc
│   │   │   │   └── *.go
│   │   │   ├── s3/
│   │   │   └── redis/
│   │   ├── llm/                     # интерфейс клиента + реализации
│   │   │   ├── client.go            # interface Client
│   │   │   ├── worker_client.go     # mTLS RPC к llm-worker (default)
│   │   │   ├── openrouter_client.go # резерв
│   │   │   ├── mock_client.go       # для dev/тестов
│   │   │   ├── prompts/
│   │   │   ├── tools/
│   │   │   └── stream.go
│   │   ├── llmworker/               # код LLM-worker'а (только там используется anthropic-sdk-go)
│   │   │   ├── server.go
│   │   │   ├── anthropic.go
│   │   │   └── tool_callback.go     # обратные RPC в РФ-бэк
│   │   ├── auth/                    # JWT, OIDC верификация
│   │   ├── audio/                   # транскрипция
│   │   └── observability/           # logging, metrics, tracing
│   ├── migrations/                  # *.sql goose
│   ├── go.mod
│   ├── go.sum
│   ├── Makefile
│   ├── Dockerfile
│   ├── docker-compose.yml           # локальная разработка
│   ├── .air.toml
│   ├── .golangci.yml
│   └── sqlc.yaml
├── mobile/
│   ├── lib/
│   │   ├── main.dart
│   │   ├── app/                     # MaterialApp, theme, router
│   │   ├── features/
│   │   │   ├── auth/
│   │   │   │   ├── data/
│   │   │   │   ├── domain/
│   │   │   │   └── presentation/
│   │   │   └── chat/
│   │   │       ├── data/
│   │   │       ├── domain/
│   │   │       └── presentation/
│   │   ├── core/
│   │   │   ├── api/                 # dio, interceptors, sse parser
│   │   │   ├── storage/             # secure_storage, drift
│   │   │   ├── permissions/
│   │   │   └── errors/
│   │   ├── widgets/                 # переиспользуемые виджеты
│   │   └── l10n/                    # ARB файлы
│   ├── test/
│   ├── integration_test/
│   ├── android/
│   ├── ios/
│   ├── pubspec.yaml
│   └── analysis_options.yaml
└── infra/
    ├── terraform/
    │   ├── envs/
    │   │   ├── stage/
    │   │   └── prod/
    │   └── modules/
    ├── helm/                        # для K8s, после миграции с Compute
    └── docker/
        └── compose/
```

---

## 4. API контракты

Все эндпоинты под `https://api.<domain>/v1/`. JSON в теле запроса/ответа, кроме SSE (`text/event-stream`) и multipart/PUT в Object Storage.

Авторизация: `Authorization: Bearer <access_token>` (JWT).

### 4.1 Аутентификация

```
POST /v1/auth/apple
  body:    { id_token: string, nonce: string }
  resp:    { access_token, refresh_token, user: { id, email?, display_name? } }

POST /v1/auth/google
  body:    { id_token: string }
  resp:    { access_token, refresh_token, user }

POST /v1/auth/email/request
  body:    { email: string }
  resp:    204

POST /v1/auth/email/verify
  body:    { email: string, code: string }     # 6 цифр, ttl 10min
  resp:    { access_token, refresh_token, user }

POST /v1/auth/refresh
  body:    { refresh_token: string }
  resp:    { access_token, refresh_token }     # ротация refresh

POST /v1/auth/logout
  body:    { refresh_token: string }
  resp:    204
```

### 4.2 Аккаунт

```
GET    /v1/account                          # текущий пользователь
DELETE /v1/account                          # каскадное удаление
```

### 4.3 Чат

```
GET  /v1/conversation
  resp: { id, messages: [Message], next_cursor? }

GET  /v1/conversation/messages?cursor=...&limit=50
  resp: { messages: [Message], next_cursor? }

POST /v1/messages
  body: {
    content: [
      { type: "text", text: string },
      { type: "image_ref", storage_key: string },
      { type: "audio_ref", storage_key: string }   # будет транскрибирован
    ]
  }
  resp: text/event-stream (SSE), события:
    event: message_started   data: { message_id }
    event: delta             data: { text: "..." }
    event: tool_use          data: { tool: "recommend_fertilizer", args: {...} }
    event: fertilizer_card   data: { products: [...] }
    event: error             data: { code, message }
    event: done              data: { message_id, tokens_used: { in, out } }

DELETE /v1/messages/:id                    # удалить только своё сообщение
```

### 4.4 Загрузки

```
POST /v1/uploads/presign
  body: { content_type: "image/jpeg" | "audio/m4a", size_bytes: int, purpose: "image" | "audio" }
  resp: { url: string, key: string, headers: {...}, expires_at }

# далее клиент делает PUT по url напрямую в Object Storage
```

### 4.5 Аудио (внутренний эндпоинт, вызывается при `audio_ref` в /messages)

Если решено выставить отдельный эндпоинт:

```
POST /v1/audio/transcribe
  body: { storage_key: string, language?: "ru" | "en" }
  resp: { text: string, duration_ms: int }
```

### 4.6 Каталог удобрений (служебный, для Claude tools)

Внутренний — не выставляется наружу. Запрос идёт из сервера к собственной БД, см. §6.4.

### 4.7 Формат ошибок

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Field `email` is invalid",
    "details": { "field": "email" }
  },
  "request_id": "req_abc123"
}
```

Коды: `unauthorized`, `forbidden`, `validation_failed`, `not_found`, `rate_limited`, `payload_too_large`, `unsupported_media_type`, `internal_error`.

---

## 5. Поток «фото + текст → ответ»

1. **Mobile:** пользователь выбирает фото → `flutter_image_compress` сжимает до ≤ 1920×1080 q=85.
2. **Mobile → Backend:** `POST /v1/uploads/presign` с `content_type` и `size_bytes`. Бэк проверяет лимит (10 МБ), генерирует уникальный `key = u/{user_id}/img/{ulid}.jpg`, возвращает presigned PUT URL (TTL 5 мин).
3. **Mobile → Object Storage:** прямой `PUT` (минимизация трафика на бэке). После 200 OK клиент знает `key`.
4. **Mobile → Backend:** `POST /v1/messages` с `content: [{type:"text",...},{type:"image_ref", storage_key: "u/.../img/...jpg"}]`.
5. **Backend (РФ):**
   - Проверяет: `user_id` владельца ключа совпадает с авторизованным.
   - Берёт `conversation` пользователя (одна на user).
   - Сохраняет `message` (role=`user`) + `message_blocks` в Postgres.
   - Загружает фото из Object Storage (S3 GetObject), кодирует в base64. Конвертирует HEIC → JPEG при необходимости.
   - Подгружает последние N сообщений (по умолчанию 20) как контекст.
   - Формирует payload для **`llm-worker`** (НЕ для Anthropic напрямую): `{ messages, system, tools, model, stream: true, metadata: { uid_hash } }`. Email и реальный UUID не передаются, только `uid_hash = sha256(uid + secret_pepper)`.
   - Делает `POST /v1/llm/messages` на worker по mTLS, открывает SSE-стрим.
6. **LLM-worker (HostKey Frankfurt):**
   - Получает payload, вызывает `client.Messages.NewStreaming(...)` через `anthropic-sdk-go`.
   - Параметры: `system` (см. §7), `messages` (история + текущее), `tools: [recommend_fertilizer]`, `model: claude-opus-4-7` (актуальный ID — проверить через ctx7 при имплементации), `cache_control: ephemeral` для system+tools.
   - При `tool_use` от Claude (`recommend_fertilizer`) — делает обратный RPC в РФ-бэкенд (`POST /internal/v1/tools/fertilizer` по mTLS), получает результат из БД каталога, передаёт в Claude как `tool_result`, продолжает стрим.
   - Ретранслирует SSE обратно в РФ-бэкенд.
7. **Backend (РФ):** ретранслирует SSE на mobile. По `stop_reason: end_turn` — сохраняет финальное `assistant`-сообщение, блоки и `usage_log` (токены, стоимость).
8. **Mobile:** парсит SSE, отрисовывает дельты, при событии `fertilizer_card` рендерит нативный блок-карточку.

### Отмена

- Клиент закрывает SSE-соединение.
- Бэк через `context.Cancel` закрывает SSE к worker'у; worker через свой `context.Cancel` прерывает запрос к Claude.
- РФ-бэк сохраняет частичный assistant-message со статусом `cancelled` (или не сохраняет, если ничего не пришло).

---

## 6. Модель данных (PostgreSQL)

### 6.1 Таблицы

```sql
-- users
CREATE TABLE users (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email          CITEXT UNIQUE,
  email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  apple_sub      TEXT UNIQUE,
  google_sub     TEXT UNIQUE,
  display_name   TEXT,
  locale         TEXT NOT NULL DEFAULT 'ru',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at     TIMESTAMPTZ
);

-- email_codes (OTP)
CREATE TABLE email_codes (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email       CITEXT NOT NULL,
  code_hash   BYTEA NOT NULL,         -- bcrypt(code)
  attempts    INT NOT NULL DEFAULT 0,
  expires_at  TIMESTAMPTZ NOT NULL,
  used_at     TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON email_codes (email, expires_at);

-- refresh tokens
CREATE TABLE refresh_tokens (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash    BYTEA NOT NULL,        -- sha256(token)
  device_id     TEXT,
  user_agent    TEXT,
  last_used_at  TIMESTAMPTZ,
  expires_at    TIMESTAMPTZ NOT NULL,
  revoked_at    TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ON refresh_tokens (token_hash);
CREATE INDEX ON refresh_tokens (user_id) WHERE revoked_at IS NULL;

-- conversations (одна на user, но схема на будущее)
CREATE TABLE conversations (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ON conversations (user_id);  -- один чат на user в v1

-- messages
CREATE TABLE messages (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,  -- denorm для быстрых проверок
  role            TEXT NOT NULL CHECK (role IN ('user','assistant','system')),
  status          TEXT NOT NULL CHECK (status IN ('pending','complete','cancelled','failed')) DEFAULT 'complete',
  tokens_in       INT,
  tokens_out      INT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON messages (conversation_id, created_at);

-- message_blocks: контент сообщения (мультимодальный)
CREATE TABLE message_blocks (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  message_id   UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  order_index  INT NOT NULL,
  type         TEXT NOT NULL CHECK (type IN ('text','image','audio','transcription','tool_use','tool_result','fertilizer_card')),
  content_text TEXT,
  storage_key  TEXT,                  -- для image/audio
  metadata     JSONB,                 -- для tool_use args / fertilizer_card payload / transcription duration
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON message_blocks (message_id, order_index);

-- uploads tracking (для GC и проверки ownership)
CREATE TABLE uploads (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  storage_key   TEXT NOT NULL UNIQUE,
  content_type  TEXT NOT NULL,
  size_bytes    BIGINT NOT NULL,
  used          BOOLEAN NOT NULL DEFAULT FALSE,  -- стало ли частью message_block
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON uploads (user_id, created_at);

-- fertilizers (каталог компании)
CREATE TABLE fertilizers (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug            TEXT UNIQUE NOT NULL,
  name            TEXT NOT NULL,
  short_desc      TEXT NOT NULL,
  long_desc       TEXT,
  image_url       TEXT,
  deeplink_url    TEXT,
  category        TEXT NOT NULL,
  -- условия применимости для tool use:
  problems        TEXT[] NOT NULL,    -- e.g. {'leaf_yellowing','phosphorus_deficiency'}
  plants          TEXT[],             -- e.g. {'tomato','cucumber'} или NULL = универсальное
  active          BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON fertilizers USING GIN (problems);

-- usage_log (для лимитов и биллинга-аналитики)
CREATE TABLE usage_log (
  id          BIGSERIAL PRIMARY KEY,
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  endpoint    TEXT NOT NULL,
  tokens_in   INT,
  tokens_out  INT,
  cost_usd    NUMERIC(10,6),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON usage_log (user_id, created_at);

-- audit (для удалений / чувствительных действий)
CREATE TABLE audit_log (
  id          BIGSERIAL PRIMARY KEY,
  user_id     UUID,
  action      TEXT NOT NULL,
  details     JSONB,
  ip          INET,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 6.2 Миграции
Goose, файлы `migrations/<timestamp>_<name>.sql` с `-- +goose Up` / `-- +goose Down`. Запуск через `cmd/migrate` или `make migrate-up`.

### 6.3 Каскадное удаление аккаунта
`DELETE FROM users WHERE id=$1` каскадно удаляет всё через FK. Параллельно — фоновое задание удаляет объекты в Object Storage по префиксу `u/{user_id}/`.

### 6.4 Tool `recommend_fertilizer`

Вход (от Claude):
```json
{
  "problem": "leaf_yellowing",
  "plant": "tomato",
  "severity": "moderate",
  "notes": "lower leaves first, no spots"
}
```

SQL (упрощённо):
```sql
SELECT id, slug, name, short_desc, image_url, deeplink_url
FROM fertilizers
WHERE active
  AND problems @> ARRAY[$1]::text[]
  AND ($2::text IS NULL OR plants IS NULL OR plants @> ARRAY[$2]::text[])
ORDER BY priority DESC NULLS LAST
LIMIT 3;
```

Возврат в Claude → Claude вплетает в ответ + отдельный `fertilizer_card` event на клиента.

---

## 7. Промптинг и Tool Use

### 7.1 System prompt (краткое содержание, итоговый текст в `internal/llm/prompts/system.txt`)

- Роль: дружелюбный эксперт-агроном (RU).
- Цель: понять проблему по описанию/фото/голосу → дать практичную помощь → подобрать удобрение из каталога через `recommend_fertilizer`.
- Формат ответа: маркдаун, секции:
  1. **Что я вижу** — краткое описание фото/жалобы (1-2 предложения).
  2. **Возможные причины** — список с уровнем уверенности (высокий/средний/низкий).
  3. **Что делать** — нумерованные шаги, конкретные действия.
  4. После шагов — обязательный вызов `recommend_fertilizer` (если применимо).
- Стиль: простым языком, без латинских названий без перевода, без агрессивных продаж.
- Безопасность: не давать токсичных советов; при подозрении на серьёзную болезнь — рекомендовать карантин/удаление; не комментировать темы вне садоводства.
- При неясности — задавать **один** уточняющий вопрос (а не список).

### 7.2 Tool definition

```json
{
  "name": "recommend_fertilizer",
  "description": "Подбирает 1-3 удобрения из каталога компании, подходящих к проблеме",
  "input_schema": {
    "type": "object",
    "properties": {
      "problem":   { "type": "string", "enum": ["leaf_yellowing","leaf_spots","wilting","stunted_growth","poor_fruiting","root_rot","pest_aphid","pest_mite","nitrogen_deficiency","phosphorus_deficiency","potassium_deficiency","calcium_deficiency","magnesium_deficiency","iron_deficiency","general_stress"] },
      "plant":    { "type": "string", "description": "tomato, cucumber, pepper, ... или omit если неизвестно" },
      "severity": { "type": "string", "enum": ["mild","moderate","severe"] },
      "notes":    { "type": "string" }
    },
    "required": ["problem"]
  }
}
```

### 7.3 Prompt caching
Передаём `system` и определения инструментов с `cache_control: ephemeral` (5-min TTL) — экономит токены при частых обращениях. Подробности — в Anthropic docs (проверить актуальное API через ctx7 при имплементации).

### 7.4 Контекст беседы
- В Claude передаём последние ~20 сообщений (с фото текущего сообщения).
- Превышение лимита токенов: усечение старых сообщений с заменой на summary (этап v2; в v1 — простое усечение).

---

## 8. Безопасность

### 8.1 Аутентификация и токены
- **Access token:** JWT RS256, ttl 15 мин. Содержит: `sub` (user_id), `iat`, `exp`, `jti`. Подписывается приватным ключом, лежащим в Lockbox; публичный ключ в JWKS-эндпоинте (внутренний).
- **Refresh token:** opaque random 32 байта (base64url), ttl 30 дней. В БД хранится только `sha256` хэш. Каждый refresh ротирует токен (старый помечается `revoked`).
- **Apple/Google id_token:** валидируется через JWKS провайдера с проверкой `aud`, `iss`, `exp`, `nonce` (для Apple). Для Sign-In with Apple — храним `apple_sub`, **не** доверяем e-mail (он может быть private relay).
- **Email OTP:** 6 цифр, bcrypt-хэш в БД, ttl 10 мин, ≤ 5 попыток ввода кода, ≤ 3 запроса кода в час на email.

### 8.2 Защита API
- **TLS 1.2+ обязателен**, HSTS на всех окружениях.
- **CORS:** mobile использует напрямую, CORS не нужен; для будущей админки — whitelist origins.
- **Rate limiting (Redis):**
  - IP: 100 RPS, 5000 RPM
  - User: 20 RPS на `/messages`, 60 RPS общий
  - Email-OTP: 3 запроса/час/email
  - Логин: 10/мин/IP
- **Размеры:** text body ≤ 32 КБ, image ≤ 10 МБ, audio ≤ 25 МБ (60s). Проверка и в presign, и в Object Storage policy.
- **Whitelist content-types:** image/jpeg, image/png, image/webp, image/heic; audio/m4a, audio/aac, audio/mp4.
- **Проверка ownership:** при любом обращении к `storage_key` или `message_id` — проверка `user_id`.

### 8.3 Защита данных
- **At-rest:** диски шифруются (Yandex Cloud encrypted volumes), Object Storage — KMS-ключ.
- **In-transit:** TLS на всех границах, включая внутренние (DB, Redis).
- **PII в логах:** запрещено. В `slog` подключаем `Replacer`, фильтрующий поля `email`, `id_token`, `access_token`, `refresh_token`, `code`, `text`, `transcription`. Mobile — аналогичный фильтр в `logger`.
- **Удаление аккаунта:** синхронно удаляем БД-записи, асинхронно (job) — объекты Object Storage по префиксу `u/{user_id}/`. Запись в `audit_log`.
- **Бэкапы:** Yandex Managed PostgreSQL — daily snapshots, retention 14 дней. Object Storage — versioning отключён (чтобы удаление было реальным).
- **Изоляция PII от worker'а и Anthropic.** В worker отправляются только `messages` + `uid_hash = sha256(uid + UID_HASH_PEPPER)`. Email, реальный UUID, IP клиента, refresh-токены не покидают РФ-инфраструктуру. Worker — stateless, без диска для долгосрочного хранения.
- **Не отправляем фото в Anthropic дольше, чем нужно для запроса.** Anthropic API не сохраняет контент по умолчанию (Zero Data Retention для enterprise; для standard tier — стандартная политика).
- **Структура хранения объектов в Object Storage:** `u/{user_id}/img/...`, `u/{user_id}/audio/...` — позволяет каскадное удаление по префиксу.

### 8.4 OAuth-специфика
- **Apple:** обязателен, если есть Google-вход (Apple Guideline 4.8). Использовать `nonce` для anti-replay.
- **Google:** разные `client_id` для iOS/Android/web (последний — для serverside верификации).
- **Email private relay (Apple):** учитывать, что email от Apple может быть `@privaterelay.appleid.com`.

### 8.5 Антибот / антифрод (v1 минимум, v2 расширить)
- Rate limit на IP+User-Agent.
- Опционально: Apple `DeviceCheck` / Google Play `Integrity API` для подписи запросов на `/messages`.
- Если злоупотребление обнаружено — soft block (повышенный rate limit) → hard block (флаг `users.blocked`).

### 8.6 Секреты
- В коде нет ни одного секрета. Все через env.
- **Бэкенд (Yandex VM):** секреты читаются из **Yandex Lockbox** при старте контейнера (sidecar или init-скрипт получает значения и кладёт в env). Доступ к Lockbox через сервис-аккаунт, привязанный к VM.
- **LLM-worker (HostKey VM):** секреты лежат в `/etc/llmworker/.env` на **LUKS-зашифрованном томе** (поднимается внутри VM на отдельном loopback-диске или дополнительном volume; провайдер-агностично). Файл с правами `600`, владелец — non-root user, под которым запущен Docker Compose. Никаких внешних secret-провайдеров — один VPS, отдельный Vault избыточен. Считаем, что у оператора VPS есть физический root-доступ к хосту, поэтому критичные секреты (`ANTHROPIC_API_KEY`, mTLS-приватный ключ) можно дополнительно держать только в смонтированном виде в `tmpfs` и при ребуте подгружать вручную (опционально — обсудить с заказчиком).
- **Секреты бэкенда (Yandex Cloud):** `DATABASE_URL`, `REDIS_URL`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `JWT_PRIVATE_KEY`, `LLM_WORKER_URL`, `LLM_WORKER_MTLS_CERT`, `LLM_WORKER_MTLS_KEY`, `LLM_WORKER_CA`, `APPLE_CLIENT_ID`, `GOOGLE_CLIENT_ID`, `SMTP_USERNAME`, `SMTP_PASSWORD` (Yandex 360), `SPEECHKIT_API_KEY`, `UID_HASH_PEPPER`, `SENTRY_DSN`.
- **Секреты worker'а (HostKey):** `ANTHROPIC_API_KEY`, `LLM_WORKER_MTLS_CERT`, `LLM_WORKER_MTLS_KEY`, `LLM_WORKER_CA`, `BACKEND_CALLBACK_URL` (для tool-callback на РФ-бэк), `SENTRY_DSN`. **`ANTHROPIC_API_KEY` хранится только тут**, бэкенд его не знает.
- Ротация: refresh-токены — ротация при каждом использовании; JWT-ключ — ручная ротация (kid в JWKS); mTLS-сертификаты — ежеквартально через cert-manager или вручную; Anthropic-ключ — ротация при подозрении на утечку.

---

## 9. Наблюдаемость

- **Логи:** JSON через `slog`, поля: `time`, `level`, `msg`, `request_id`, `user_id` (хэш), `endpoint`, `latency_ms`. Никакого контента сообщений.
- **Метрики (Prometheus):** RPS/latency по эндпоинтам, ошибки по кодам, токены Claude (in/out), стоимость, размер очередей, статусы аплоадов.
- **Трейсинг:** OpenTelemetry-совместимые спаны на каждом запросе; жёлтое — Claude латенси, серое — БД.
- **Sentry:** все unhandled errors с `request_id`. Mobile — паники Dart, нативные крэши через `sentry_flutter`.
- **Алерты:**
  - 5xx > 1% за 5 мин
  - Latency p95 > 5s за 10 мин
  - Claude error rate > 5%
  - Disk > 80%
  - DB connections > 80%

---

## 10. CI/CD

### 10.1 Backend CI (`.github/workflows/backend-ci.yml`)
- triggers: pull_request, push на main
- jobs:
  1. `lint` — golangci-lint
  2. `unit` — `go test ./...` без тегов
  3. `integration` — `go test -tags=integration ./...` (Postgres testcontainer)
  4. `build` — multi-stage Docker, push в Yandex Container Registry (для api) и в GitHub Container Registry (для llmworker, чтобы образ скачивался с HostKey-VM без зависимости от российских CR) на main
  5. `deploy-prod` — manual approval; SSH на VM → `docker compose pull && docker compose up -d` + healthcheck

### 10.2 Mobile CI (`.github/workflows/mobile-ci.yml`)
- jobs:
  1. `analyze` — `dart format --set-exit-if-changed`, `dart analyze`
  2. `test` — `flutter test --coverage`
  3. `build-android` — `flutter build apk --release` (с подписью на release-теге)
  4. `build-ios` — `flutter build ipa --release` (требует macOS runner)

### 10.3 Окружения

В v1 у нас **только два окружения** (см. SPEC §8 D15):
- **dev:** локально, docker-compose, фейковый OAuth, mock LLM-клиент или подключение к prod-worker'у с dev-ключом Anthropic.
- **prod:** `api.<domain>` (Yandex Cloud VM) + `llm.<internal-domain>` (HostKey Frankfurt VM). До публикации в сторы prod используется и для нашего ручного тестирования.

После релиза в сторы — добавить **stage** окружение (отдельная пара VM, отдельная БД, отдельный Anthropic-ключ) для тестов, чтобы новые фичи не катились сразу на пользователей.

### 10.4 Релизы
- Backend: SemVer теги, Container image теги = git SHA + версия.
- Mobile: build number = CI run number, version в `pubspec.yaml`. Загрузка в TestFlight / Play Internal Track автоматически на тег `v*`.

---

## 11. Доступ к Anthropic API из РФ

### 11.1 Контекст (на конец 2025 — начало 2026)

Anthropic блокирует доступ из РФ многоуровнево:
- GeoIP-блокировка (MaxMind + Cloudflare).
- Блок-лист российских AS: Yandex Cloud, Selectel, VK Cloud, Rostelecom, Beeline, MTS, Megafon — все российские облака отвергаются на каждом запросе, не только при логине.
- Проверка billing/IP mismatch: иностранная карта при российском IP → подозрительный паттерн → бан ключа.
- Ownership clause: контроль организации >50% из РФ → нарушение Acceptable Use Policy.

**AWS Bedrock не работает** для РФ-инфраструктуры: AWS приостановил оформление новых аккаунтов из РФ с марта 2022 и блокирует входящий трафик из российских AS. **GCP Vertex AI** — аналогично + зависимость от той же политики Anthropic.

### 11.2 Решение: split LLM-worker

Отдельный микросервис `llm-worker` развёрнут на VPS вне РФ. РФ-бэкенд не делает запросов к Anthropic вообще — только к worker'у через mTLS.

**Конфигурация:**
- **Хостинг:** HostKey Frankfurt (Frankfurt 1 Data Center; Amsterdam EuNetworks как резерв). Тариф `vm.v2-nano` (2 vCPU / 4 GB / 60 GB NVMe / 1 Gbps / 3 TB трафика, ~830 ₽/мес) на старте; масштабирование вертикальное через `vm.v2-small/medium`.
- **Платежи за инфру:** рублёвая карта/банковский перевод заказчика на ООО «АЙТИБ» (HostKey).
- **Anthropic-аккаунт:** оформлен на **иностранное юрлицо/email**, привязан к зарубежной карте и адресу. Anthropic в рублях не принимает оплату — зарубежная карта для самой Anthropic остаётся обязательной. Sandbox-ключ для stage, production-ключ для prod.
- **Billing/IP mismatch для Anthropic:** не возникает — Anthropic-аккаунт привязан к иностранной карте, а исходящий трафик идёт с IP HostKey-VM во Frankfurt. То, что инфра оплачена с рублёвого счёта, Anthropic не видит.
- **Транспорт:** mTLS поверх HTTPS на нестандартном порту. Публичный IP HostKey-VM, доступ ограничен HostKey-firewall (или iptables на VM) — IP-allowlist только для IP бэкенда в Yandex Cloud. Приватная сеть HostKey между VM не используется, т.к. worker и бэк физически в разных площадках/провайдерах.
- **Изоляция данных:** worker не имеет долговременного хранилища PII, без БД, без метрик с PII. Логи только: `request_id`, `model`, `tokens_in`, `tokens_out`, `latency_ms`.

### 11.3 API между РФ-бэком и worker'ом

```
POST /v1/llm/messages              (приватный, mTLS)
  body: {
    model: "claude-opus-4-7",
    system: "...",
    messages: [...],
    tools: [...],
    stream: true,
    metadata: { uid_hash: "sha256(uid+pepper)", request_id: "req_..." }
  }
  resp: text/event-stream
    event: delta            data: { text: "..." }
    event: tool_use         data: { tool, args }
    event: usage            data: { tokens_in, tokens_out }
    event: error            data: { code, message }
    event: done             data: {}
```

**Tool callback** (от worker'а обратно в РФ-бэк, mTLS):
```
POST /internal/v1/tools/fertilizer
  body: { args: { problem, plant?, severity?, notes? } }
  resp: { products: [{ id, slug, name, short_desc, image_url, deeplink_url }] }
```

### 11.4 Что НЕ передаётся в worker

- email пользователя
- реальный UUID `users.id`
- refresh-токены, JWT
- пароли/OTP
- IP-адрес клиента

Передаётся только хэш `uid_hash` для антифрод-метрик (одно значение → один пользователь, без обратимости).

### 11.5 Резервные пути (если основная схема перестаёт работать)

В коде — интерфейс `internal/llm.Client` с реализациями:
- `worker_client` (default, через свой worker).
- `openrouter_client` (резерв через OpenRouter — агрегатор; +20-30% к цене, чужой ключ, подключается за час).
- `mock_client` (dev/тесты).

Переключение через env `LLM_CLIENT_KIND`. Worker и резерв стоят в одном `internal/llm` модуле, выбираются на старте бэкенда.

### 11.6 Соответствие 152-ФЗ

Архитектура **улучшает** соблюдение 152-ФЗ относительно «прямого» вызова:
- Все ПДн (email, имя, фото с лицами/окружением, аудио с голосом) хранятся в Yandex Cloud (РФ).
- Trans-border data flow: только обезличенные сообщения в worker → Anthropic. ПДн не покидают РФ.
- При запросе субъекта на удаление — каскадное удаление в Yandex Cloud, worker очищать не нужно (он stateless, ничего не хранит).

### 11.7 Принятые риски HostKey

Hosting-провайдер worker'а — **ООО «АЙТИБ»** (HostKey, юрлицо в РФ). Физическая площадка VM — Frankfurt, Германия. Это решение принято заказчиком осознанно ради оплаты в рублях; ниже зафиксированы риски и принятые компенсирующие меры.

| Риск | Вероятность | Влияние | Компенсация |
| ---- | ----------- | ------- | ----------- |
| Российские власти (РКН, ФСБ, суд) принуждают ООО «АЙТИБ» прекратить услугу или раскрыть данные | Низкая в обычной обстановке, средняя при ужесточении регулирования | Полная остановка worker'а; в худшем случае — попытка снять snapshot диска | (а) Готовая инструкция переезда на резервный VPS у иностранного провайдера (Hetzner / OVH / Vultr Frankfurt) — оператор переключает DNS + перевыпускает mTLS-сертификаты за < 2 часов; (б) `ANTHROPIC_API_KEY` и mTLS-приватный ключ — только в LUKS-томе, при ребуте требуют ручной разблокировки; snapshot диска без passphrase не выдаёт ключи; (в) IP-allowlist на worker'е + рейт-лимит, чтобы попавший в чужие руки ключ нельзя было сразу слить через прокси. |
| Anthropic в будущем включит в политику явный запрет инфры, аффилированной с РФ-юрлицом | Низкая | Бан ключа Anthropic | Резервный `openrouter_client` уже описан в §11.5; миграция на иностранного провайдера готова (см. выше). |
| HostKey блокирует Anthropic-трафик «сверху» (исполнение требования РКН о прекращении доступа к зарубежным AI-сервисам из РФ-связанной инфры) | Низкая, но с течением времени растёт | Worker не достучится до Anthropic | Тот же план миграции; алерт на 5xx от worker'а. |
| Snapshot/IPMI-доступ оператора к VM | Высокая (это нормальная функция любого VPS-провайдера) | Доступ к данным на диске | LUKS + ручная разблокировка после ребута; ключи Anthropic регулярно ротируются (см. §8.6). |

**Triggers для миграции на иностранного провайдера (заранее согласованы):**
1. Anthropic банит ключ под формулировкой про юрисдикцию инфраструктуры.
2. HostKey уведомляет о законодательных требованиях прекратить услугу или раскрыть данные.
3. Появляется регуляторный акт, прямо запрещающий проксирование AI-трафика через инфру РФ-юрлица.

Миграция готовится не реактивно, а **заранее**: terraform-описание worker'а параметризовано (`provider = "hostkey" | "hetzner" | "ovh"`), docker-образ один и тот же, mTLS-CA общий. В норме это однокнопочный переезд.

---

## 12. Локальная разработка

`backend/docker-compose.yml`:

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: dev
      POSTGRES_DB: safegarden
    ports: ["5432:5432"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: dev
      MINIO_ROOT_PASSWORD: devdevdev
    ports: ["9000:9000", "9001:9001"]

  mailhog:
    image: mailhog/mailhog
    ports: ["8025:8025", "1025:1025"]
```

Бэкенд запускается отдельно через `air` (live reload) с env `.env.local`.

Для Claude в dev есть три режима:
1. **Mock** (`LLM_CLIENT_KIND=mock`) — `internal/llm/mock` воспроизводит ответы из фикстур; быстро, бесплатно, без сети.
2. **Локальный worker** (`LLM_CLIENT_KIND=worker`, `LLM_WORKER_URL=http://localhost:8081`) — запуск worker'а локально через `make worker-dev`. Использует stage-ключ Anthropic. **Внимание:** запросы из РФ-IP к api.anthropic.com будут отвергнуты — нужен VPN на dev-машине или прокси через prod-worker во Frankfurt.
3. **Удалённый stage worker** — пинг на stage-VM HostKey Frankfurt.

---

## 13. Риски и смягчения

| Риск                                                          | Влияние   | Смягчение                                                                                                                                  |
| ------------------------------------------------------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Anthropic банит ключ (детект паттернов)                       | Критично  | Резервный `openrouter_client` готов и протестирован; разные ключи для stage/prod; мониторинг 401/403 от Anthropic с алертом.               |
| Worker (HostKey) недоступен                                   | Критично  | SLO worker'а 99.5%; health-check каждые 30s; при 5xx от worker'а — graceful 503 с понятным сообщением; готовая инструкция переезда на резервный VPS у иностранного провайдера (Hetzner / OVH / Vultr). См. §11.7. |
| Apple/Google отклоняют приложение                             | Высокое   | Чтение Guidelines заранее (App Store §5.1.1, Sign in with Apple §4.8); чек-лист перед сабмитом (см. ROADMAP §7).                            |
| Утечка фото пользователей                                     | Критично  | Раздельные buckets, presigned URLs short TTL, KMS, audit_log, без общедоступного листинга.                                                 |
| Пользователи отправляют не-садоводческий контент              | Среднее   | System prompt с инструкцией отказа; рейт-лимит; модерация Claude; отказ в ответе с дружелюбным сообщением.                                 |
| Стоимость Claude растёт с ростом аудитории                    | Среднее   | Prompt caching, дневной лимит запросов на пользователя, мониторинг `usage_log`, оповещения по бюджету.                                     |
| Ответ Claude фактически неверен                               | Среднее   | Дисклеймер «рекомендации носят информационный характер»; обратная связь (reaction up/down) — v2.                                           |
| Перегрузка при вирусном росте                                 | Среднее   | Auto-scaling backend, очередь сообщений (опционально через Redis Streams), graceful degradation (без стриминга).                           |
| Yandex SpeechKit недоступен / ошибка распознавания            | Низкое    | Фолбэк: показать пользователю «не распознали — пришлите текстом»; альтернативный провайдер GigaChat/Sber готов как резерв.                  |

---

## 14. Что проверить через ctx7 при имплементации

Перед написанием кода в каждом этапе обязательно подтянуть актуальную документацию:

- **Этап 0–2:** `chi`, `pgx/v5`, `sqlc`, `golangci-lint`, `riverpod`, `go_router`, `dio`.
- **Этап 1:** `sign_in_with_apple`, `google_sign_in`, `golang-jwt/jwt/v5`, `coreos/go-oidc`, `gomail` (или альтернатива) для SMTP через Yandex 360.
- **Этап 2:** `anthropic-sdk-go` (актуальный модельный ID Opus, формат streaming, prompt caching API). mTLS-конфигурация в Go (`crypto/tls`).
- **Этап 3:** `image_picker`, `flutter_image_compress`, S3 presigned PUT (Yandex Object Storage).
- **Этап 4:** `record` (Flutter), Yandex SpeechKit v3 API (sync recognize, форматы, ошибки), `ffmpeg` для конвертации m4a→OggOpus.
- **Этап 5:** Anthropic Tool Use — последний формат `tool_use` / `tool_result` событий в стриме.
- **Этап 7:** Apple App Store Review Guidelines, Google Play Data Safety form.
