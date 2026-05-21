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
DNS). Команда на VM:

```bash
cd /etc/safegarden/compose
docker compose pull
docker compose up -d
```

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
