# Terraform — Safe Garden AI

Infrastructure-as-code для prod-окружения. Описано в Этапе 0.7
([`../../ROADMAP.md`](../../ROADMAP.md)), но **`terraform apply` будет в
Этапе 2.2** — после того, как заказчик подключит Yandex Cloud / HostKey /
домен и появятся реальные креды.

В 0.7 цель — чтобы код в этой папке проходил `terraform fmt -check` и
`terraform validate` в CI. Без сети, без apply, без секретов в репозитории.

## Структура

```
infra/terraform/
├── envs/
│   └── prod/                # инстанс окружения, единственное окружение в v1
│       ├── main.tf
│       ├── variables.tf
│       ├── outputs.tf
│       ├── versions.tf
│       └── terraform.tfvars.example
└── modules/
    ├── worker-vm/           # провайдер-агностичная VM для llm-worker'а
    ├── yandex-vm/           # VM для основного бэкенда в Yandex Cloud
    ├── yandex-postgres/     # Managed PostgreSQL
    ├── yandex-redis/        # Managed Redis (v2)
    └── yandex-s3/           # Object Storage bucket
```

`worker-vm` параметризован переменной `provider` со значениями `hostkey`,
`hetzner`, `ovh` — выполнение принятого риска HostKey
([`../../ARCHITECTURE.md`](../../ARCHITECTURE.md) §11.7): DR-переезд на
иностранного провайдера должен быть однокнопочным. На HostKey у Terraform
нет первоклассного провайдера — этот случай оформлен как `null_resource`
с ручной инструкцией в `modules/worker-vm/README.md`.

## Локальная проверка (без apply)

```bash
cd infra/terraform
terraform fmt -check -recursive

cd envs/prod
terraform init -backend=false
terraform validate
```

Оба шага не требуют облачных кредов и выполняются на любом ноутбуке.

## State

В 0.7 state — локальный (`*.tfstate` в gitignore). Remote backend в Yandex
Object Storage добавим в Этапе 2.2 вместе с первым `apply` — для этого
заказчик подключит Yandex Cloud аккаунт.

## Что в 0.7 НЕ делается

- Не запускается `terraform plan` / `terraform apply`.
- Не создаются реальные VM, БД, бакеты.
- Не описывается DNS-модуль — zone `agronomai.site` создаётся вручную
  в Yandex Cloud DNS по runbook'у `backend/README.md` (Этап 0.6).
  A-записи на VM добавим в 2.2/7.2.
