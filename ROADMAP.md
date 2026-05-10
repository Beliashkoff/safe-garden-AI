# ROADMAP — Safe Garden AI

Поэтапный план реализации. Каждый этап — самостоятельный кирпич с критериями приёмки (DoD), который можно мерджить и (после Этапа 0) деплоить в stage.

> Оценки в неделях даны для **одного fullstack-инженера**. С двумя (отдельно бэк и мобилка) сжимается на ~30%.

---

## Этап 0 — Фундамент (~1.5 недели)

**Цель:** репозиторий готов к разработке, локальное окружение поднимается одной командой, CI зелёный на пустом проекте.

### 0.1 Структура репозитория
- [ ] Создать каталоги `backend/`, `mobile/`, `infra/`, `.github/workflows/` согласно `ARCHITECTURE.md` §3.
- [ ] `.editorconfig`, корневой `README.md` с быстрым стартом.
- [ ] `.gitignore`: вывести из шаблонов Go + Flutter + Terraform + macOS/Windows.
- [ ] LICENSE — обсудить (предлагаю closed source, без файла).

### 0.2 Backend skeleton
- [ ] `go mod init github.com/<org>/safegarden-backend`
- [ ] Минимальный `cmd/api/main.go`: chi-роутер, `/healthz` + `/readyz`.
- [ ] `internal/config` через envconfig, `.env.example`.
- [ ] `Makefile`: `dev`, `test`, `lint`, `build`, `migrate-up`, `migrate-down`, `sqlc-gen`.
- [ ] `Dockerfile` (multi-stage, distroless).
- [ ] `docker-compose.yml`: postgres, redis, minio, mailhog.
- [ ] `.golangci.yml` с разумным набором линтеров.
- [ ] `air.toml` для live-reload.
- [ ] `slog`-handler с JSON-выводом и фильтром PII.
- [ ] Sentry init (опционально по env).

### 0.3 Mobile skeleton
- [ ] `flutter create` (template app).
- [ ] `pubspec.yaml`: подтянуть базовые пакеты (riverpod, go_router, dio, freezed, dev: build_runner, mocktail).
- [ ] `analysis_options.yaml`: `flutter_lints` + строгие правила (`prefer_const_constructors`, `unawaited_futures`, `avoid_print`).
- [ ] Корневая структура `lib/app`, `lib/core`, `lib/features`, `lib/widgets`, `lib/l10n`.
- [ ] Темизация (Material 3, light/dark), базовая локализация (RU).
- [ ] Заглушка экрана «логин» и «чат» с навигацией.

### 0.4 CI/CD bootstrap
- [ ] `.github/workflows/backend-ci.yml`: lint + test (без интеграционных).
- [ ] `.github/workflows/mobile-ci.yml`: analyze + test + build apk.
- [ ] Branch protection: PR-only мердж в `main`, требовать зелёный CI.

### 0.5 Документация
- [ ] `README.md` корневой: цель, ссылки на SPEC/ARCH/ROADMAP/CLAUDE, быстрый старт.
- [ ] `backend/README.md`: команды, переменные окружения.
- [ ] `mobile/README.md`: запуск, настройка iOS/Android симуляторов.

### 0.6 Внешние аккаунты и инфраструктура (заказчик)
- [ ] **Hetzner Cloud аккаунт** на иностранную карту/юрлицо заказчика. Сразу не покупаем VPS — только аккаунт.
- [ ] **Anthropic API аккаунт** на иностранное юрлицо/email; запрос production-ключа (Tier 1 на старте).
- [ ] **Yandex Cloud организация**, биллинг.
- [ ] **Yandex 360** — подключение почты для домена, создание `noreply@<domain>`, генерация SMTP-пароля.
- [ ] **Yandex SpeechKit** — сервис-аккаунт + API-ключ.
- [ ] **Домены:** регистрация (`<domain>` для api, `<domain>` для лендинга/политики), привязка к Yandex Cloud DNS.

