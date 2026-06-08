# Docker Compose — prod

Описания для prod-окружения. Запускаются на двух разных VM:

| Файл | На какой VM | Содержит |
| --- | --- | --- |
| `prod-yandex.yml` + `prod-yandex.Caddyfile` | Yandex Compute Cloud (Москва/Питер) | `api` + `caddy` |
| `prod-llmworker.yml` + `prod-llmworker.Caddyfile` | Worker-VM (HostKey Frankfurt по умолчанию, в случае DR — Hetzner/OVH) | `llmworker` + `caddy` с mTLS client_auth |

## Раскладка на VM

На обеих VM ожидается одна и та же структура:

```
/etc/safegarden/
├── compose/
│   ├── docker-compose.yml      # ← prod-yandex.yml или prod-llmworker.yml
│   └── Caddyfile               # ← соответствующий .Caddyfile
├── .env                        # секреты (chmod 600)
└── certs/                      # только на worker-VM, на LUKS-томе
    ├── ca.pem
    ├── worker.crt
    └── worker.key
```

Деплой выполняется в Этапе 2.2 (после `terraform apply` и подключения
DNS). Ручная команда на VM (первый bootstrap и откат):

```bash
cd /etc/safegarden/compose
docker compose pull
docker compose up -d
```

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
   `ssh-keyscan -t ed25519 <host> | ssh-keygen -lf -` → берём часть `SHA256:...`.
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
