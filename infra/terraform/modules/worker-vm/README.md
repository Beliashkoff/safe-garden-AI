# Module: worker-vm

Провайдер-агностичная VM для `llm-worker`. Параметризована `var.provider_kind`:

| `provider_kind` | Используется в prod | Создаётся Terraform'ом |
| --- | --- | --- |
| `manual` | да, default — ручной VPS вне РФ (сейчас Финляндия, `vds.chsl.one`), IP в `manual_ip` | нет — VM создаётся вручную в панели провайдера |
| `hostkey` | legacy-алиас ручного провижининга (тот же `null_resource.manual`) | нет |
| `hetzner` | резерв для DR | да — `hcloud_server` + `hcloud_firewall` |

## Ручной провижининг (VPS вне РФ)

У ручного VPS нет первоклассного Terraform-провайдера. Шаги:

1. Заказать VPS (≥2 vCPU / 2 GB RAM / Ubuntu 24.04) у провайдера вне РФ
   (сейчас — Финляндия, `vds.chsl.one`; DR-резерв — Hetzner, см. ниже).
2. Получить публичный IP, прописать его в `terraform.tfvars` как
   `worker_manual_ip = "X.Y.Z.W"`. Terraform положит значение в state через
   `null_resource.manual` — `terraform plan` покажет дрейф при изменении IP.
   Этот же IP открывает ingress 8090 в SG api (callback worker→backend, §11.3).
3. На самой VM настроить firewall (UFW): 443/tcp и 22/tcp только с IP
   бэкенд-VM в Yandex Cloud (`allowed_source_ips`); SSH на ключи, root/пароль off.
4. Выполнить шаги cloud-init вручную (см. `cloud-init.yaml.tftpl`) — Docker, UFW.
5. LUKS-том + секреты: `sudo bash infra/scripts/setup-luks.sh init` создаёт
   зашифрованный том на `/etc/llmworker`. Туда кладутся `.env`
   (`ANTHROPIC_API_KEY`, `UID_HASH_PEPPER`, `WORKER_MAX_TOKENS`,
   `BACKEND_CALLBACK_URL` + callback cert paths, `SENTRY_DSN`) и
   `certs/{ca.pem,worker.crt,worker.key,worker-client.crt,worker-client.key}`
   из `infra/mtls/gen-certs.sh`. После каждого ребута том переоткрывается вручную
   (`setup-luks.sh open` + `docker compose up -d`) — snapshot диска без passphrase
   ключи не отдаёт (§11.7). Полный пошаговый деплой — в `envs/prod/README.md`.

## DR-переезд на Hetzner

Triggers миграции — в ARCH §11.7. Шаги переезда:

1. Поменять `worker_provider = "hetzner"` в `envs/prod/terraform.tfvars`.
2. `terraform apply` — поднимется новая VM в Falkenstein/DE1.
3. На новой VM развернуть `infra/docker/compose/prod-llmworker.yml`, переложить
   секреты с LUKS-тома (Anthropic API key, mTLS cert/key) — через ручной SSH.
4. Поменять A-запись `worker.agronomai.site` на новый IP в Yandex Cloud DNS.
5. После прогрева — `terraform destroy -target=module.worker_vm` старой VM
   (или ручное удаление в панели текущего провайдера).