### 0.7 Скелет llm-worker
- [ ] `cmd/llmworker/main.go` — заглушка, `/healthz` + `/v1/llm/messages` echo.
- [ ] `internal/llm/worker_client.go` — клиент с mTLS (заглушка).
- [ ] `internal/llm/mock_client.go` — фикстуры для dev/тестов.
- [ ] `Makefile`: `make worker-dev` — запуск worker'а локально на `:8081`.
- [ ] `infra/terraform/envs/prod` — описание Yandex VM + Hetzner CX22 + Managed PostgreSQL + Managed Redis + Object Storage bucket. **Без apply на этом этапе** — apply в Этапе 2, когда будет что деплоить.
- [ ] `infra/docker/compose/prod-yandex.yml` — Docker Compose для бэкенда (api + Caddy).
- [ ] `infra/docker/compose/prod-hetzner.yml` — Docker Compose для worker'а (llmworker + Caddy).

### DoD Этапа 0
- `make dev` поднимает локальное окружение с docker-compose.
- `flutter run` запускает заглушку с двумя экранами.
- CI зелёный на пустом коде.
- Любой новый разработчик может склонировать репо и запустить всё за < 30 минут.

---

## Этап 1 — Авторизация (~1.5 недели)

**Цель:** пользователь может зарегистрироваться/войти через Apple, Google или Email + OTP, токены хранятся безопасно, refresh работает.

### 1.1 Backend: модели и хранилище
- [ ] Миграции: `users`, `refresh_tokens`, `email_codes`, `audit_log`.
- [ ] sqlc-запросы: создание/поиск user, выпуск/поиск/ревок refresh-токена, OTP CRUD.
- [ ] `internal/auth/jwt.go`: генерация/проверка RS256, ротация ключей через `kid`.
- [ ] `internal/auth/oidc.go`: верификация Apple и Google id_token через JWKS (`coreos/go-oidc`).

### 1.2 Backend: эндпоинты
- [ ] `POST /v1/auth/apple`
- [ ] `POST /v1/auth/google`
- [ ] `POST /v1/auth/email/request` (с rate limit 3/час/email)
- [ ] `POST /v1/auth/email/verify` (≤ 5 попыток на код)
- [ ] `POST /v1/auth/refresh` (с ротацией)
- [ ] `POST /v1/auth/logout`
- [ ] `GET /v1/account`
- [ ] `DELETE /v1/account` (каскад + аудит; в этом этапе — без удаления медиа, доделаем в Этапе 3)
- [ ] Middleware `RequireAuth`: проверяет JWT, кладёт `user_id` в context.
- [ ] Middleware `RequestID`, `RealIP`, `Logger`, `Recoverer`.

### 1.3 Backend: email-провайдер
- [ ] Абстракция `Mailer` интерфейс. Реализации: `yandex360` (через SMTP `smtp.yandex.ru:465` SSL/TLS) и `dev` (в mailhog для локалки).
- [ ] Использовать `gomail.v2` или эквивалент. AUTH LOGIN, SMTPS.
- [ ] Креды: `SMTP_USERNAME=noreply@<domain>`, `SMTP_PASSWORD=<пароль приложения из Yandex 360>`.
- [ ] Тексты OTP-писем (RU + EN), HTML + text/plain. SPF/DKIM/DMARC настроены через Yandex 360.

### 1.4 Mobile: UI
- [ ] Экран онбординга (1 слайд) → экран логина.
- [ ] Кнопки: «Войти с Apple» (только iOS), «Войти с Google», «По email».
- [ ] Экран «введите email» → «введите код 6 цифр».
- [ ] Корректные тексты ошибок (нет сети, неверный код, истёк код, лимит).
- [ ] Loading/disabled состояния.

### 1.5 Mobile: интеграция
- [ ] `core/api/api_client.dart`: `dio` + интерсептор Auth (Bearer) + интерсептор Refresh (на 401 — вызвать /refresh, повторить).
- [ ] `core/storage/secure_token_store.dart` (через `flutter_secure_storage`).
- [ ] `features/auth/data/auth_repository.dart` + `auth_notifier.dart`.
- [ ] `sign_in_with_apple` и `google_sign_in` интеграция.
- [ ] Авто-вход при запуске (если есть refresh-токен).

