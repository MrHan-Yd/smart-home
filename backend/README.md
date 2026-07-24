# smart-home-service（后端）

Go BFF · 统一认证 + Home Assistant。设计见 `../docs/backend/`。

## 已实现（P0）

| 模块 | 路径 / 说明 |
|------|-------------|
| health | `GET /healthz` · `GET /readyz` |
| OAuth | `GET /oauth/login` · `/callback` · `/complete` · `POST /oauth/logout` |
| me | `GET /api/v1/me`（含默认 home） |
| meta / HA | `GET /api/v1/meta` · `GET /api/v1/ha/status` |
| 发现 | `GET /api/v1/discover/entities` |
| 设备 | CRUD + batch · `POST /api/v1/devices/{id}/actions` |
| Session | Cookie `sh_sid` + Redis；PKCE；refresh 单飞 |
| 迁移 | `001_init` · **`002_telemetry_energy`**（时序/日汇总表，分析用） |

## 启动

```powershell
cd backend
# 编辑 .env（OAuth client_id/secret、DATABASE_URL、REDIS_URL）
# 建库：CREATE DATABASE smart_home;
# psql ... -f migrations/001_init.sql
# psql ... -f migrations/002_telemetry_energy.sql   # 时序/用电分析入库
# IdP redirect 须为：http://127.0.0.1:3002/oauth/callback

.\run.ps1
# 或：
$env:GOTOOLCHAIN = "local"
$env:GOPROXY = "https://goproxy.cn,direct"
go run ./cmd/server
```

默认 `0.0.0.0:3002`。未配置 HA Token 可启动（`READYZ_REQUIRE_HA=false`）。

### Go 版本（与 ProjectManagement 统一）

| 项 | 值 |
|----|-----|
| `go.mod` | `go 1.24` + `toolchain go1.24.3` |
| 启动 | `.\run.ps1` 或 `.\start.cmd`（会拉/用 go1.24.3，再 `bin\server.exe`） |

本机即使装了 Go 1.26，也会按 `toolchain` 使用 **1.24.3**，与 PM 一致。

## 目录

```
internal/
  auth/           # session · oauth · jwks
  middleware/     # Require session
  module/user|home|device
  adapter/hass/   # REST + capability + action map
  httpserver/
```
