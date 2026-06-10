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
- **LLM:** Claude Opus 4.x через **отдельный `llm-worker`** на VPS в Финляндии (физически вне РФ; Anthropic блокирует РФ-AS, поэтому прямой вызов и AWS Bedrock не работают). Принятые риски размещения и план миграции провайдера — в `ARCHITECTURE.md` §11.7.
- **Email:** Yandex 360 SMTP (OTP-коды)
- **Транскрипция:** Yandex SpeechKit v3 (с конвертацией m4a→OggOpus через `ffmpeg`)
- **Облако:** Yandex Cloud (152-ФЗ, РФ-юрисдикция, PII не покидает РФ)
- **Деплой:** Docker Compose на VM (Yandex Compute для api + VPS в Финляндии для llm-worker). Без Kubernetes в v1.
- **Окружения:** dev (локально через docker-compose) + prod (Yandex Cloud + VPS в Финляндии). Stage-окружения нет — до релиза prod используется и для ручного тестирования.
- **Сторы:** App Store, Google Play

## Структура репозитория

Монорепо. Полная схема — в `ARCHITECTURE.md` §3.

```
safe-garden-AI/
├── backend/              # Go: HTTP API (РФ) + LLM-worker (VPS Финляндия)
├── mobile/               # Flutter: iOS / Android
├── infra/                # Terraform + Docker Compose для prod
├── .github/workflows/    # CI: backend, mobile, release
├── SPEC.md / ARCHITECTURE.md / ROADMAP.md / CLAUDE.md
```

## Требования к окружению

Для разработки нужно:

- **Go** 1.25+ (см. `backend/go.mod`)
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

CI состоит из трёх workflow:

