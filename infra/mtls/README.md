# mTLS trust chain (backend ↔ llm-worker)

`gen-certs.sh` builds a private CA and two leaf certs for the mutual-TLS link
between the RU backend and the Frankfurt worker (ARCH §8.6, §11.2). The worker
listens plain HTTP inside its VM; **Caddy terminates mTLS** in front of it
(`infra/docker/compose/prod-llmworker.Caddyfile`, `client_auth require_and_verify`).

## Generate

```bash
WORKER_DOMAIN=worker.agronomai.site API_DOMAIN=api.agronomai.site bash infra/mtls/gen-certs.sh
# → infra/mtls/out/{ca,worker,api-client,internal,worker-client}.{key,crt}  (ca.pem, not ca.crt)
```

`out/` is gitignored — certs and keys must never be committed.

## Distribute (ARCH §8.6)

Two mTLS directions: backend → worker (chat) and worker → backend internal
listener (recommend_fertilizer callback, §11.3). Each side gets the peer CA, its
own server cert (where it terminates TLS), and its own client cert (where it dials).

| Host | Files | Maps to |
| ---- | ----- | ------- |
| Backend VM (Yandex) | `ca.pem` | `INTERNAL_MTLS_CLIENT_CA_PATH`, `LLM_WORKER_CA_PATH` |
| Backend VM | `internal.crt`, `internal.key` | `INTERNAL_MTLS_CERT_PATH` / `..._KEY_PATH` (server side of the callback) |
| Backend VM | `api-client.crt`, `api-client.key` | `LLM_WORKER_CLIENT_CERT_PATH` / `..._KEY_PATH` (client to worker) |
| Worker VM | `ca.pem`, `worker.crt`, `worker.key` | Caddy `/etc/caddy/certs` (server side of chat), on **LUKS** |
| Worker VM | `worker-client.crt`, `worker-client.key` | worker's client cert for the callback (Stage 5.2), on **LUKS** |

Backend secrets go to Yandex **Lockbox** → mounted/written to the VM at deploy;
worker secrets live on the worker VM's LUKS volume. `internal.crt` SAN must match
`api.agronomai.site` (the host the worker dials for the callback).

Keep `ca.key` **offline** (it is not needed on any server — only to sign new
leaves). Rotation: regenerate leaf certs quarterly (`DAYS_LEAF`), redeploy; the
CA stays stable so both sides keep trusting each other across a rotation.
