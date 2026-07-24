# 智能家居子系统 — 移动端与 APK

> 结论：**不新开前端仓库**；在现有 **Vue 3 `frontend/`** 上，用 **Capacitor** 打 **Android APK**。  
> 关联：[技术选型.md](./技术选型.md) · [../backend/接口设计.md](../backend/接口设计.md)

---

## 1. 已定策略

| 项 | 决定 |
|----|------|
| 业务 UI | **Vue 3**（与 Web 同一套 `frontend/src`） |
| 安装包 | **Capacitor → Android APK**（自用侧载） |
| 工程位置 | **`smart-home/frontend/` 内**，不另建 App 项目 |
| 日常使用 | **响应式 Web** 优先（任意浏览器打开 + 收藏） |
| 不做（现阶段） | Flutter / RN 重写；小程序；以夸克 PWA 安装为主路径 |
| iOS | 可后续同一 Capacitor 工程出包，需 Mac；**当前优先安卓** |

---

## 2. 为什么选 APK 而不是只靠浏览器安装

| 方式 | 卸掉 Chrome 后 | 夸克等国产浏览器 | 自用评价 |
|------|----------------|------------------|----------|
| 浏览器直接打开 | 无关 | 可用 | **日常足够** |
| Chrome「安装应用 / 主屏幕」 | **常不可用**（依赖 Chrome） | 支持差 | 不作为主方案 |
| **Capacitor APK** | **可用** | 无关 | **要桌面图标 / 独立 App 时采用** |

结论：需要「装上就能用、不绑浏览器」→ **打 APK**；未打包前用 Web 即可。

---

## 3. 架构关系

```
┌─────────────────────────────────────┐
│  frontend/src  （Vue 页面与逻辑）     │
└───────────────┬─────────────────────┘
                │ pnpm build → dist/
        ┌───────┴────────┐
        ▼                ▼
  浏览器 / 反代      Capacitor
  （Web 部署）        │
                     ▼
              android/（WebView 壳）
                     │
                     ▼
                   APK 安装包
                     │
                     ▼  HTTPS / 局域网
              smart-home-service（Go）
                     │
                     ▼
              Home Assistant
```

- **后端 API 不变**；App 与 Web 共用 `/api/v1`、OAuth 方案（壳内注意 Cookie / 回调域名）。  
- **不直连 HA**；规则与 Web 相同。

---

## 4. 工程目录（打包阶段）

```
frontend/
├── src/                    # 唯一业务代码
├── dist/                   # vite build 产物（Capacitor webDir）
├── android/                # npx cap add android 后生成
├── capacitor.config.ts     # appId、appName、server 等
└── package.json            # @capacitor/core、cli、android
```

**禁止：** 再建 `smart-home-app/` 复制一套 Vue。

---

## 5. 实施节奏

| 阶段 | 内容 |
|------|------|
| **现在** | 完成 Web：OAuth、设备、控制、响应式；**不**初始化 Capacitor |
| **Web 可用后** | 在 `frontend` 安装 Capacitor；`webDir: dist`；`cap add android` |
| **联调** | 配置 API 基地址（局域网 IP / 域名）；OAuth 回调与 Cookie / 或改 Token |
| **出包** | Android Studio 或 CLI 打 debug/release APK，自用安装 |
| **可选** | iOS 工程、推送、安全存储 Token |

---

## 6. 打包时注意点（预研清单）

1. **可达性**：手机须能访问后端（家宽同网、或 frp/域名 + HTTPS）。  
2. **鉴权**：Web 用 Cookie BFF；App WebView 可继续 Cookie，但 **OAuth redirect、Cookie Domain、SameSite** 要按真机环境验；必要时再上 **Bearer Token** 双通道。  
3. **混合内容**：生产 API 建议 HTTPS，避免 Android 拦 HTTP。  
4. **清缓存**：发版后 `cap sync`，避免旧 dist。  
5. **权限**：自用控家居一般无需通讯录等；按功能再加。

---

## 7. 与「只做 Web」的关系

| 用户场景 | 推荐 |
|----------|------|
| 家里 Wi‑Fi 打开控制 | Web 书签即可 |
| 想要图标、不依赖浏览器 | 安装 APK |
| 开发调试 | 电脑/手机浏览器 → `5175` 或预发环境 |

两套入口，**一套 Vue 代码**。

---

## 8. 变更记录

| 日期 | 说明 |
|------|------|
| 2026-07-24 | 记录：Vue 同仓 + Capacitor APK；不做独立 App 仓库；PWA 非主路径 |