### 1.6 Тесты
- [ ] Backend: integration-тесты на каждый эндпоинт (фейковый OIDC через locally signed JWT).
- [ ] Mobile: unit-тесты `auth_repository`, widget-тесты экранов.

### DoD Этапа 1
- Полный цикл регистрации/входа работает на iOS и Android против prod-домена (см. SPEC §8 D15 — stage появится после релиза).
- Refresh при 401 происходит прозрачно.
- Удаление аккаунта удаляет user в БД и отзывает refresh-токены.
- На prod-домене работает HTTPS через Caddy + Let's Encrypt.

---

## Этап 2 — Чат-MVP с Claude (текст) (~2 недели)

**Цель:** пользователь видит свой единственный чат, отправляет текст, получает стримящийся ответ от Claude.

### 2.1 Backend: модели чата
- [ ] Миграции: `conversations`, `messages`, `message_blocks`, `usage_log`.
- [ ] sqlc-запросы для чтения истории, добавления сообщений, обновления статусов.
- [ ] При первом запросе чата — авто-создание `conversation` для пользователя.

### 2.2 LLM-worker и интеграция Claude (см. ARCH §11)
- [ ] **На worker'е (Hetzner stage):**
  - Подключить `anthropic-sdk-go` непосредственно к `api.anthropic.com`.
  - `internal/llmworker/server.go` — HTTP-сервер с mTLS, эндпоинт `POST /v1/llm/messages` (SSE).
  - System prompt в `internal/llm/prompts/system_v1.md`, грузится через `embed.FS`. Доставляется в worker'а в payload или хранится локально (предпочтительнее в payload — единая версия из РФ-репо).
  - Базовый запрос: передача `messages`, стриминг ответа, проброс `usage` события.
  - Запрос Bedrock/Vertex как резерв — пока не реализуем, пишем абстракцию.
- [ ] **На бэкенде (РФ):**
  - `internal/llm/client.go` — интерфейс `Client.Send(ctx, req) (<-chan StreamEvent, error)`.
  - `internal/llm/worker_client.go` — реализация: mTLS POST на `LLM_WORKER_URL`, парсинг SSE.
  - `internal/llm/mock_client.go` — для unit-тестов.
  - `LLM_CLIENT_KIND` env-флаг: `worker` / `mock`.
- [ ] **Поднятие prod-инфраструктуры (terraform apply):**
  - Yandex Cloud VM (s-standard, 2 vCPU / 4 GB) + Caddy + Docker Compose.
  - Yandex Managed PostgreSQL (минимальный кластер, 1 нода для старта).
  - Yandex Managed Redis (минимальный кластер).
  - Yandex Object Storage bucket с приватным доступом.
  - Hetzner CX22 в Frankfurt с LUKS-encrypted volume + Caddy + Docker Compose.
- [ ] mTLS-сертификаты сгенерированы (CA + бэк + worker). На бэкенде — секрет читается из Yandex Lockbox; на worker'е — из `/etc/llmworker/.env` на encrypted volume (см. ARCH §8.6).

### 2.3 Backend: SSE-эндпоинт
- [ ] `POST /v1/messages` — handler:
  - валидация payload (только text в этом этапе),
  - сохранение user-message,
  - подгрузка истории,
  - вызов `llm.Client.Send(...)`, ретрансляция дельт в SSE на mobile,
  - сохранение assistant-message по завершении (или `cancelled` при разрыве).
- [ ] `GET /v1/conversation` — отдача истории с пагинацией.
- [ ] Rate-limit 20 RPS на `/messages` per user.
- [ ] `usage_log` запись при завершении (токены берутся из worker SSE-события `usage`).

### 2.4 Mobile: чат-UI
- [ ] Экран чата: список сообщений, инпут, кнопка «Отправить».
- [ ] Виджет `MessageBubble` (user / assistant), маркдаун-рендер для assistant (`flutter_markdown`).
- [ ] Загрузка истории при открытии (с индикатором).
- [ ] Локальный кэш в `drift` (offline-показ).

