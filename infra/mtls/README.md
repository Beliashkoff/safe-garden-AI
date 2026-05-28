# mTLS trust chain (backend ↔ llm-worker)

`gen-certs.sh` builds a private CA and two leaf certs for the mutual-TLS link
between the RU backend and the Frankfurt worker (ARCH §8.6, §11.2). The worker
listens plain HTTP inside its VM; **Caddy terminates mTLS** in front of it
(`infra/docker/compose/prod-llmworker.Caddyfile`, `client_auth require_and_verify`).

## Generate

```bash
WORKER_DOMAIN=worker.agronomai.site bash infra/mtls/gen-certs.sh
# → infra/mtls/out/{ca.key,ca.pem,worker.key,worker.crt,api-client.key,api-client.crt}
```

`out/` is gitignored — certs and keys must never be committed.

## Distribute (ARCH §8.6)

| Host | Files | Where |
| ---- | ----- | ----- |
| Backend VM (Yandex) | `ca.pem`, `api-client.crt`, `api-client.key` | Yandex **Lockbox** → env at boot; backend reads `LLM_WORKER_CA_PATH` / `..._CLIENT_CERT_PATH` / `..._CLIENT_KEY_PATH` |
| Worker VM (HostKey) | `ca.pem`, `worker.crt`, `worker.key` | **LUKS** volume `/etc/llmworker/certs`, mounted read-only into Caddy |

Keep `ca.key` **offline** (it is not needed on any server — only to sign new
leaves). Rotation: regenerate leaf certs quarterly (`DAYS_LEAF`), redeploy; the
CA stays stable so both sides keep trusting each other across a rotation.
