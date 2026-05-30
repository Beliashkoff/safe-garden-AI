# ROADMAP — Safe Garden AI

Поэтапный план реализации. Каждый этап — самостоятельный кирпич с критериями приёмки (DoD), который можно мерджить и (после Этапа 2) выкатывать в prod. До публикации в сторы prod-окружение используется и для нашего ручного тестирования (см. SPEC §8 D15 — stage-окружения нет).

> Оценки в неделях даны для **одного fullstack-инженера**. С двумя (отдельно бэк и мобилка) сжимается на ~30%.

---

## Этап 0 — Фундамент (~1.5 недели)

**Цель:** репозиторий готов к разработке, локальное окружение поднимается одной командой, CI зелёный на пустом проекте.

### 0.1 Структура репозитория ✅
- [x] Создать каталоги `backend/`, `mobile/`, `infra/`, `.github/workflows/` согласно `ARCHITECTURE.md` §3.
- [x] `.editorconfig`, корневой `README.md` с быстрым стартом.
- [x] `.gitignore`: вывести из шаблонов Go + Flutter + Terraform + macOS/Windows.
- [x] LICENSE — обсудить (предлагаю closed source, без файла). _Решение: closed source, без файла._

### 0.2 Backend skeleton ✅
- [x] `go mod init github.com/<org>/safegarden-backend` — модуль `github.com/Beliashkoff/safe-garden-AI/backend`.
- [x] Минимальный `cmd/api/main.go`: chi-роутер, `/healthz` + `/readyz`.
- [x] `internal/config` через envconfig, `.env.example`.
- [x] `Makefile`: `dev`, `test`, `lint`, `build`, `migrate-up`, `migrate-down`, `sqlc-gen`.
- [x] `Dockerfile` (multi-stage, distroless).
- [x] `docker-compose.yml`: postgres, redis, minio, mailhog.
- [x] `.golangci.yml` с разумным набором линтеров.
- [x] `air.toml` для live-reload.
- [x] `slog`-handler с JSON-выводом и фильтром PII — в `internal/observability/`.
- [x] Sentry init (опционально по env) — через `SENTRY_DSN`.

### 0.3 Mobile skeleton ✅
- [x] `flutter create` (template app). _Пакет `agronom_ai`, `mobile/pubspec.yaml`._
- [x] `pubspec.yaml`: подтянуть базовые пакеты (riverpod, go_router, dio, freezed, dev: build_runner, mocktail).
- [x] `analysis_options.yaml`: `flutter_lints` + строгие правила (`prefer_const_constructors`, `unawaited_futures`, `avoid_print`).
- [x] Корневая структура `lib/app`, `lib/core`, `lib/features`, `lib/widgets`, `lib/l10n`. _`lib/core` и `lib/widgets` создадим в Этапе 1, когда туда появится содержимое (избегаем пустых каталогов)._
- [x] Темизация (Material 3, light/dark), базовая локализация (RU) — `lib/app/theme.dart`, `lib/l10n/app_ru.arb` + `app_en.arb`.
- [x] Заглушка экрана «логин» и «чат» с навигацией — `lib/features/auth/presentation/`, `lib/features/chat/presentation/`, роутер в `lib/app/router.dart`.

### 0.4 CI/CD bootstrap ✅
- [x] `.github/workflows/backend-ci.yml`: lint + test (без интеграционных).
- [x] `.github/workflows/mobile-ci.yml`: analyze + test + build apk (debug на PR, release на push в main).
- [x] Branch protection: PR-only мердж в `main`, требовать зелёный CI. _Инструкция по включению — в корневом `README.md` §«Настройка branch protection». Само включение — off-repo действие в GitHub Settings, выполняется владельцем репо._

### 0.5 Документация ✅
- [x] `README.md` корневой: цель, ссылки на SPEC/ARCH/ROADMAP/CLAUDE, быстрый старт. _Актуализирован Quick Start (рабочие команды) и блок Статус 0.1–0.4 ✅._
- [x] `backend/README.md`: команды, переменные окружения. _Полный гайд: prerequisites, env, Makefile-цели, happy path, Troubleshooting (Windows + Go-прокси из РФ)._
- [x] `mobile/README.md`: запуск, настройка iOS/Android симуляторов. _Полный гайд: prerequisites, команды, симуляторы Android/iOS, CI-jobs, Troubleshooting (Flutter SDK на Windows, отсутствие Android SDK)._

