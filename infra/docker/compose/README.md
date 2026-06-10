# Docker Compose — prod

Описания для prod-окружения. Запускаются на двух разных VM:

| Файл | На какой VM | Содержит |
| --- | --- | --- |
| `prod-yandex.yml` + `prod-yandex.Caddyfile` | Yandex Compute Cloud (Москва/Питер) | `api` + `caddy` |
| `prod-llmworker.yml` + `prod-llmworker.Caddyfile` | Worker-VM (VPS вне РФ, сейчас Финляндия; DR — Hetzner) | `llmworker` + `caddy` с mTLS client_auth |

## Раскладка на VM

`compose/` (docker-compose.yml + Caddyfile + интерполяционный `.env` с
`DOMAIN`/`IMAGE_TAG`) лежит на корневом диске обеих VM в `/etc/safegarden/compose`
(его пишет CD). Расположение секретов различается:

**API-VM (Yandex)** — секреты и серты на корневом диске:
```
/etc/safegarden/
├── compose/{docker-compose.yml, Caddyfile, .env}
├── .env            # секреты api (chmod 600) — env_file контейнера
├── certs/          # ca.pem, api-client.{crt,key}, internal.{crt,key}
└── jwt-keys/
```

**Worker-VM (VPS вне РФ)** — секреты и серты на LUKS-томе `/etc/llmworker`
(ARCH §8.6); `compose/` — на корневом диске:
```
/etc/safegarden/compose/{docker-compose.yml, Caddyfile, .env}   # корневой диск
/etc/llmworker/                                                 # LUKS-том
├── .env            # ANTHROPIC_API_KEY, UID_HASH_PEPPER, BACKEND_CALLBACK_* (chmod 600)
└── certs/          # ca.pem, worker.{crt,key} (Caddy) + worker-client.{crt,key} (callback)
```

Деплой/откат — `cd /etc/safegarden/compose && docker compose pull && docker compose up -d`.

### LUKS на worker-VM (ARCH §8.6) — обязательная разблокировка после ребута

Секреты воркера лежат на LUKS-томе (`/var/lib/llmworker.luks` → `/etc/llmworker`),
который **не открывается автоматически** при загрузке (snapshot диска без passphrase
ключи не отдаёт, §11.7). Первичная настройка — `sudo bash setup-luks.sh init`.
**После каждого ребута VPS:**
```bash
sudo bash setup-luks.sh open                       # ввести passphrase
cd /etc/safegarden/compose && docker compose up -d
```
До разблокировки Caddy не загрузит серты (443 down) и worker стартует в echo-режиме —
бэкенд получит 503. Это ожидаемое поведение.

### Хост-настройка VM (вне compose, для воспроизводимости)

Применяется на VM руками (не в репозитории):
- **Ротация docker-логов:** `/etc/docker/daemon.json` =
  `{"log-driver":"json-file","log-opts":{"max-size":"10m","max-file":"3"}}` →
  `systemctl restart docker` → `docker compose up -d --force-recreate` (обе VM).
- **SSH-харденинг:** `apt install fail2ban` (jail sshd по умолчанию) + drop-in
  `/etc/ssh/sshd_config.d/00-safegarden-hardening.conf` (`PermitRootLogin no`,
  `PasswordAuthentication no`, `PubkeyAuthentication yes`) → `sshd -t && systemctl restart ssh`.
  Порт 22 в Yandex SG оставлен открытым — CD-деплой ходит по SSH с GitHub-раннеров.

### Мониторинг (на API-VM)

`monitoring.yml` поднимается рядом с `prod-yandex.yml`: цепляется к внешней сети
`safegarden-api_internal`, Prometheus скрейпит `api:9100`. Деплой:
`docker compose -f monitoring.yml --env-file /etc/safegarden/monitoring.env up -d`
(переменные — см. `monitoring.env.example`). Любая последующая `docker compose -f
monitoring.yml …` команда тоже требует `--env-file …` (иначе ошибка интерполяции
`GRAFANA_ADMIN_PASSWORD`). Grafana — только на loopback: доступ через
`ssh -L 3000:localhost:3000 safegarden@<api-vm>` → http://localhost:3000.
Алерты — email (Alertmanager → Yandex SMTP, секрет
`/etc/safegarden/monitoring/secrets/smtp_password`, тот же app-пароль, что у api).
`postgres-exporter` опционален (профиль `pg`); метрики БД иначе — в нативном
мониторинге Yandex Cloud.

## Авто-деплой (CD)

