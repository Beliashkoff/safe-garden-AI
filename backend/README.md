# Backend — Safe Garden AI

HTTP API на Go (chi, pgx, sqlc), часть РФ-бэкенда из [`../ARCHITECTURE.md`](../ARCHITECTURE.md) §3. Бизнес-логика, БД, S3 и SMTP — здесь. Вызовы Claude идут через отдельный `llm-worker` во Frankfurt (на HostKey, появится в Этапе 0.7), не напрямую.

> Текущее состояние — скелет: `cmd/api` отдаёт `/healthz` и `/readyz`, config через envconfig, slog-логи с PII-фильтром, опциональный Sentry. Полная реализация — по этапам [`../ROADMAP.md`](../ROADMAP.md).

## Prerequisites

- **Go 1.24+** — версия фиксируется в [`go.mod`](./go.mod).
- **Docker** + **Docker Compose** — для зависимостей через `docker compose up -d`.
- **make** — все команды зашиты в [`Makefile`](./Makefile).
- (Опционально) **air** для live-reload: `go install github.com/air-verse/air@latest`.
- (Опционально) **golangci-lint** v1.62+ для `make lint`: см. [официальные инструкции](https://golangci-lint.run/welcome/install/).
- (Опционально) **goose** для миграций (появятся в Этапе 1): `go install github.com/pressly/goose/v3/cmd/goose@latest`.
- (Опционально) **sqlc** для генерации запросов (появится в Этапе 1): `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`.

## Структура каталога

```
backend/
├── cmd/
│   └── api/                  # HTTP-сервис (точка входа main.go)
├── internal/
│   ├── config/               # envconfig + .env
│   └── observability/        # slog handler (PII-фильтр) + Sentry init
├── .air.toml                 # live-reload
├── .dockerignore
├── .env.example              # шаблон env для локалки
├── .golangci.yml             # линтеры
├── docker-compose.yml        # postgres + redis + minio + mailhog
├── Dockerfile                # multi-stage, distroless
├── go.mod / go.sum
└── Makefile
```

Каталоги, которые появятся по этапам (по `ROADMAP.md`):

- `migrations/` — goose-миграции (Этап 1.1).
- `internal/auth/`, `internal/storage/`, `internal/usecase/`, `internal/transport/http/` — слои домена (Этап 1+).
- `internal/llm/` — клиент к `llm-worker` (Этап 2.2).
- `internal/llmworker/` — код самого worker'а (Этап 0.7).

## Конфигурация (env)

Шаблон в [`.env.example`](./.env.example). Для локального запуска скопировать в `.env`:

```bash
cp .env.example .env
```

| Переменная   | По умолчанию | Назначение |
| ------------ | ------------ | ---------- |
| `ENV`        | `dev`        | `dev` / `stage` / `prod`. Влияет на формат логов и обязательность ряда полей. |
| `HTTP_HOST`  | (пусто)      | Интерфейс HTTP-сервера. Пусто — слушать все интерфейсы. |
| `HTTP_PORT`  | `8080`       | Порт HTTP-сервера. |
| `LOG_LEVEL`  | `info`       | `debug` / `info` / `warn` / `error`. |
| `SENTRY_DSN` | (пусто)      | DSN для Sentry. Пусто — Sentry не инициализируется. |

Список будет расширяться по этапам: БД (Этап 1.1), Redis, S3-креды (Этап 3.1), Anthropic / Yandex SpeechKit / Yandex 360 SMTP (Этапы 2, 4). Все секреты — только через env, никогда в коде.

В `.env.example` уже лежат **закомментированные** placeholder'ы под все ключи Этапа 0.6. Раскомментируем по мере подключения соответствующего этапа.

## Регистрация внешних аккаунтов (Этап 0.6)

Этап 0.6 — организационный: заказчик регистрирует внешние сервисы и получает ключи. Кода тут нет, но именно из этих регистраций берутся значения для `.env` следующих этапов. Чек-лист задач — в [`../ROADMAP.md`](../ROADMAP.md) §0.6.

> **Секреты никогда не коммитим.** Хранение — в password manager заказчика (1Password / Bitwarden / Yandex Vault). В CI добавим через GitHub Secrets в Этапе 2. В prod-VM — через Yandex Lockbox и LUKS-том (см. `../ARCHITECTURE.md` §8.6).

### A. HostKey (hostkey.ru)

VPS-провайдер для `llm-worker'а` во Frankfurt. Юрлицо — ООО «АЙТИБ» (РФ), оплата ₽. Принятые юридические риски и план миграции на иностранного провайдера — в [`../ARCHITECTURE.md`](../ARCHITECTURE.md) §11.7.

1. Регистрация аккаунта на юрлицо/карту заказчика на `hostkey.ru`.
2. Пополнение баланса (минимум для активации — около 1 000 ₽).
3. **VPS пока не покупаем** — VM создадим в Этапе 2.2 через terraform. Сейчас нужен только аккаунт.
4. Включить 2FA в личном кабинете.
5. (Опционально) Залить SSH public key в профиль — пригодится в Этапе 0.7.

### B. Anthropic (console.anthropic.com)

Источник Claude API для worker'а. Anthropic в рублях не принимает — **зарубежная карта и иностранное юрлицо/email обязательны**, даже несмотря на то, что инфра у HostKey оплачивается в ₽.

1. Регистрация на иностранное юрлицо/email.
2. Workspace + Organization → ввести billing details, привязать зарубежную карту.
3. Settings → Privacy → **отключить** «training on my data» (важно: PII не должна попадать в обучение).
4. Create API key → сохранить в password manager как `ANTHROPIC_API_KEY`. **Только** для `internal/llmworker/`, никогда не использовать в основном бэкенде (CLAUDE.md инвариант №5).
5. Запросить Tier 1 production (через usage-tier upgrade / org verification — Anthropic просит небольшой первый платёж).
6. Включить billing alerts на $20 / $50 / $100.

### C. Yandex Cloud (console.cloud.yandex.ru)

Облако для РФ-бэкенда. PII (email, фото) физически не покидает РФ (152-ФЗ).

1. Создать организацию, привязать платёжный аккаунт. Free Tier даёт стартовый грант.
2. Создать каталоги (folder): `safe-garden-prod`, `safe-garden-stage`.
3. Service account для бэкенда. Роли на старте — `viewer` на каталог; в Этапе 2.2 расширим до:
   - `managed-postgresql.editor`
   - `managed-redis.editor`
   - `storage.editor` (Object Storage)
   - `dns.editor` (Cloud DNS)
   - `container-registry.images.puller`
   - `lockbox.payloadViewer`
4. Скачать IAM-ключ service account → положить в password manager. **Не** в git.
5. Включить Yandex Cloud DNS (бесплатно) — туда перевезём `agronomai.site` (см. F).
6. Bucket для медиа (`safe-garden-prod-media`) создадим в Этапе 3.1.

### D. Yandex 360 для домена `agronomai.site`

Email-провайдер для OTP. SMTP-отправитель — `noreply@agronomai.site`.

1. На `360.yandex.ru/business` подключить домен `agronomai.site` — Yandex даст инструкцию по TXT/CNAME подтверждению владения.
2. Создать ящик `noreply@agronomai.site`.
3. В настройках ящика → пароли приложений → создать пароль для **«Почта по протоколу IMAP/SMTP»** → сохранить как `SMTP_PASSWORD`. Не использовать основной пароль ящика.
4. В DNS-зоне `agronomai.site` (в Yandex Cloud DNS — см. F) добавить:
   - `MX` → `mx.yandex.net.` приоритет 10
   - `TXT` SPF: `v=spf1 include:_spf.yandex.net ~all`
   - `TXT` DKIM: значение из Yandex 360 → DKIM-настройки
   - `TXT` DMARC: `v=DMARC1; p=quarantine; rua=mailto:postmaster@agronomai.site`
5. Smoke test отправки (с любой машины с интернетом):

   ```bash
   swaks --to <your-test@gmail.com> --from noreply@agronomai.site \
         --server smtp.yandex.ru:465 --tls-on-connect \
         --auth-user noreply@agronomai.site \
         --auth-password '<app-password>'
   ```

   В заголовках письма должен быть валидный DKIM-подпись (`DKIM-Signature: ...`, `Authentication-Results: ... dkim=pass`).

### E. Yandex SpeechKit

Транскрипция голосовых сообщений (понадобится в Этапе 4).

1. В Yandex Cloud → IAM → Service accounts → создать `speechkit-stt` с ролью `ai.speechkit-stt.user` (на каталог `safe-garden-prod`).
2. Создать API-ключ → сохранить как `YANDEX_SPEECHKIT_API_KEY`.
3. Включить billing-алёрты.
4. Sync recognize ограничен 30 с — обрабатывается на уровне приложения в Этапе 4.

### F. Домен `agronomai.site` + DNS

Один домен на всё через поддомены. Карта:

| Имя | Назначение | Когда создаём A-запись |
| --- | --- | --- |
| `agronomai.site` (apex) | Лендинг + Privacy + Terms | Этап 7.2 |
| `api.agronomai.site` | РФ-бэкенд (Yandex Compute) | Этап 2.2 |
| `worker.agronomai.site` | LLM-worker (опционально; mTLS защищает от анонимных обращений) | Этап 2.2 |
| `noreply@agronomai.site` | SMTP-отправитель | Этап 0.6 (через MX, см. раздел D) |

1. Определить текущего регистратора `agronomai.site` (`.site` — международный TLD, обычно международный регистратор).
2. В Yandex Cloud DNS создать публичную зону `agronomai.site`.
3. У регистратора заменить NS-серверы на выданные Yandex Cloud DNS (например, `ns1.yandexcloud.net`, `ns2.yandexcloud.net`). Распространение NS — до 48 ч, обычно <2 ч.
4. После пропагации NS — добавить в зону записи из раздела D (MX/SPF/DKIM/DMARC). A-записи на VM пойдут в Этапах 2.2 и 7.2.
5. Проверка:

   ```bash
   dig NS agronomai.site            # должны быть ns*.yandexcloud.net
   dig MX agronomai.site            # должен быть mx.yandex.net.
   dig TXT agronomai.site           # SPF + DKIM + DMARC
   ```

### Когда раскомментировать переменные в `.env.example`

| Переменная | Раскомментируется в этапе |
| --- | --- |
| `POSTGRES_DSN` | 1.1 |
| `REDIS_ADDR` | 1.2 |
| `S3_*` | 3.1 |
| `JWT_*` | 1.1 |
| `APPLE_*`, `GOOGLE_CLIENT_ID_*` | 1.1 |
| `SMTP_*` | 1.3 |
| `LLM_WORKER_*` | 2.2 |
| `YANDEX_SPEECHKIT_API_KEY` | 4.2 |

`ANTHROPIC_API_KEY` и `UID_HASH_PEPPER` в этом файле никогда не появятся — они только в `.env` worker'а (`internal/llmworker/`, появится в Этапе 0.7).

## Команды Makefile

| Цель                  | Что делает |
| --------------------- | ---------- |
| `make dev`            | `docker compose up -d` + `air` (live-reload). |
| `make test`           | `go test ./...` — unit-тесты. |
| `make test-integration` | `go test -tags=integration ./...` — integration (testcontainer Postgres, появятся в Этапе 1). |
| `make lint`           | `golangci-lint run ./...`. |
| `make build`          | Сборка статического бинаря в `bin/api` (`-ldflags="-s -w"`). |
| `make migrate-up`     | `goose -dir migrations postgres "$POSTGRES_DSN" up` (Этап 1+). |
| `make migrate-down`   | `goose ... down` — для локальной отладки миграций. В prod применяется только `up`. |
| `make sqlc-gen`       | `sqlc generate` — перегенерация sqlc-кода после правки `*.sql`. |
| `make clean`          | Удалить `bin/` и `tmp/` (артефакты `air`). |

## Локальный запуск

```bash
# 1. Зависимости
cp .env.example .env
docker compose up -d

# 2. HTTP-сервер (любой из вариантов)
go run ./cmd/api          # одноразовый запуск
air                       # live-reload (требует установленный air)
make dev                  # docker compose + air одной командой

# 3. Проверка
curl http://localhost:8080/healthz   # → 200 ok
curl http://localhost:8080/readyz    # → 200 ok
```

Сервисы, поднимаемые `docker compose up -d` (см. [`docker-compose.yml`](./docker-compose.yml)):

| Сервис   | Порт(ы)        | Учётка по умолчанию             | Назначение |
| -------- | -------------- | ------------------------------- | ---------- |
| postgres | `5432`         | `safegarden` / `safegarden` (db `safegarden`) | Основная БД. |
| redis    | `6379`         | —                               | Кэш, rate-limit, фоновые очереди. |
| minio    | `9000` (API), `9001` (Console) | `minioadmin` / `minioadmin` | Локальная замена Yandex Object Storage. Web-консоль: `http://localhost:9001`. |
| mailhog  | `1025` (SMTP), `8025` (Web) | —                | Перехват SMTP-писем в dev. Все письма видны на `http://localhost:8025`. |

## Тесты

```bash
make test                  # unit, без внешних зависимостей
make test-integration      # integration с testcontainer Postgres
```

CI прогоняет `go test -race -shuffle=on -count=1 ./...` плюс `go vet` и `go build` (см. [`../.github/workflows/backend-ci.yml`](../.github/workflows/backend-ci.yml)).

Моки — только на внешние границы (Anthropic, S3, SMTP, OIDC). Внутри слоёв — реальные реализации с testcontainer Postgres. См. CLAUDE.md §«Стиль кода».

## Линт и форматирование

```bash
gofmt -l .                 # должен ничего не вывести
go fmt ./...               # форматирует на месте
make lint                  # golangci-lint
```

CI отклоняет PR, если `gofmt -l .` непустой или `golangci-lint run ./...` падает. Локальный setup для большинства IDE: включить «format on save» и линтер `golangci-lint`.

## Логи

- Формат — JSON через `slog` ([`internal/observability/`](./internal/observability)).
- PII-фильтр режет чувствительные поля: `email`, `password`, `otp`, `id_token`, `refresh_token`, `message_content`, фото-байты. **Никогда** не логировать содержимое сообщений и медиа.
- Уровень — через `LOG_LEVEL` (`debug` / `info` / `warn` / `error`).
- Sentry — опциональный, включается через `SENTRY_DSN`. В `dev` обычно выключен.

При добавлении нового чувствительного поля — обязательно добавить его в Replacer (см. `internal/observability/`), не полагаться на структурированные логи «по умолчанию».

## Troubleshooting

### Windows: окончания строк

В корневом `.editorconfig` зафиксирован `end_of_line = lf`. Если на Windows `git diff` показывает CRLF — выполнить один раз:

```bash
git config --global core.autocrlf input
```

### Доступ к Go-прокси из РФ

`proxy.golang.org` и `sum.golang.org` блокируются TLS-фильтром в РФ. Чистый `go mod download` упадёт с TLS-ошибкой. Варианты:

1. **VPN на момент `go mod download` / `go get`.** Самое простое для нечастых апдейтов.
2. **Локальный кэш + отключение sumdb.** Если в `$GOPATH/pkg/mod/cache/download/` уже есть нужные версии (от предыдущего скачивания через VPN), можно работать без сети:

   ```powershell
   # PowerShell — установить на сессию
   $env:GOPROXY = "file:///C:/Users/<you>/go/pkg/mod/cache/download,direct"
   $env:GOSUMDB = "off"
   ```

   ```bash
   # bash / git bash
   export GOPROXY="file:///$HOME/go/pkg/mod/cache/download,direct"
   export GOSUMDB=off
   ```

   `GOSUMDB=off` нужен, потому что `sum.golang.org` тоже блокируется. Проверка `go.sum` локально остаётся (хэши уже там).

3. **Корпоративный/публичный прокси** (например, `goproxy.cn`) — допустимо, если команда согласится. В этом случае `GOSUMDB=off` всё ещё нужен.

### Docker Desktop не запущен

`make dev` падает с `Cannot connect to the Docker daemon` — запустить Docker Desktop (Windows/macOS) или сервис `docker` (Linux).

### Порты заняты

Postgres `5432`, Redis `6379`, MinIO `9000`/`9001`, MailHog `1025`/`8025`. Если какой-то порт уже используется (свой Postgres / Redis на хосте) — создать `docker-compose.override.yml`:

```yaml
services:
  postgres:
    ports:
      - "55432:5432"
```

и поправить `POSTGRES_DSN` в `.env` под новый порт.

### Air не найден

`make dev` зовёт `air` напрямую. Если бинарь не установлен — `go install github.com/air-verse/air@latest`, проверить, что `$GOPATH/bin` в `PATH`. Альтернатива — `go run ./cmd/api` без live-reload.

## Связанные документы

- [`../ARCHITECTURE.md`](../ARCHITECTURE.md) — архитектура, API-контракты, схема БД, безопасность.
- [`../ROADMAP.md`](../ROADMAP.md) — этапы реализации и DoD.
- [`../CLAUDE.md`](../CLAUDE.md) — правила работы Claude Code в проекте (язык, инварианты, стиль).
- [`../SPEC.md`](../SPEC.md) — продуктовая спецификация.