### 0.6 Внешние аккаунты и инфраструктура (заказчик)
- [ ] **HostKey аккаунт** (hostkey.ru, ООО «АЙТИБ») на юрлицо/карту заказчика, с пополнением баланса в рублях. Сразу не покупаем VPS — только аккаунт. _Принятые юридические риски — в ARCHITECTURE.md §11.7._
- [ ] **Anthropic API аккаунт** на иностранное юрлицо/email; запрос production-ключа (Tier 1 на старте).
- [ ] **Yandex Cloud организация**, биллинг.
- [ ] **Yandex 360** — подключение почты для домена, создание `noreply@<domain>`, генерация SMTP-пароля.
- [ ] **Yandex SpeechKit** — сервис-аккаунт + API-ключ.
- [ ] **Домен `agronomai.site`** (заказчик уже владеет): перевод NS на Yandex Cloud DNS, создание зоны, MX/SPF/DKIM/DMARC для `noreply@agronomai.site`. Поддомены: `api.agronomai.site` (бэкенд, A-record в Этапе 2.2), `agronomai.site` (apex — лендинг, Этап 7.2), `worker.agronomai.site` (опционально, HostKey VM, Этап 2.2). Подробный runbook — в `backend/README.md` §«Регистрация внешних аккаунтов».

### 0.7 Скелет llm-worker ✅
- [x] `cmd/llmworker/main.go` — заглушка, `/healthz` + `/v1/llm/messages` echo. _Echo-SSE по контракту ARCH §11.3 (`message_started` → `delta`* → `usage` → `done`); защита от PII в payload (CLAUDE.md инвариант №5)._
- [x] `internal/llm/worker_client.go` — клиент с mTLS (заглушка). _Tls.Config из путей CA/cert/key, SSE-парсер, отмена через ctx._
- [x] `internal/llm/mock_client.go` — фикстуры для dev/тестов. _Дефолтная фикстура + словарь по `Model`._
- [x] `Makefile`: `make worker-dev` — запуск worker'а локально на `:8081`.
- [x] `infra/terraform/envs/prod` — описание Yandex VM + HostKey VM (`vm.v2-nano` Frankfurt) + Managed PostgreSQL + Managed Redis + Object Storage bucket. Terraform-модуль worker'а параметризовать `provider = "hostkey" | "hetzner" | "ovh"`, чтобы готов был DR-переезд (см. ARCH §11.7). **Без apply на этом этапе** — apply в Этапе 2, когда будет что деплоить. _Модули в `infra/terraform/modules/`, env в `envs/prod/`, state local; CI прогоняет `fmt -check` и `validate`. На HostKey TF-провайдера нет — `null_resource` + ручной runbook в `modules/worker-vm/README.md`._
- [x] `infra/docker/compose/prod-yandex.yml` — Docker Compose для бэкенда (api + Caddy).
- [x] `infra/docker/compose/prod-llmworker.yml` — Docker Compose для worker'а (llmworker + Caddy). Имя файла нейтрально к провайдеру, т.к. образ один и тот же для HostKey/Hetzner/OVH. _mTLS-client_auth терминируется Caddy; worker слушает HTTP внутри docker-сети._

### DoD Этапа 0
- `make dev` поднимает локальное окружение с docker-compose.
- `flutter run` запускает заглушку с двумя экранами.
- CI зелёный на пустом коде.
- Любой новый разработчик может склонировать репо и запустить всё за < 30 минут.

---

## Этап 1 — Авторизация (~1.5 недели)

**Цель:** пользователь может зарегистрироваться/войти через Apple, Google или Email + OTP, токены хранятся безопасно, refresh работает.

