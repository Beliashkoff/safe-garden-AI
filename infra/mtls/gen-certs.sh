#!/usr/bin/env bash
# Generates the mTLS trust chain for backend <-> llm-worker (ARCH §8.6, §11.2):
#   - ca.{key,pem}          private CA (keep ca.key offline / in a vault)
#   - worker.{key,crt}      server cert for Caddy on the worker VM (SAN = worker domain)
#   - api-client.{key,crt}  client cert the RU backend presents
#
# Output goes to infra/mtls/out/ (gitignored). Distribute per §8.6:
#   backend VM : ca.pem + api-client.crt + api-client.key  -> Yandex Lockbox
#   worker VM  : ca.pem + worker.crt + worker.key          -> LUKS volume (Caddy)
#
# Rotation: quarterly (regenerate leaf certs, keep the CA). See worker-vm/README.md.
set -euo pipefail

# Stop Git Bash / MSYS on Windows from rewriting the leading slash in openssl
# -subj arguments into a Windows path. No-op on Linux/macOS.
export MSYS_NO_PATHCONV=1

WORKER_DOMAIN="${WORKER_DOMAIN:-worker.agronomai.site}"
DAYS_CA="${DAYS_CA:-3650}"
DAYS_LEAF="${DAYS_LEAF:-100}"   # ~quarter + buffer; rotate on schedule

OUT="$(cd "$(dirname "$0")" && pwd)/out"
mkdir -p "$OUT"
cd "$OUT"

echo ">> CA"
openssl ecparam -name prime256v1 -genkey -noout -out ca.key
openssl req -x509 -new -nodes -key ca.key -sha256 -days "$DAYS_CA" \
  -subj "/CN=safe-garden-mtls-ca" -out ca.pem

gen_leaf() {
  local name="$1" cn="$2" ext="$3"
  echo ">> $name ($cn)"
  printf '%s\n' "$ext" > "$name.ext"
  openssl ecparam -name prime256v1 -genkey -noout -out "$name.key"
  openssl req -new -key "$name.key" -subj "/CN=$cn" -out "$name.csr"
  openssl x509 -req -in "$name.csr" -CA ca.pem -CAkey ca.key -CAcreateserial \
    -days "$DAYS_LEAF" -sha256 -extfile "$name.ext" -out "$name.crt"
  rm -f "$name.csr" "$name.ext"
}

# Worker server cert: needs SAN matching the domain Caddy serves.
gen_leaf worker "$WORKER_DOMAIN" \
  "subjectAltName=DNS:${WORKER_DOMAIN}
extendedKeyUsage=serverAuth"

# Backend client cert: client auth only.
gen_leaf api-client "safe-garden-api-client" \
  "extendedKeyUsage=clientAuth"

rm -f ca.srl
chmod 600 ./*.key
echo ">> done. Files in: $OUT"
ls -1 "$OUT"