### 2.5 Mobile: SSE-клиент
- [ ] Парсер SSE поверх `dio` `ResponseType.stream`.
- [ ] `chat_notifier`: добавляет user-message локально → стримит дельты в pending assistant-message → завершает по `done`.
- [ ] Отмена через жест (или кнопку «Стоп») → отправляется DELETE на `/messages/:id` или просто закрытие соединения.

### 2.6 Тесты
- [ ] Backend: интеграционные (мок Claude через интерфейс), включая отмену.
- [ ] Mobile: golden-тесты пузырей сообщений, integration_test «отправить → получить».

### DoD Этапа 2
- Пользователь отправляет текст, видит стримящийся ответ.
- История сохраняется и подгружается при перезапуске.
- Отмена работает корректно (нет пустых сообщений).
- Prod-окружение поднято и использует реальный Claude через `llm-worker` на Hetzner.
- Worker доступен только с IP бэкенда в Yandex Cloud (Hetzner Cloud Firewall + IP-allowlist + mTLS).
- До публикации в сторы prod-окружение используется и для нашего ручного тестирования (см. D15).

---

## Этап 3 — Фото в чате (~1 неделя)

**Цель:** пользователь отправляет фото вместе с текстом, Claude его анализирует.

### 3.1 Backend
- [ ] Миграция: `uploads`.
- [ ] `POST /v1/uploads/presign` — генерация presigned PUT (Yandex Object Storage / minio в dev).
- [ ] Валидация content-type/size до выдачи presigned.
- [ ] Расширение `POST /v1/messages`: блок `image_ref { storage_key }`. Проверка ownership через `uploads.user_id`.
- [ ] Загрузка фото из Object Storage, передача в Claude как `image` block (base64).
- [ ] Конвертация HEIC → JPEG на стороне сервера (`disintegration/imaging` или `chai2010/webp`), если необходимо.

### 3.2 Backend: каскадное удаление медиа
- [ ] При `DELETE /v1/account` → фоновый job удаляет всё под префиксом `u/{user_id}/` в Object Storage.
- [ ] Фоновый GC: каждые сутки — удаление `uploads` старше 7 дней и `used=false`.

### 3.3 Mobile
- [ ] Кнопка-скрепка с выбором: «Камера» / «Галерея».
- [ ] `permission_handler` для камеры/фото.
- [ ] `image_picker` + `flutter_image_compress` (1920×1080, q=85).
- [ ] Прогресс загрузки: presign → PUT → POST /messages.
- [ ] Превью фото в bubble сообщения.
- [ ] Поддержка нескольких фото за сообщение (до 4).