### 1.1 Backend: модели и хранилище ✅
- [x] Миграции: `users`, `refresh_tokens`, `email_codes`, `audit_log`. _DDL дословно из ARCH §6.1; индексы `(token_hash) UNIQUE`, partial `(user_id) WHERE revoked_at IS NULL`, `(email, expires_at)`, `(user_id, created_at DESC)`._
- [x] sqlc-запросы: создание/поиск user, выпуск/поиск/ревок refresh-токена, OTP CRUD. _`internal/storage/queries/` + сгенерированный `internal/storage/db/`. `Store.ExecTx` для атомарной ротации refresh-токена. Интеграционные тесты под `make test-integration` (testcontainers)._
- [x] `internal/auth/jwt.go`: генерация/проверка RS256, ротация ключей через `kid`. _Multi-key `KeysDir` + single-file fallback. Parser: `WithValidMethods([RS256])` отвергает `alg=none`/HS256-confusion, `WithExpirationRequired`, leeway 60s, lookup по `kid` из header._
- [x] `internal/auth/oidc.go`: верификация Apple и Google id_token через JWKS (`coreos/go-oidc`). _Apple — обязательный nonce через `subtle.ConstantTimeCompare`, identity = `apple_sub` (email не доверяем, ARCH §11.1). Google — manual aud-allowlist для iOS+Android client_id._

### 1.2 Backend: эндпоинты ✅
- [x] `POST /v1/auth/apple`
- [x] `POST /v1/auth/google`
- [x] `POST /v1/auth/email/request` (с rate limit 3/час/email) _DB-baseline через `CountRecentEmailCodes` за интерфейсом `ratelimit.Limiter`; Redis-уровень — в 2.3._
- [x] `POST /v1/auth/email/verify` (≤ 5 попыток на код) _Атомарный `IncrementEmailCodeAttempts` до сверки — cap держится и на неверных попытках._
- [x] `POST /v1/auth/refresh` (с ротацией) _Атомарная ротация в `ExecTx`; предъявление revoked/expired токена → `RevokeAllUserRefreshTokens` + аудит `refresh_reuse_detected`._
- [x] `POST /v1/auth/logout` _Идемпотентно: отзыв предъявленного refresh._
- [x] `GET /v1/account`
- [x] `DELETE /v1/account` (каскад + аудит; в этом этапе — без удаления медиа, доделаем в Этапе 3) _Soft-delete (обнуляет email/sub) + revoke all + аудит `account_deleted`, атомарно._
- [x] Middleware `RequireAuth`: проверяет JWT, кладёт `user_id` в context. _`internal/transport/http/middleware`; `user_id` через типизированный `ctxkey`._
- [x] Middleware `RequestID`, `RealIP`, `Logger`, `Recoverer`. _chi RequestID/RealIP/Recoverer + `observability.AccessLog` (роль «Logger»)._

_Слои: `transport/http` (хендлеры + `httperr` — единый формат ошибок ARCH §4.7) → `usecase/auth` (Service: оркестрация, auto-link аккаунтов по email) → примитивы `internal/auth` + `storage`. Auto-link: sign-in с Apple/Google привязывается к существующему аккаунту при совпадении email (Apple — без private-relay; Google — при `email_verified`). **Swagger:** spec-first OpenAPI 3.0 (`internal/transport/http/docs/openapi.yaml`, embed) + Swagger UI на `/v1/docs` (флаг `DOCS_ENABLED`). Тесты: unit (usecase-хелперы, middleware, mailer, ratelimit) + integration хендлеров (testcontainers + fake OIDC) под `make test-integration`._

### 1.3 Backend: email-провайдер ✅
- [x] Абстракция `Mailer` интерфейс (`internal/mailer`). Реализации: SMTP (`yandex360` через `smtp.yandex.ru:465` implicit TLS **и** `dev`/MailHog `localhost:1025` plaintext — выбор по `SMTP_TLS`) + log-fallback при пустом `SMTP_HOST`.
- [x] ~~`gomail.v2`~~ → stdlib `net/smtp` (без сторонней зависимости; go-mail недоступен из РФ-прокси). AUTH LOGIN (кастомный, поверх implicit TLS), SMTPS на 465.
- [x] Креды: `SMTP_USERNAME=noreply@<domain>`, `SMTP_PASSWORD=<пароль приложения из Yandex 360>` (обязательны в prod).
- [x] Тексты OTP-писем (RU + EN), HTML + text/plain (multipart/alternative, quoted-printable, RFC 2047 subject). _SPF/DKIM/DMARC для `noreply@agronomai.site` — DNS-настройка, часть Этапа 0.6._

