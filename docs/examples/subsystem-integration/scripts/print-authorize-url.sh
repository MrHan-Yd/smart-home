#!/usr/bin/env bash
# 打印授权 URL；浏览器打开，登录后跳回 redirect 并带 code
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
[[ -f "$ROOT/.env" ]] && source "$ROOT/.env"

AUTH_BASE="${AUTH_BASE:-http://127.0.0.1:3000}"
CLIENT_ID="${CLIENT_ID:-demo_app}"
REDIRECT_URI="${REDIRECT_URI:-http://127.0.0.1:9999/oauth/callback}"

# 简易 PKCE（演示）
VERIFIER="$(openssl rand -hex 32)"
CHALLENGE="$(printf '%s' "$VERIFIER" | openssl dgst -binary -sha256 | openssl base64 -A | tr '+/' '-_' | tr -d '=')"
STATE="$(openssl rand -hex 8)"

echo "export CODE_VERIFIER=$VERIFIER"
echo "export OAUTH_STATE=$STATE"
echo ""
echo "在浏览器打开（先保证已能访问登录页）："
echo ""
echo "${AUTH_BASE}/oauth/authorize?response_type=code&client_id=${CLIENT_ID}&redirect_uri=$(python -c "import urllib.parse,os; print(urllib.parse.quote(os.environ.get('REDIRECT_URI','$REDIRECT_URI'),safe=''))" 2>/dev/null || echo "$REDIRECT_URI")&scope=openid%20profile%20email&state=${STATE}&code_challenge=${CHALLENGE}&code_challenge_method=S256"
echo ""
echo "回调后把 code 写入 .env 的 CODE=，并 export CODE_VERIFIER 后再跑 exchange-token.sh"