### 3.4 Тесты
- [ ] Backend: integration на presign + проверка ownership (попытка от чужого user'а → 403).
- [ ] Mobile: widget-тест выбора фото с моком permission'ов.

### DoD Этапа 3
- Пользователь фотографирует томат, получает диагностический ответ от Claude.
- Удаление аккаунта удаляет фото из Object Storage.
- Лимиты размера и типа работают на сервере и клиенте.

---

## Этап 4 — Голосовые сообщения (~1 неделя)

**Цель:** пользователь записывает голосовое, получает ответ.

### 4.1 Провайдер: Yandex SpeechKit v3 (закрыто Q4)
- [ ] Сервис-аккаунт + API-ключ из Yandex Cloud (создан в Этапе 0).
- [ ] **Конвертация аудио:** SpeechKit не принимает m4a/AAC напрямую. Перед отправкой бэкенд конвертирует через `ffmpeg` (`-i in.m4a -c:a libopus -ar 16000 -ac 1 out.ogg`). `ffmpeg` устанавливается в Docker-образ бэкенда.
- [ ] Резервный провайдер: GigaChat (Sber GigaAM-v3) — не реализуем сейчас, держим как запасной интерфейс.

### 4.2 Backend
- [ ] Расширение presign на `audio` (m4a/aac/mp3, ≤ 25 МБ, ≤ 60s — длительность валидируется после конвертации).
- [ ] `internal/audio/transcriber.go` — интерфейс `Transcriber.Transcribe(ctx, key, lang) (text, durationMs, error)`.
- [ ] `internal/audio/speechkit.go` — реализация SpeechKit v3 через REST (sync recognize). Заголовок `Authorization: Api-Key <KEY>`.
- [ ] `internal/audio/converter.go` — обёртка над `ffmpeg` через `os/exec`.
- [ ] При обработке `audio_ref` в `/messages`: загрузить аудио из Object Storage → конвертировать → транскрибировать → передать как text-блок в Claude (с пометкой `[голосовое сообщение]: ...`).
- [ ] Сохранять транскрипцию как `message_blocks.type = 'transcription'` рядом с `audio` блоком — UI показывает плеер и текст.

### 4.3 Mobile
- [ ] Кнопка-микрофон (long-press to record, как в Telegram, или tap-to-toggle).
- [ ] `record` — запись в m4a/AAC.
- [ ] Индикатор уровня звука + таймер.
- [ ] Возможность отмены (свайп влево).
- [ ] Прослушивание перед отправкой (опционально для v1).
- [ ] Отображение в чате: плеер + текст транскрипции.

### 4.4 Тесты
- [ ] Backend: мок транскрайбера, тест полного цикла.
- [ ] Mobile: тест на разрешения микрофона.

### DoD Этапа 4
- Пользователь записал «у меня помидоры вянут», получил полный ответ.
- Транскрипция корректно отображается в чате.

---

## Этап 5 — Каталог удобрений + Tool Use (~1 неделя)

**Цель:** Claude вызывает `recommend_fertilizer`, в чате появляется нативная карточка продукта.

> ⚠️ **Каталог удобрений** заказчик передаст позже (Q2 в SPEC). На этом этапе делаем **архитектуру** (таблица `fertilizers`, tool, callback из worker'а в РФ-бэк, UI-карточка). Если каталог пуст — tool возвращает «нет подходящего удобрения», Claude отвечает текстом без карточки.

### 5.1 Каталог
- [ ] Миграция: `fertilizers`.
- [ ] Сидер `cmd/seed/main.go` принимает CSV из `infra/data/fertilizers.csv`. До получения данных от заказчика — пустой файл-плейсхолдер с заголовком и одной демо-записью для тестов.
- [ ] Минимальная админка — отдельным шагом v2; в v1 правим SQL/CSV.

### 5.2 Backend и worker: Tool use
- [ ] **На worker'е:** добавить `tools: [recommend_fertilizer]` в запросы к Anthropic. При получении `tool_use` — RPC обратно на РФ-бэкенд (`POST /internal/v1/tools/fertilizer` через mTLS), полученный `tool_result` отправляется в Claude, цикл продолжается.
- [ ] **На РФ-бэкенде:** эндпоинт `/internal/v1/tools/fertilizer` (mTLS only) — выполняет SQL-запрос из ARCH §6.4, возвращает 0–3 продукта.
- [ ] **На worker'е:** SSE-событие `fertilizer_card` параллельно с дельтами текста.
- [ ] **На бэкенде:** ретрансляция `fertilizer_card` на mobile, сохранение блока `fertilizer_card` в `message_blocks.metadata`.

### 5.3 Mobile
- [ ] Виджет `FertilizerCard`: фото, название, описание, кнопка «Подробнее» → `url_launcher` на deeplink.
- [ ] Парсинг `fertilizer_card` event → вставка карточки в bubble.
- [ ] Аналитика тапов по карточкам (внутренняя — в `usage_log`).

### 5.4 Промпт-инжиниринг
- [ ] Финализировать system prompt: формат ответа, обязательность tool call, тон.
- [ ] Eval-набор: 20 типичных кейсов (томаты желтеют, огурцы вянут, плохо плодоносит и т.д.) — прогон руками, фиксация качества.
- [ ] Включить prompt caching для system prompt + tool definitions.

### 5.5 Тесты
- [ ] Backend: тест tool-use цикла с моком Claude.
- [ ] Eval-скрипт: прогон 20 кейсов через stage Claude, ручная оценка.

### DoD Этапа 5
- На фото томата с признаками дефицита калия Claude возвращает диагноз + tool call → карточка с подходящим удобрением.
- Карточка кликабельна, ведёт на сайт компании.

---

## Этап 6 — Полировка, безопасность, соответствие (~2 недели)

**Цель:** прод-готовое качество, прохождение чек-листов сторов, безопасность подтверждена.

### 6.1 Безопасность (см. ARCH §8)
- [ ] Полная проверка `RequireAuth` middleware на всех приватных эндпоинтах.
- [ ] Проверка ownership при доступе к `storage_key`, `message_id`, `conversation_id`.
- [ ] Penetration-чек-лист OWASP Mobile Top 10.
- [ ] Логи: убедиться, что ни email, ни OTP, ни id_token не попадают в логи.
- [ ] Запуск `gosec` и `nancy` на бэкенде в CI.
- [ ] Запуск `dependabot` для обоих проектов.

### 6.2 Производительность
- [ ] Профилирование `/messages` под нагрузкой (k6 или vegeta).
- [ ] Индексы Postgres проверены через `EXPLAIN ANALYZE`.
- [ ] Mobile: проверить отсутствие jank на slowest target device (Android 8 low-end).

### 6.3 Юридическое
- [ ] Privacy Policy — RU + EN, опубликована на сайте.
- [ ] Terms of Service — RU + EN.
- [ ] Согласие на обработку ПДн при регистрации.
- [ ] 152-ФЗ: уведомление в Роскомнадзор (если требуется по объёму).
- [ ] Apple §5.1.1: data deletion из приложения (✅ уже есть).
- [ ] Apple §4.8: Apple Sign-In присутствует там, где Google.
- [ ] Google Play Data Safety form заполнена.

### 6.4 UX-полировка
- [ ] Онбординг: 1-3 экрана объяснения возможностей.
- [ ] Пустое состояние чата: подсказка «Сфотографируйте растение или опишите проблему».
- [ ] Состояния ошибок: нет сети, ошибка Claude, лимит запросов.
- [ ] Iconography, app icon, splash screen.
- [ ] Полная ревизия копирайтинга (RU).

### 6.5 Локализация (только RU в v1, но инфраструктура готова)
- [ ] Все строки через `intl` ARB.
- [ ] Бэк: `Accept-Language` → выбор шаблонов писем и сообщений ошибок.

### 6.6 Аналитика и мониторинг
- [ ] Дашборды в Grafana: RPS, latency, ошибки, токены, стоимость.
- [ ] Алерты настроены и проверены (тест-алерт на email/Telegram).
- [ ] Sentry интегрирован, источники подтверждены.

### DoD Этапа 6
- Пройден внутренний security-ревью (см. `agent-skills:security-and-hardening`).
- Все приватные эндпоинты защищены и проверены тестами.
- Стейджинг прошёл нагрузочное тестирование (50 RPS на `/messages` без падений).

---

## Этап 7 — Деплой и релиз в сторы (~1.5–2 недели)

**Цель:** приложение опубликовано в App Store и Google Play.

> ℹ️ Apple Developer Account и Google Play Console заказчик оформляет в начале этого этапа (закрытие Q7 в SPEC). Privacy Policy и ToS также готовятся к этому моменту (Q6).

### 7.1 Production-готовность

> Prod-окружение уже поднято в Этапе 2 и использовалось для нашего тестирования. На Этапе 7 готовим его к реальным пользователям.

- [ ] Аудит конфигов: backup-расписание PG, retention, ротация Caddy-логов.
- [ ] Бэкапы PostgreSQL — **проверены восстановлением** на отдельной VM.
- [ ] Sentry/Grafana дашборды проверены, алерты приходят.
- [ ] Hetzner Cloud Firewall: разрешён только IP Yandex Cloud VM.
- [ ] Yandex Cloud Security Group: разрешён только трафик от мобильных клиентов на 443 (Caddy).
- [ ] Очистка тестовых данных из prod-БД (тестовые users, conversations, uploads).
- [ ] **Подъём отдельного stage-окружения** (отдельная пара VM, отдельная БД, отдельный Anthropic-ключ) — для постpелизных фич. См. SPEC §8 D15.

### 7.2 Apple App Store
- [ ] Apple Developer Account ($99/год) — оформляет заказчик.
- [ ] Bundle ID, App ID, profiles.
- [ ] App Store Connect: создание приложения.
- [ ] Подготовка ассетов: иконки, скрины (6.5", 5.5"), описание, ключевые слова, превью-видео (опционально).
- [ ] App Privacy: data collection answers.
- [ ] Sign-In with Apple — конфигурация в Capabilities.
- [ ] Загрузка билда через Xcode / Transporter.
- [ ] TestFlight beta — внутренние тестировщики.
- [ ] Сабмит на ревью (учесть, что обычно 1-3 дня).

### 7.3 Google Play
- [ ] Google Play Console ($25 разово) — оформляет заказчик.
- [ ] Создание приложения, заполнение Data Safety, Content Rating, Privacy Policy.
- [ ] Подготовка ассетов: иконка, скрины (телефон + 7" + 10"), feature graphic, описание.
- [ ] Загрузка AAB через Play Console / fastlane.
- [ ] Internal testing → Closed testing → Production.

### 7.4 Финальный smoke
- [ ] Ручной чек на свежеустановленных приложениях из TestFlight и Play Internal:
  - регистрация всеми тремя способами,
  - отправка фото, голоса, текста,
  - получение карточки удобрения,
  - удаление аккаунта,
  - переустановка → автологин при сохранённых токенах не должен происходить (нужен повторный вход).

### 7.5 Запуск
- [ ] Staged rollout: 1% → 10% → 50% → 100% за 5-7 дней.
- [ ] Дежурство первые 48 часов после публикации.
- [ ] Готовность hotfix-релиза (Apple expedited review план).

### DoD Этапа 7
- Приложение доступно публично в обоих сторах.
- Метрики собираются.
- Crash-free rate ≥ 99.5% за первые 48 часов.

---

## После v1 (бэклог v2+)

- [ ] Push-уведомления (напоминания об уходе, сезонные подсказки).
- [ ] Несколько чатов / темы.
- [ ] RAG: pgvector + собственная база знаний по болезням и культурам.
- [ ] Реакции на ответы (👍/👎) → файнтюнинг промпта.
- [ ] Подписка / monetisation (если решено).
- [ ] Веб-версия для дублирования каталога/админки.
- [ ] Мини-админка для редактирования каталога удобрений.
- [ ] Английская локализация.
- [ ] iPad / Android Tablet полноценная адаптация.

---

## Критические зависимости (что блокирует старт)

| Зависимость                                                           | Этап       | Кто решает          | Статус   |
| --------------------------------------------------------------------- | ---------- | ------------------- | -------- |
| Hetzner Cloud аккаунт + Anthropic API-ключ (зарубежная карта)         | Этап 0–2   | Заказчик            | открыто  |
| Yandex Cloud организация и биллинг                                    | Этап 0     | Заказчик            | открыто  |
| Yandex 360 для домена (SMTP)                                          | Этап 1     | Заказчик / DevOps   | открыто  |
| Yandex SpeechKit ключ                                                 | Этап 4     | DevOps              | открыто  |
| Apple Developer Account                                               | Этап 7     | Заказчик            | отложено (Q7) |
| Google Play Console                                                   | Этап 7     | Заказчик            | отложено (Q7) |
| Дизайн (иконка, ассеты, экранчики) — делаем сами                      | Этап 1+    | Команда             | закрыто (D14) |
| Каталог удобрений (CSV с полями из ARCH §6.1)                         | Этап 5     | Заказчик / контент  | отложено (Q2) |
| Privacy Policy / ToS на двух языках                                   | Этап 7     | Юрист               | отложено (Q6) |
| Домен и DNS                                                           | Этап 0     | DevOps / заказчик   | открыто  |
