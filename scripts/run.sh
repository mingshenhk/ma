#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"
export LISTEN_ADDR="${LISTEN_ADDR:-127.0.0.1:8080}"
export DATA_DIR="${DATA_DIR:-$ROOT/data}"
export STATIC_DIR="${STATIC_DIR:-$ROOT/frontend/dist}"
umask 077
mkdir -p "$DATA_DIR"
if [ -z "${ADMIN_KEY:-}" ]; then
  if [ ! -s "$DATA_DIR/admin-key.txt" ]; then
    od -An -N32 -tx1 /dev/urandom | tr -d ' \n' > "$DATA_DIR/admin-key.txt"
  fi
  ADMIN_KEY=$(cat "$DATA_DIR/admin-key.txt")
  export ADMIN_KEY
  printf '管理密钥保存在 %s/admin-key.txt\n' "$DATA_DIR"
fi
printf '服务监听 %s，按 Ctrl+C 停止。\n' "$LISTEN_ADDR"
exec "$ROOT/bin/haojiayuan"
