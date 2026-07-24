#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
[[ -f "$ROOT/.env" ]] && source "$ROOT/.env"

AUTH_BASE="${AUTH_BASE:-http://127.0.0.1:3000}"
CLIENT_ID="${CLIENT_ID:-demo_app}"
CLIENT_SECRET="${CLIENT_SECRET:-demo_secret_change_me}"
REDIRECT_URI="${REDIRECT_URI:-http://127.0.0.1:9999/oauth/callback}"

if [[ -z "${CODE:-}" ]]; then
  echo "请设置 CODE=（authorize 回调中的 code）" >&2
  exit 1
fi

ARGS=(
  -s -X POST "${AUTH_BASE}/oauth/token"
  -H "Content-Type: application/x-www-form-urlencoded"
  -d "grant_type=authorization_code"
  -d "code=${CODE}"
  -d "redirect_uri=${REDIRECT_URI}"
  -d "client_id=${CLIENT_ID}"
  -d "client_secret=${CLIENT_SECRET}"
)
if [[ -n "${CODE_VERIFIER:-}" ]]; then
  ARGS+=(-d "code_verifier=${CODE_VERIFIER}")
fi

echo "→ token" >&2
RESP="$(curl "${ARGS[@]}")"
echo "$RESP"
echo "" >&2
echo "将 refresh_token 写入 .env 的 REFRESH_TOKEN= 后可执行 refresh-token.sh" >&2
