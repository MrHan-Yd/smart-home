# smart-home 前端

Vue 3 + Vite + TS + Tailwind · 视觉对齐 `docs/frontend/ui-prototype/`。

## 启动

```bash
pnpm install
pnpm dev
```

http://127.0.0.1:5175 · API 代理到 `:3002`

## P0 页面

| 路由 | 说明 |
|------|------|
| `/login` | SSO 入口 |
| `/` | 总览状态墙 |
| `/devices` | 设备列表 / 筛选 |
| `/devices/:id` | 详情 + 开关 |
| `/add` | HA 发现与纳入 |
| `/settings` | 账号 + HA 状态 |

## 说明

- Cookie 会话：`credentials: 'include'`；401 → `/oauth/login`
- 开发请统一用 `127.0.0.1`（勿混 localhost）
- 移动端 APK：见 `../docs/frontend/移动端与APK.md`（Web 稳定后再 Capacitor）
