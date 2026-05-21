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

## Этап 2.2: первый apply

1. Заказчик создал Yandex Cloud организацию + folder, выдал `yc_token`,
   `yc_cloud_id`, `yc_folder_id` (см. `backend/README.md` §«Регистрация
   внешних аккаунтов»).
2. Скопировать `terraform.tfvars.example` → `terraform.tfvars`, заполнить.
3. Добавить в `versions.tf` блок `backend "s3" { ... }` для remote state
   в Yandex Object Storage (отдельный бакет `safe-garden-tfstate`).
4. `terraform init`, `terraform plan`, `terraform apply`.
5. После поднятия VM:
   - А-запись `api.agronomai.site` → `api_public_ip` output.
   - А-запись `worker.agronomai.site` → `worker_ip` output (если используется).
   - Сертификаты mTLS генерируются отдельно (CA + клиентский для бэкенда +
     серверный для Caddy на worker-VM), кладутся в Yandex Lockbox / LUKS-том.

## DR-переезд worker'а

См. `infra/terraform/modules/worker-vm/README.md` — поменять `worker_provider`,
заново применить, переключить DNS.
