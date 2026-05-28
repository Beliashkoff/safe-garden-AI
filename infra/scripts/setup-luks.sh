#!/usr/bin/env bash
# One-time LUKS setup for worker secrets on the HostKey VM (ARCH §8.6, §11.7).
#
# Creates an encrypted volume backed by a file, mounts it at /etc/llmworker, and
# leaves you to drop in the worker .env + mTLS certs. After every reboot the
# volume must be re-opened MANUALLY (open-luks below) — a disk snapshot taken by
# the VPS operator without the passphrase yields no ANTHROPIC_API_KEY / mTLS key.
#
# Usage:
#   sudo bash setup-luks.sh init        # first time: create + format + mount
#   sudo bash setup-luks.sh open        # after a reboot: re-open + mount
#   sudo bash setup-luks.sh close       # unmount + close
set -euo pipefail

IMG="${LUKS_IMG:-/var/lib/llmworker.luks}"
SIZE_MB="${LUKS_SIZE_MB:-64}"
MAPPER="llmworker"
MOUNT="/etc/llmworker"
OWNER="${SECRET_OWNER:-safegarden}"

require_root() { [ "$(id -u)" -eq 0 ] || { echo "run as root"; exit 1; }; }

case "${1:-}" in
  init)
    require_root
    [ -f "$IMG" ] && { echo "$IMG already exists; refusing to overwrite"; exit 1; }
    fallocate -l "${SIZE_MB}M" "$IMG"
    chmod 600 "$IMG"
    echo ">> luksFormat (you will set the passphrase)"
    cryptsetup luksFormat "$IMG"
    echo ">> luksOpen"
    cryptsetup luksOpen "$IMG" "$MAPPER"
    mkfs.ext4 -q "/dev/mapper/$MAPPER"
    mkdir -p "$MOUNT"
    mount "/dev/mapper/$MAPPER" "$MOUNT"
    mkdir -p "$MOUNT/certs"
    chown -R "$OWNER":"$OWNER" "$MOUNT"
    chmod 700 "$MOUNT"
    echo ">> mounted at $MOUNT. Place .env (chmod 600) and certs/ here."
    ;;
  open)
    require_root
    cryptsetup luksOpen "$IMG" "$MAPPER"
    mkdir -p "$MOUNT"
    mount "/dev/mapper/$MAPPER" "$MOUNT"
    echo ">> reopened $MOUNT"
    ;;
  close)
    require_root
    umount "$MOUNT" || true
    cryptsetup luksClose "$MAPPER" || true
    echo ">> closed"
    ;;
  *)
    echo "usage: $0 {init|open|close}"; exit 1;;
esac
