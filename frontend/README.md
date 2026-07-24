# smart-home 前端

Vue 3 + Vite + TypeScript + Tailwind CSS。

## 启动

```bash
cd frontend
pnpm install   # 或 npm install
pnpm dev       # http://127.0.0.1:5175
```

代理：`/api` · `/oauth` → `http://127.0.0.1:3002`

## 说明

当前为工程骨架 + 连通性页。完整 UI 参考 `../docs/frontend/ui-prototype/`。

## 移动端 / APK

- **不新开项目**：业务继续写在本目录 `src/`。  
- 需要安装包时：本工程接入 **Capacitor** → 构建 `dist` → 同步 `android/` → 出 APK。  
- 说明文档：[`../docs/frontend/移动端与APK.md`](../docs/frontend/移动端与APK.md)。  
- **现阶段**先做 Web；功能稳定后再初始化 Capacitor。
