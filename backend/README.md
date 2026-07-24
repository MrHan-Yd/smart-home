# smart-home-service（后端）

Go BFF · 统一认证 + Home Assistant。设计见 `../docs/backend/`。

## 已有

| 模块 | 说明 |
|------|------|
| health | `GET /healthz` · `GET /readyz` |
| meta | `GET /api/v1/meta` |
| HA | `GET /api/v1/ha/status` · `internal/adapter/hass` |
| 配置 | OAuth / PG / Redis / HA 环境变量 |
| 迁移 | `migrations/001_init.sql` |

## 启动

```bash
cd backend
cp .env.example .env
# 建库：CREATE DATABASE smart_home;
# psql "$DATABASE_URL" -f migrations/001_init.sql

go mod tidy
go run ./cmd/server
```

默认 `0.0.0.0:3002`。未配置 HA Token 也可启动（`READYZ_REQUIRE_HA=false`）。

## 下一步

- OAuth 登录（对齐 PM / 认证契约）
- 发现实体 · 设备 CRUD · actions
