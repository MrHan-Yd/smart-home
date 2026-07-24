# smart-home · 智能家居子系统

自用门户：统一认证 + Home Assistant + Web 管理/分析。  
移动端方向：**同一套 Vue**，稳定后用 **Capacitor 打 Android APK**（不另建 App 仓库）。详见 [docs/frontend/移动端与APK.md](./docs/frontend/移动端与APK.md)。

| 目录 | 说明 |
|------|------|
| [docs/](./docs/) | 架构 / 功能 / HA / 库表 / 接口 / UI 原型 / 移动端 |
| [backend/](./backend/) | Go 服务 `:3002` |
| [frontend/](./frontend/) | Vue 3 + Vite `:5175`（未来 APK 同源） |

## 快速启动

```bash
# 1. 后端
cd backend
cp .env.example .env   # 改 DATABASE_URL / REDIS_URL
# CREATE DATABASE smart_home;
# psql ... -f migrations/001_init.sql
go run ./cmd/server

# 2. 前端
cd frontend
pnpm install
pnpm dev
```

浏览器：http://127.0.0.1:5175  
设计稿：`docs/frontend/ui-prototype/index.html`
