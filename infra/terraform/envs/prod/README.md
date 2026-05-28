# envs/prod — Production окружение Safe Garden AI

Единственное окружение в v1 (SPEC §8 D15). Состав:

| Ресурс | Модуль | Расположение |
| --- | --- | --- |
| API VM | `modules/yandex-vm` | Yandex Cloud, `ru-central1-a` |
| Managed PostgreSQL | `modules/yandex-postgres` | Yandex Cloud |
| Managed Redis | `modules/yandex-redis` | Yandex Cloud |
| Object Storage (media) | `modules/yandex-s3` | Yandex Cloud |
| LLM-worker VM | `modules/worker-vm` | HostKey Frankfurt (default), DR — Hetzner/OVH |
| VPC + Security Groups | inline в `main.tf` | Yandex Cloud |

## Этап 0.7: validate-only

```bash
terraform init -backend=false
terraform validate
```

Креды не нужны. apply не делается — реальная инфра поднимается в Этапе 2.2,
когда заказчик подключит Yandex Cloud / HostKey.

## Этап 2.2: runbook первого apply (оператор)

Выполняется заказчиком/оператором после Этапа 0.6 (нужны реальные аккаунты:
Yandex Cloud + billing, HostKey, Anthropic prod-ключ, домен `agronomai.site`).
Код приложения worker'а уже готов (`internal/llmworker` + `anthropic-sdk-go`);
ниже — поднятие инфраструктуры и подключение Claude.

### 0. Предусловия
- Yandex Cloud: организация + folder + billing; `yc_token`/`yc_cloud_id`/
  `yc_folder_id` (см. `backend/README.md` §«Регистрация внешних аккаунтов»).
- HostKey-аккаунт с балансом в ₽ (юрлицо ООО «АЙТИБ», ARCH §11.7).
- Anthropic prod-ключ на иностранное юрлицо/карту (ARCH §11.2).
- Домен `agronomai.site` с NS на Yandex Cloud DNS.

### 1. Bootstrap remote state
```bash
# создать приватный бакет под state (через yc CLI или консоль)
yc storage bucket create --name safe-garden-tfstate
# раскомментировать backend "s3" в versions.tf, затем:
export AWS_ACCESS_KEY_ID=<static-key-id> AWS_SECRET_ACCESS_KEY=<static-key-secret>
terraform init -migrate-state
```

### 2. apply Yandex-ресурсов
```bash
cp terraform.tfvars.example terraform.tfvars   # заполнить yc_*, ssh_public_key, worker_provider="hostkey"
terraform plan && terraform apply
```
Создаёт: API VM, Managed PostgreSQL, Managed Redis, Object Storage (private),
Lockbox, VPC + SG (БД доступна только из api-SG), service account. Снять outputs:
`api_public_ip`, `postgres_fqdn`, `redis`, `media_bucket`.

### 3. Worker-VM (HostKey Frankfurt)
- Заказать `vm.v2-nano` (Ubuntu 24.04) в панели HostKey → получить публичный IP.
- В `terraform.tfvars`: `worker_provider="hostkey"`, `hostkey_manual_ip="<ip>"` →
  `terraform apply` (cloud-init поставит docker/ufw/пользователя).
  DR: тот же модуль с `worker_provider="hetzner"|"ovh"` (ARCH §11.7).

### 4. mTLS-сертификаты
```bash
WORKER_DOMAIN=worker.agronomai.site bash infra/mtls/gen-certs.sh   # → infra/mtls/out/
```
Раздать (ARCH §8.6): `ca.pem`+`api-client.{crt,key}` → **Yandex Lockbox** (бэкенд);
`ca.pem`+`worker.{crt,key}` → LUKS-том worker-VM. `ca.key` хранить offline.

### 5. Деплой worker'а
```bash
# на worker-VM:
sudo bash infra/scripts/setup-luks.sh init          # LUKS-том → /etc/llmworker
# положить /etc/llmworker/.env (ANTHROPIC_API_KEY, UID_HASH_PEPPER, SENTRY_DSN,
#   WORKER_MAX_TOKENS, BACKEND_CALLBACK_URL) и /etc/llmworker/certs/{ca,worker}.*
docker compose -f infra/docker/compose/prod-llmworker.yml up -d   # Caddy mTLS + llmworker:8081
```
HostKey firewall: открыть 443 только с `api_public_ip` (панель HostKey или iptables, §11.2).

### 6. Деплой бэкенда + DNS
```bash
# на API VM: env из Lockbox, выставить
#   LLM_CLIENT_KIND=worker
#   LLM_WORKER_BASE_URL=https://worker.agronomai.site
#   LLM_WORKER_MTLS_ENABLED=true + пути к ca.pem/api-client.{crt,key}
docker compose -f infra/docker/compose/prod-yandex.yml up -d       # Caddy LE + api
```
DNS A-записи: `api.agronomai.site`→`api_public_ip`, `worker.agronomai.site`→worker IP.

### 7. Smoke-test
- С API VM: `curl --cert api-client.crt --key api-client.key --cacert ca.pem
  https://worker.agronomai.site/healthz` → 200.
- Реальный стрим-запрос → Claude отвечает; в логах worker'а — `usage`
  (tokens_in/out), без текста/PII (ARCH §11.2). После публичного теста — алерты
  на 5xx worker'а (триггер плана миграции §11.7) и бюджет Anthropic.

## DR-переезд worker'а

См. `infra/terraform/modules/worker-vm/README.md` — поменять `worker_provider`,
заново применить, переключить DNS.