- `.github/workflows/backend-ci.yml` — `lint` (golangci-lint + gofmt) и `test` (`go test -race`) на каждом PR/push в main.
- `.github/workflows/mobile-ci.yml` — `analyze` (`dart format` + `flutter analyze`), `test` (`flutter test`), `build-apk-debug` (на PR), `build-apk-release` (на push в main).
- `.github/workflows/infra-ci.yml` — `terraform` (`fmt -check` + `validate`) и `compose` (`docker compose config`). Запускается только при изменениях в `infra/**`.

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
- 0.5 Документация ✅
- 0.7 Скелет llm-worker ✅ — `cmd/llmworker` (echo-SSE), `internal/llm` (Client/WorkerClient/MockClient), Terraform-каркас, prod Docker Compose. `terraform apply` отложен до 2.2.
- 1.1 Backend: модели и хранилище ✅ — миграции `users`/`refresh_tokens`/`email_codes`/`audit_log`, sqlc-запросы, `internal/auth/jwt.go` (RS256 + `kid`-ротация), `internal/auth/oidc.go` (Apple + Google), refresh/OTP-примитивы. Integration-тесты через testcontainers.
- 1.2 Backend: эндпоинты ✅ — 8 эндпоинтов auth/account (`internal/transport/http`), `RequireAuth` + цепочка middleware, единый формат ошибок (`httperr`), usecase `internal/usecase/auth` (sign-in Apple/Google/email с auto-link по email, ротация refresh + reuse-detection, soft-delete + аудит), DB-baseline rate limit. OpenAPI 3.0 + Swagger UI на `/v1/docs`. Integration-тесты хендлеров через testcontainers + fake OIDC.
- 1.3 Backend: email-провайдер ✅ — `internal/mailer` (интерфейс + SMTP на stdlib `net/smtp`: implicit TLS для Yandex 360, plaintext для MailHog, log-fallback), шаблоны OTP RU/EN (HTML+text). _SPF/DKIM/DMARC для `noreply@agronomai.site` настраиваются в DNS — часть Этапа 0.6._
- 1.4–1.6 Mobile: авторизация ✅ — экраны login/email/verify, единый `ApiClient` (dio + интерсепторы auth/refresh-on-401), `SecureTokenStore`, `AuthController` (riverpod), auto-вход + redirect-гард роутера, email-OTP/Apple/Google, удаление аккаунта. Тесты: `flutter analyze` + 33 `flutter test` зелёные. _On-device E2E и реальные OAuth отложены до Android SDK / Google Cloud OAuth (0.6) / macOS+Apple Developer (7)._
- 2.1 Backend: модели чата ✅ — миграции `0006…0011` (вся ARCH §6.1: conversations/messages/message_blocks/usage_log/uploads/fertilizers, FK-каскады, CHECK-и, GIN), sqlc-запросы (keyset-история, статусы, get-or-create чата, recommend, usage-sum). Integration-тесты через testcontainers. _Schema-only; HTTP/SSE чата — 2.3, llm-worker — 2.2._
- 2.3 Backend: SSE-чат ✅ — `POST /v1/messages` (стрим ответа через SSE: сохранение user-msg, история ~20, `llm.Client.Send`, финал complete/cancelled/failed + usage_log), `GET /v1/conversation` (+`/conversation/messages` keyset-пагинация), `DELETE /v1/messages/{id}`. Per-user rate-limit 20 RPS на Redis (`redis_rate`, fail-open). `internal/usecase/chat` + `handler/chat.go`. Тесты: юнит + integration (testcontainers + MockClient). _Реальный Claude через worker; локально `LLM_CLIENT_KIND=mock`._
- 2.2 LLM-worker + Claude ✅ (код) — `anthropic-sdk-go` на worker'е за провайдер-абстракцией (`internal/llmworker`: anthropic/echo/provider, стриминг→SSE, prompt caching, usage), system prompt (`internal/llm/prompts`), модель `claude-opus-4-7`. РФ-клиент (mTLS+SSE) из 0.7. mTLS cert-gen + LUKS-скрипты + terraform remote-state шаблон. Unit-тесты зелёные; SDK-импорт только в worker (инвариант №5). _`terraform apply`/деплой/реальный Claude — runbook оператора после 0.6: `infra/terraform/envs/prod/README.md`._
- 2.4–2.5 Mobile: чат ✅ — экран чата (`features/chat`: `chat_screen.dart`, `MessageBubble` user/assistant с маркдауном через `flutter_markdown_plus`), `ChatController` (riverpod `AsyncNotifier`): оптимистичный user-msg → стрим дельт в pending assistant → финал complete/cancelled/failed + reconcile с сервером. SSE-клиент поверх `dio` `ResponseType.stream` (`ApiClient.openEventStream` + чистый `parseSse`; 401→refresh→retry, разбор error-конверта). Offline-кэш истории в `drift` (cache-first, очистка на logout). Отмена кнопкой «Стоп» (`CancelToken`). Тесты: `flutter analyze` + 56 `flutter test` зелёные (парсер/модели/openEventStream/контроллер/пузыри/«отправить→стрим→получить»). _Device E2E отложен — Android SDK не установлен._
- 3.1 Backend: фото в чате ✅ — presigned PUT (`POST /v1/uploads/presign`, `internal/objstore` на `aws-sdk-go-v2`, `internal/usecase/upload`; ключ `u/{user_id}/img/{uuid}`, TTL 5 мин, whitelist+≤10 МБ, `used`-флаг). `POST /v1/messages` принимает `image_ref { storage_key }` (ownership через `uploads.user_id` + префикс ключа, `MarkUploadUsed`); бэкенд грузит фото из Object Storage, конвертирует HEIC→JPEG (`internal/imageconv`, `gen2brain/heic` — libheif через WASM, без CGO) и шлёт в Claude как base64-`image` через worker. Фото из истории ре-отправляются с лимитом 4 (старше → маркер «[фото]»). MinIO в dev, Yandex Object Storage в prod. Тесты: юнит + integration (фейк-objstore). OpenAPI обновлён (presign + image_ref). _Реальный Claude-vision — после 0.6._
- 3.2 Backend: каскадное удаление медиа ✅ — `DELETE /v1/account` стирает контент в одной транзакции (conversations→messages→blocks + uploads-строки, анонимизация users, usage_log сохраняется для биллинга). Объекты Object Storage удаляются асинхронно бинарём `cmd/cleanup` по префиксу `u/{user_id}/` (durable через `users.media_purged_at`, миграция 0012; `objstore.DeletePrefix`/`DeleteKeys`). Ежедневный GC `uploads` старше 7д с `used=false` (`internal/usecase/cleanup`). Запуск — cron / `docker compose run --rm cleanup` (рек. ежечасно). Тесты: юнит + integration. _Планировщик — этап деплоя._
- 3.3 Mobile: фото в чате ✅ — скрепка с bottom-sheet «Камера»/«Галерея», `image_picker` + `flutter_image_compress` (1920×1080, q=85, JPEG) и `permission_handler` за портами (`media_ports.dart`); пайплайн presign → прямой PUT в Object Storage с прогрессом → POST /messages в `MessageComposer` (`UploadApi`), до 4 фото, «только фото» допустимо. Превью в bubble через `MediaCache` (`storage_key`→локальный файл; история — ленивая докачка). Доп. к скоупу: бэкенд-endpoint `POST /v1/uploads/view` (presigned GET, ownership по префиксу) — чтобы показывать фото из истории после переустановки/на другом устройстве. Тесты: `flutter analyze` + 63 `flutter test` зелёные; backend юнит+integration (`view` foreign→403). _On-device E2E отложен — Android SDK не установлен._
- 4.1 Backend: провайдер транскрипции ✅ (код) — `internal/audio` за абстракцией `Transcriber`: `speechkit.go` (Yandex SpeechKit v3, **streaming gRPC** `Recognizer.RecognizeStreaming`, `authorization: Api-Key`, OggOpus 16kHz mono, нормализация, лимит 60с), `converter.go` (ffmpeg/ffprobe: m4a/aac/mp3→OggOpus + валидация длительности), `mock.go`, `gigachat.go` (заглушка-фолбэк SaluteSpeech за тем же интерфейсом), `config.go`/`factory.go` (`STT_PROVIDER_KIND`, ключи из env). ffmpeg добавлен в API-образ (`debian:12-slim`). Тесты: gRPC-клиент через bufconn + реальная ffmpeg-конвертация. _mobile — 4.3._
- 4.2 Backend: голос в чате ✅ — presign расширен на audio (m4a/aac/mp4/mpeg ≤25 МБ, ключ `u/{user_id}/audio/…`, тип по content-type); `POST /v1/messages` принимает `audio_ref`: до старта стрима бэкенд грузит аудио (`objstore.GetLimited`), конвертирует (ffmpeg→OggOpus) и транскрибирует (SpeechKit), затем сохраняет блоки `audio` + `transcription` (`duration_ms` в metadata) и отдаёт текст Claude с пометкой `[голосовое сообщение]:`. Текст транскрипции уходит клиенту новым SSE-событием `transcription` (перед ответом ассистента) и в `GET /conversation`. Сбой = pre-stream JSON-ошибка (too-long→400, не распознано→400, провайдер недоступен→503 `service_unavailable`). `internal/usecase/chat` + `objstore.GetLimited`; OpenAPI + ARCH §4.3/§4.7 обновлены. Тесты: юнит (transcribeAudio/история/проекция) + integration (voice happy-path, foreign-audio→404, too-long→400). _mobile — 4.3._
- 4.3 Mobile: голосовые сообщения ✅ — hold-to-record на кнопке-микрофоне (пустое поле → микрофон, текст → отправка): зажать → запись с таймером и индикатором уровня (свайп влево — отмена, авто-стоп 60с), отпустить → превью с плеером и «Удалить»/«Отправить». Запись `record` (AAC/m4a, 16kHz mono) и воспроизведение `just_audio` за портами (`audio_ports.dart`), `VoiceRecorderController` (presign→PUT→`ChatController.sendMessage(audioStorageKey)`). В чате — `VoiceMessagePlayer` (плеер из media-кэша) + текст транскрипции; SSE-событие `transcription` прикрепляет текст к user-сообщению до ответа ассистента. Доп.: `MediaCache` хранит расширение по ключу (аудио кэшируется), оффлайн-кэш блоков переведён на JSON (storage_key/duration переживают перезапуск). Разрешение микрофона (Info.plist + RECORD_AUDIO) за `PermissionPort.ensureMicrophone`. Тесты: `flutter analyze` + 72 `flutter test` зелёные (контроллер записи, SSE `transcription`, модель, бабл-плеер). _On-device E2E (микрофон, реальная запись/воспроизведение) отложен — Android SDK не установлен._
- 5.1–5.5 Каталог удобрений + Tool Use ✅ (код) — таблица `fertilizers` + сидер `cmd/seed` из `infra/data/fertilizers.csv` (плейсхолдер с демо-записью). На worker'е — мультитёрн tool-use цикл: при `recommend_fertilizer` обратный RPC на РФ-бэкенд (`POST /internal/v1/tools/fertilizer`, отдельный mTLS-листенер, не под `RequireAuth`), результат → `tool_result` Claude, параллельно SSE-событие `fertilizer_card`. Бэкенд ретранслирует карточку и сохраняет блок `fertilizer_card` (`message_blocks.metadata.products`); история восстанавливает карточки. Mobile — `FertilizerProduct`/`ContentBlock.products`, парсинг события, виджет `FertilizerCardList` (вертикальный стек, фото/название/описание, «Подробнее» → `url_launcher`), аналитика тапов в `usage_log`. System prompt финализирован (обязательность tool call, тон), prompt caching на system+tools, eval-набор из 20 кейсов (`internal/llmworker/eval/cases.json`). Тесты: backend юнит (tool-loop с фейк-каталогом: эмит карточки / пустой каталог / ошибка callback; эндпоинт `/internal/v1/tools/fertilizer`; usecase каталога) + mobile (парсинг события, widget-тест карточки). _Каталог наполняется позже (данные заказчика, SPEC Q2); операторский прогон eval через живой Anthropic — после 0.6._
- 6.1–6.6 Полировка / безопасность / мониторинг 🚧 (в работе) — security-CI (`dependabot`, `govulncheck` блокирующий + `nancy` nightly, gosec через golangci) и регрессионные тесты `RequireAuth`/ownership/изоляции переписки + аудит PII-логов; Prometheus-метрики (`/metrics` на отдельном внутреннем порту: http по route-паттерну, `claude_*`/`message_total`/`upload_status_total`), стек `infra/docker/compose/monitoring` (Prometheus+Grafana+Alertmanager+exporters, дашборд + алерты порогов ARCH §9, провалидированы `promtool`/`amtool`), Sentry backend+mobile (`sendDefaultPii=false`); k6-скрипт `infra/loadtest/messages.js` + `make loadtest`; mobile — онбординг (3 экрана), согласие на обработку ПДн со ссылками на Privacy/ToS, плейсхолдер app-icon/splash + тулинг. Тесты зелёные: backend `go test ./...` (+integration), mobile 78 `flutter test`. _Pending (runtime/внешнее): деплой мониторинга + тест-алерт, прогон k6 и `EXPLAIN ANALYZE` на prod-данных, генерация платформенных ассетов иконки/splash и финальный дизайн, тексты Privacy/ToS + РКН + Google Play Data Safety, Sentry symbol upload, OWASP sign-off._
- 🚀 **Прод-деплой** ✅ (2026-06-10) — РФ-бэкенд (Yandex Cloud, `api.agronomai.site`) + `llm-worker` на VPS в Финляндии (`worker.agronomai.site`, Caddy mTLS, UFW 443←только API-VM); бэкенд переключён в `LLM_CLIENT_KIND=worker` — реальный Claude работает в проде. CD-автодеплой обеих VM активен (`build-images.yml`).
- В работе — **Этап 6** (полировка/безопасность/мониторинг) и подготовка к релизу (**Этап 7**). Внешние аккаунты (Anthropic, Yandex Cloud, Yandex 360, DNS `agronomai.site`) подключены. Дальнейшие этапы — в `ROADMAP.md`.

## Открытые блокеры по этапам

См. `ROADMAP.md` «Критические зависимости». Краткий обзор:

- **Этап 0.6:** HostKey аккаунт (рубли) + Anthropic API-ключ (зарубежная карта/юрлицо — Anthropic в рублях не принимает), Yandex Cloud организация, Yandex 360 для домена, перевод NS домена `agronomai.site` на Yandex Cloud DNS + MX/SPF/DKIM/DMARC. Пошагово — в [`backend/README.md`](./backend/README.md) §«Регистрация внешних аккаунтов».
- **Этап 4:** Yandex SpeechKit ключ.
- **Этап 5:** Каталог удобрений (CSV от заказчика).
- **Этап 7:** Apple Developer Account, Google Play Console, Privacy Policy / ToS на двух языках.
