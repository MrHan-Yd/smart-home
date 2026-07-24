# 子系统接入联调包（无完整业务工程）

本目录**不是**业务子系统，只提供：

1. 环境变量模板  
2. 浏览器 + curl 走通 code → token → refresh → userinfo  
3. 无感刷新伪代码（复制到你的子系统）

**原则：** Token **只由统一认证中心** `/oauth/token` 颁发；脚本与子系统都是**向认证中心换票 / 验票**，不在本地签发 JWT。

完整契约：`docs/backend/开发/子系统接入契约.md`（见 §0 核心原则）。

## 前置

1. PG/Redis 已起，已执行 `001`、`002`、`003`、**`006_user_consents`** 迁移  
2. JWT 密钥已生成，`auth-service` 可启动  
3. **登录页可访问**（`pnpm dev` 或网关挂 auth-web）  
4. `APP_BASE_URL` 指向登录页根（如 `http://127.0.0.1:5173`）；与浏览器地址 **同源**（勿混用 `localhost` / `127.0.0.1`）  
5. 演示客户端：`client_id=demo_app`，secret 见 `DEMO_CLIENT_SECRET`（默认 `demo_secret_change_me`）  
6. 开发态 Vite 已代理 `/oauth` → 后端，authorize 可用 `http://127.0.0.1:5173/oauth/authorize?...`

若子系统回调不是 `http://127.0.0.1:9999/oauth/callback`，先在管理台改 `demo_app` 的 redirect_uris。

### 预期浏览器路径

1. 打开 authorize URL（未登录）→ 登录页（带 `redirect=/oauth/authorize...`）  
2. 登录 → 回到 authorize  
3. **首次**该 client → 同意页「授权请求」→ 点授权  
4. 302 到子系统 `redirect_uri?code=&state=`  
5. 同用户再次 authorize → **不再**弹同意页

## 快速路径

```bash
# 复制并编辑
cp .env.example .env

# Windows PowerShell：点开授权 URL（需已登录认证中心，或会先跳登录）
# 见 scripts/print-authorize-url.ps1

# 拿到浏览器地址栏 code 后：
# bash scripts/exchange-token.sh
# bash scripts/refresh-token.sh
```

## 子系统无感刷新（伪代码）

```text
// 服务端中间件 / API 客户端
async function apiFetch(url, opts):
  token = session.access_token
  if almostExpired(token) or lastStatus == 401:
    await refreshSingleFlight()   // 全局锁，禁止并发 refresh
  res = fetch(url, Authorization: Bearer session.access_token)
  if res.status == 401:
    ok = await refreshSingleFlight()
    if not ok: redirectToLogin()
    res = fetch(url, Authorization: Bearer session.access_token)
  return res

async function refreshSingleFlight():
  lock
  body = grant_type=refresh_token&refresh_token=...&client_id=...&client_secret=...
  r = POST AUTH/oauth/token
  if fail: clearSession; return false
  session.access = r.access_token
  session.refresh = r.refresh_token   // 轮换：必须覆盖
  session.exp = now + r.expires_in
  unlock
  return true
```

## 安全提醒

- `client_secret`、refresh 只放子系统**服务端**  
- 生产改掉 `demo_secret_change_me` 与默认 admin 密码  
