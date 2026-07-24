#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
[[ -f "$ROOT/.env" ]] && source "$ROOT/.env"

AUTH_BASE="${AUTH_BASE:-http://127.0.0.1:3000}"
if [[ -z "${ACCESS_TOKEN:-}" ]]; then
  echo "请设置 ACCESS_TOKEN=" >&2
  exit 1
fi

curl -s "${AUTH_BASE}/oauth/userinfo" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
echo