### 1.4 Mobile: UI ✅
- [ ] Экран онбординга (1 слайд) → экран логина. _Отложено: опциональный слайд, не входил в скоуп шага; login-экран несёт заголовок+подзаголовок. Сделаем при UX-полише._
- [x] Кнопки: «Войти с Apple» (только iOS), «Войти с Google», «По email». _`login_screen.dart` (ConsumerStatefulWidget), Apple скрыт вне iOS/macOS._
- [x] Экран «введите email» → «введите код 6 цифр». _`email_request_screen.dart` + `email_verify_screen.dart` (авто-submit на 6 цифр, resend)._
- [x] Корректные тексты ошибок (нет сети, неверный код, истёк код, лимит). _`auth_error_message.dart` мапит `ApiException.code`/`NetworkException` в локализованные строки (RU/EN)._
- [x] Loading/disabled состояния. _Локальные `_busy`-флаги в экранах; глобальный AsyncLoading — только на bootstrap._

### 1.5 Mobile: интеграция ✅
- [x] `core/network/api_client.dart`: `dio` + интерсептор Auth (Bearer) + интерсептор Refresh (на 401 — вызвать /refresh, повторить). _Один общий refresh с мьютексом, без рекурсии, retry один раз; при провале refresh → полный логаут._
- [x] `core/storage/secure_token_store.dart` (через `flutter_secure_storage`). _Интерфейс `TokenStore` для тестируемости._
- [x] `features/auth/data/auth_repository.dart` + контроллер. _`auth_repository.dart` + `application/auth_controller.dart` (AsyncNotifier); OAuth-обёртки в `data/oauth_providers.dart`._
- [x] `sign_in_with_apple` и `google_sign_in` интеграция. _Apple: raw nonce + sha256; Google v7: `initialize(serverClientId)` + `authenticate()`. Бэк-allowlist расширен `GOOGLE_CLIENT_ID_WEB`. E2E отложено до OAuth-аккаунтов (0.6) и macOS (7)._
- [x] Авто-вход при запуске (если есть refresh-токен). _`AuthController.build()` → `tryRestoreSession`; `go_router` redirect-гард по `authStatusProvider` (splash → login/chat)._

### 1.6 Тесты ✅
- [x] Backend: integration-тесты на каждый эндпоинт (фейковый OIDC через locally signed JWT). _Сделано в 1.2: `internal/transport/http/handler/*_test.go` (testcontainers + fake IdP), под `make test-integration`._
- [x] Mobile: unit-тесты `auth_repository` + контроллера + сетевого слоя, widget-тесты экранов. _33 теста; `flutter analyze --fatal-infos` + `flutter test` зелёные. On-device E2E отложен (нет Android SDK/macOS)._

### DoD Этапа 1
- Полный цикл регистрации/входа работает на iOS и Android против prod-домена `api.agronomai.site` (см. SPEC §8 D15 — отдельного stage-окружения нет).
- Refresh при 401 происходит прозрачно.
- Удаление аккаунта удаляет user в БД и отзывает refresh-токены.
- На prod-домене работает HTTPS через Caddy + Let's Encrypt.

---

## Этап 2 — Чат-MVP с Claude (текст) (~2 недели)

**Цель:** пользователь видит свой единственный чат, отправляет текст, получает стримящийся ответ от Claude.

### 2.1 Backend: модели чата ✅
- [x] Миграции: `conversations`, `messages`, `message_blocks`, `usage_log` (+ `uploads`, `fertilizers` — вся §6.1 заложена сразу). _Файлы `migrations/0006_…0011_…`, по одному объекту на файл, именованные индексы, FK `ON DELETE CASCADE`, CHECK на `role`/`status`/`type`. В `fertilizers` добавлена колонка `priority` (синк с ARCH §6.4 — в §6.1 DDL её не было)._
- [x] sqlc-запросы для чтения истории, добавления сообщений, обновления статусов. _`internal/storage/queries/{conversations,messages,message_blocks,uploads,fertilizers,usage_log}.sql`. Keyset-пагинация (`ListRecentMessages`/`ListMessagesBefore` с tiebreak по id), `Create/Complete/UpdateMessageStatus`, `DeleteMessage :execrows` (ownership), `ListBlocksByMessageIDs` (без N+1), `RecommendFertilizers` (§6.4), `InsertUsage`/`SumUserTokensSince`._
- [x] При первом запросе чата — авто-создание `conversation` для пользователя. _`GetOrCreateConversation` — `INSERT … ON CONFLICT (user_id) DO UPDATE … RETURNING` (атомарно, один чат на user)._

