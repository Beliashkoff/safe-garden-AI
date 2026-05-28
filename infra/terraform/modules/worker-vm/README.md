# Module: worker-vm

Провайдер-агностичная VM для `llm-worker`. Параметризована `var.provider_kind`:

| `provider_kind` | Используется в prod | Создаётся Terraform'ом |
| --- | --- | --- |
| `hostkey` | да, default (см. ARCH §11.7) | нет — VM создаётся вручную в личном кабинете HostKey, IP вписывается в `hostkey_manual_ip` |
| `hetzner` | резерв для DR | да — `hcloud_server` + `hcloud_firewall` |
| `ovh` | резерв для DR | да — `ovh_cloud_project_instance` |

## HostKey (ручной провижининг)

HostKey не имеет первоклассного Terraform-провайдера. Шаги:

1. В личном кабинете `hostkey.ru` заказать VM `vm.v2-nano` (2 vCPU / 4 GB / 60 GB
   NVMe) в локации **Frankfurt 1** (резерв — Amsterdam EuNetworks). Образ —
   Ubuntu 24.04.
2. Получить публичный IP, прописать его в `terraform.tfvars` как
   `hostkey_manual_ip = "X.Y.Z.W"`. Terraform положит значение в state через
   `null_resource.hostkey_manual` — `terraform plan` будет показывать дрейф,
   если кто-то поменяет IP.
3. В личном кабинете HostKey включить firewall: разрешить 443/tcp и 22/tcp
   только с IP бэкенд-VM в Yandex Cloud (`allowed_source_ips`).
4. SSH-подключение, выполнить шаги cloud-init вручную (см. `cloud-init.yaml.tftpl`)
   — установка Docker, UFW, монтирование LUKS-тома для секретов
   (ARCH §8.6).
5. LUKS-том + секреты: `sudo bash infra/scripts/setup-luks.sh init` создаёт
   зашифрованный том на `/etc/llmworker`. Туда кладутся `.env`
   (`ANTHROPIC_API_KEY`, `UID_HASH_PEPPER`, `WORKER_MAX_TOKENS`,
   `BACKEND_CALLBACK_URL`, `SENTRY_DSN`) и `certs/{ca.pem,worker.crt,worker.key}`
   из `infra/mtls/gen-certs.sh`. После каждого ребута том переоткрывается вручную
   (`setup-luks.sh open`) — snapshot диска без passphrase ключи не отдаёт (§11.7).
   Полный пошаговый деплой — в `envs/prod/README.md` §«runbook первого apply».

## DR-переезд на Hetzner/OVH

Triggers миграции — в ARCH §11.7. Шаги переезда:

1. Поменять `worker_provider = "hetzner"` в `envs/prod/terraform.tfvars`.
2. `terraform apply` — поднимется новая VM в Falkenstein/DE1.
3. На новой VM развернуть `infra/docker/compose/prod-llmworker.yml`, переложить
   секреты с LUKS-тома (Anthropic API key, mTLS cert/key) — через ручной SSH.
4. Поменять A-запись `worker.agronomai.site` на новый IP в Yandex Cloud DNS.
5. После прогрева — `terraform destroy -target=module.worker_vm` старой VM
   (или ручное удаление в HostKey).