После настройки (см. ниже) ручной шаг не нужен: workflow
`.github/workflows/build-images.yml` после сборки образов сам заходит по SSH
на обе VM и катит на них **точный** собранный тег `sha-<short>` (не плавающий
`:latest`). Скрипт на VM пишет `DOMAIN` и `IMAGE_TAG` в
`/etc/safegarden/compose/.env` (это файл интерполяции compose, не путать с
секретным `/etc/safegarden/.env` = `env_file`), затем `docker compose pull &&
up -d`. `up -d` пересоздаёт контейнер только если дайджест образа изменился.

Job `deploy` привязан к GitHub Environment `production` — при желании туда можно
добавить required reviewers и получить ручной аппрув прода, не трогая workflow.

### Конфиг в GitHub (Settings → Secrets and variables → Actions)

Secrets (на каждую VM свои — компрометация одной не даёт доступа к другой):

| Secret | Что |
| --- | --- |
| `API_VM_HOST` / `WORKER_VM_HOST` | IP или хостнейм VM |
| `API_VM_SSH_KEY` / `WORKER_VM_SSH_KEY` | приватный ключ деплой-пары (PEM целиком) |
| `API_VM_FINGERPRINT` / `WORKER_VM_FINGERPRINT` | отпечаток host-ключа VM для пиннинга |

Variables:

| Variable | Что | Дефолт |
| --- | --- | --- |
| `API_VM_DOMAIN` / `WORKER_VM_DOMAIN` | домен для Caddy (`api.agronomai.site` / `worker.agronomai.site`) | — (обязателен) |
| `API_VM_SSH_USER` / `WORKER_VM_SSH_USER` | пользователь SSH | `safegarden` |
| `API_VM_SSH_PORT` / `WORKER_VM_SSH_PORT` | порт SSH | `22` |

### Разовая настройка VM

Деплой идёт под пользователем `safegarden` — тем же, что уже состоит в группе
`docker` и имеет ghcr-логин в `~/.docker/config.json` (его использует
cleanup-юнит). Нужно:

1. Сгенерировать выделенную пару ключей (ed25519) для CD:
   `ssh-keygen -t ed25519 -f deploy_key -N '' -C cd@safegarden`.
2. Публичный ключ — в `/home/safegarden/.ssh/authorized_keys`, желательно с
   ограничениями: `no-port-forwarding,no-x11-forwarding,no-agent-forwarding`.
   Приватный — в secret `*_VM_SSH_KEY`.
3. Снять отпечаток host-ключа VM в secret `*_VM_FINGERPRINT`:
   `ssh-keyscan -t ecdsa <host> | ssh-keygen -lf -` → берём часть `SHA256:...`.
   ВАЖНО: именно `ecdsa`, не `ed25519`. appleboy/ssh-action (drone-ssh, Go
   `x/crypto/ssh`) при наличии у хоста ecdsa-ключа согласует `ecdsa-sha2-nistp256`
   и сверяет его отпечаток; ed25519-значение даст `host key fingerprint mismatch`.
4. Убедиться, что у `safegarden` есть login-shell и доступ к каталогу
   `/etc/safegarden/compose` (запись `.env` + чтение compose-файла).

### Откат

Каждый деплой фиксирует `IMAGE_TAG` в `.env`, поэтому откат — это пин предыдущего
SHA на VM:

```bash
cd /etc/safegarden/compose
sed -i 's/^IMAGE_TAG=.*/IMAGE_TAG=sha-<prev>/' .env
docker compose pull && docker compose up -d
```

Старые `sha-`-образы остаются в локальном кэше (их не трогает `image prune -f`,
он чистит только dangling-слои), так что цель отката обычно уже на диске.

### Граница

CD катит **образы**. Изменения самих compose-файлов / Caddyfile в `infra/` сюда
не входят — их по-прежнему надо разложить на VM и применить `up -d` вручную (либо
отдельным config-sync шагом, если понадобится).

## Локальная валидация (без apply)

В CI и локально проверяется только синтаксис:

```bash
DOMAIN=api.agronomai.site IMAGE_TAG=latest \
  docker compose -f infra/docker/compose/prod-yandex.yml config -q
DOMAIN=worker.agronomai.site IMAGE_TAG=latest \
  docker compose -f infra/docker/compose/prod-llmworker.yml config -q
```

## mTLS

mTLS терминируется Caddy перед `llmworker`. Сам worker слушает обычный HTTP
на `:8081` внутри docker-сети — это упрощает код (см. план Этапа 0.7,
решение 2). Сертификаты (`ca.pem`, `worker.crt`, `worker.key`) живут на
LUKS-зашифрованном томе worker-VM (ARCH §8.6) и подмонтированы в Caddy
read-only по пути `/etc/caddy/certs/`.

В Этапе 0.7 шаблоны Caddyfile и compose-файлы лежат в репо, реальные
сертификаты — нет. Генерация CA + выпуск сертификатов — Этап 2.2.