_Schema-only этап: HTTP/usecase чата — 2.3, llm-worker — 2.2. Тесты: `chat_integration_test.go` (testcontainers) — get-or-create, пагинация, статусы, ownership-delete, блоки/metadata, uploads, recommend, usage-sum, каскады (включая hard-delete user). Расхождение soft-delete (1.2) ↔ каскад (ARCH §6.3) задокументировано; очистка чата при удалении аккаунта — Этап 3._

### 2.2 LLM-worker и интеграция Claude (см. ARCH §11)
- [x] **На worker'е (HostKey Frankfurt, prod):** _код готов; реальный E2E с Claude — после 0.6 (ключ + не-РФ egress)._
  - [x] Подключить `anthropic-sdk-go` (`v1.45.0`) к `api.anthropic.com` — импорт только в `internal/llmworker/anthropic.go` (инвариант №5).
  - [x] `internal/llmworker/server.go` — `POST /v1/llm/messages` (SSE) за провайдер-абстракцией (`provider`/`eventSink`); mTLS терминируется Caddy перед worker'ом (§8.6).
  - [x] System prompt `internal/llm/prompts/system_v1.md` через `embed.FS` (`prompts.SystemV1()`), доставляется в payload из РФ-репо.
  - [x] Стриминг `Messages.NewStreaming` → маппинг событий в SSE (`message_started`/`delta`/`tool_use`/`usage`/`done`/`error`), `usage` из накопленного `Message.Usage`; prompt caching (`cache_control: ephemeral`) на system+tools.
  - [x] Резерв Bedrock/Vertex/OpenRouter — абстракция `provider` заложена (echo-fallback без ключа); сами реализации позже (§11.5).
- [x] **На бэкенде (РФ):** _уже было готово в 0.7; подтверждено._
  - [x] `internal/llm/client.go`, `worker_client.go` (mTLS+SSE), `mock_client.go`, `factory.go` (`LLM_CLIENT_KIND` worker|mock). Модель — `internal/llm/model.go` (`DefaultModel`).
- [ ] **Поднятие prod-инфраструктуры (terraform apply):** _код/скрипты готовы, сам apply — runbook оператора (требует аккаунтов 0.6); см. `infra/terraform/envs/prod/README.md` §«runbook первого apply»._
  - [x] Terraform-модули (yandex-vm/postgres/redis/s3 + worker-vm) + remote-state backend-шаблон (`versions.tf`); `fmt`/`validate` в CI.
  - [ ] Сам `terraform apply` (Yandex VM + Managed PG/Redis + Object Storage + Lockbox; HostKey `vm.v2-nano` Frankfurt + firewall 443 только с Yandex-IP) — оператор после 0.6.
