#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
[[ -f "$ROOT/.env" ]] && source "$ROOT/.env"

AUTH_BASE="${AUTH_BASE:-http://127.0.0.1:3000}"
CLIENT_ID="${CLIENT_ID:-demo_app}"
CLIENT_SECRET="${CLIENT_SECRET:-demo_secret_change_me}"

if [[ -z "${REFRESH_TOKEN:-}" ]]; then
  echo "请设置 REFRESH_TOKEN=" >&2
  exit 1
fi

echo "→ refresh（注意：旧 refresh 将失效，请用响应中的新 refresh_token）" >&2
curl -s -X POST "${AUTH_BASE}/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token" \
  -d "refresh_token=${REFRESH_TOKEN}" \
  -d "client_id=${CLIENT_ID}" \
  -d "client_secret=${CLIENT_SECRET}"
echo