- [x] mTLS-инструментарий: `infra/mtls/gen-certs.sh` (CA + worker + api-client) + `infra/scripts/setup-luks.sh` (LUKS-том секретов) + Go-тест `buildMTLSConfig`. _Генерация/раздача сертификатов (Lockbox для бэка, LUKS для worker'а, §8.6) — шаг runbook'а оператора._

### 2.3 Backend: SSE-эндпоинт ✅
- [x] `POST /v1/messages` — handler (text-only): валидация (image/audio → 415), сохранение user-message+блока (ExecTx), подгрузка истории (~20, ARCH §7.4), `llm.Client.Send`, ретрансляция дельт в SSE; финал — `complete` (+токены+блок+usage) / `cancelled` при разрыве (на detached-контексте, сохраняет частичный ответ) / `failed`. _`internal/usecase/chat` (Service+Sink+relay) + `handler/chat.go` (sseSink, lazy-headers: pre-stream ошибки → JSON, далее → SSE). uid_hash = sha256(user_id+UID_HASH_PEPPER) — единственный идентификатор в payload (§11.4)._
- [x] `GET /v1/conversation` (+`/conversation/messages`) — keyset-пагинация (курсор base64 `created_at:id`), хронологический порядок; `DELETE /v1/messages/{id}` (ownership) добавлен (ARCH §4.3).
- [x] Rate-limit 20 RPS на `/messages` per user — Redis `redis_rate` (`ratelimit.RedisLimiter`, fail-open); dev без `REDIS_ADDR` → noop. Тесты через miniredis.
- [x] `usage_log` запись при завершении (токены из SSE-события `usage`). _cost_usd пока NULL — расчёт по прайсингу позже._

_Реальный Claude — через worker (Этап 2.2); локально/в тестах `LLM_CLIENT_KIND=mock`. OpenAPI обновлён (4 эндпоинта). Тесты: юнит (курсор/история/uidHash/валидация) + integration хендлеров (testcontainers + MockClient: happy/cancel/429/pagination/delete/415) под `make test-integration`._

### 2.4 Mobile: чат-UI ✅
- [x] Экран чата: список сообщений (`ListView reverse:true`), инпут, кнопка «Отправить»/«Стоп» (`chat_screen.dart`).
- [x] Виджет `MessageBubble` (user — `SelectableText`, assistant — маркдаун). Рендер через `flutter_markdown_plus` (оригинальный `flutter_markdown` discontinued — поддерживаемый форк с тем же `MarkdownBody`).
- [x] Загрузка истории при открытии (cache-first из `drift` → рефреш с сервера; спиннер при пустом кэше).
- [x] Локальный кэш в `drift` (`chat_database.dart`: таблицы Messages/MessageBlocks, offline-показ; на logout кэш очищается — ownership).

### 2.5 Mobile: SSE-клиент ✅
- [x] Парсер SSE поверх `dio` `ResponseType.stream` (`ApiClient.openEventStream` + чистый `parseSse`; 401→refresh→retry inline, разбор error-конверта для не-200).
- [x] `ChatController` (riverpod `AsyncNotifier`): добавляет user-message локально → стримит дельты в pending assistant → завершает по `done`; финал complete/cancelled/failed, reconcile с сервером после хода.
- [x] Отмена кнопкой «Стоп» → `CancelToken.cancel()` (закрытие соединения; бэкенд сам финализирует `cancelled` + частичный текст). DELETE оставлен для явного удаления сообщения (long-press).

### 2.6 Тесты
- [x] Backend: интеграционные (мок Claude через `llm.MockClient`), включая отмену — сделано в §2.3 (`chat_endpoints_test.go`, testcontainers).
- [x] Mobile: тесты пузырей (`message_bubble_test.dart`) + «отправить → стрим → получить» на widget-уровне (`chat_screen_test.dart`); юнит `sse_parser`/`chat_models`/`openEventStream`/`ChatController`. _Device integration_test отложен — Android SDK не установлен (как on-device E2E в §1.4–1.6)._

### DoD Этапа 2
- Пользователь отправляет текст, видит стримящийся ответ.
- История сохраняется и подгружается при перезапуске.
- Отмена работает корректно (нет пустых сообщений).
- Prod-окружение поднято и использует реальный Claude через `llm-worker` на HostKey Frankfurt.
- Worker доступен только с IP бэкенда в Yandex Cloud (HostKey firewall / iptables IP-allowlist + mTLS).
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
- [ ] Eval-скрипт: прогон 20 кейсов через prod-worker (живой Anthropic), ручная оценка качества и стоимости.

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
- [ ] HostKey firewall / iptables на worker-VM: разрешён только IP Yandex Cloud VM на 443.
- [ ] Yandex Cloud Security Group: разрешён только трафик от мобильных клиентов на 443 (Caddy).
- [ ] Очистка тестовых данных из prod-БД (тестовые users, conversations, uploads).

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
| HostKey аккаунт (рубли) + Anthropic API-ключ (зарубежная карта для самой Anthropic) | Этап 0–2 | Заказчик          | открыто  |
| Yandex Cloud организация и биллинг                                    | Этап 0     | Заказчик            | открыто  |
| Yandex 360 для домена (SMTP)                                          | Этап 1     | Заказчик / DevOps   | открыто  |
| Yandex SpeechKit ключ                                                 | Этап 4     | DevOps              | открыто  |
| Apple Developer Account                                               | Этап 7     | Заказчик            | отложено (Q7) |
| Google Play Console                                                   | Этап 7     | Заказчик            | отложено (Q7) |
| Дизайн (иконка, ассеты, экранчики) — делаем сами                      | Этап 1+    | Команда             | закрыто (D14) |
| Каталог удобрений (CSV с полями из ARCH §6.1)                         | Этап 5     | Заказчик / контент  | отложено (Q2) |
| Privacy Policy / ToS на двух языках                                   | Этап 7     | Юрист               | отложено (Q6) |
| Домен и DNS                                                           | Этап 0     | DevOps / заказчик   | открыто  |
