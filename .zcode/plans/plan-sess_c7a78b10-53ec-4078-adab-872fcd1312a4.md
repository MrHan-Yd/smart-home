## P2 实现计划

四块并行推进，分批验证。新增依赖：后端 `github.com/coder/websocket`（纯 Go、活跃维护，比 gorilla 更省依赖）；前端无新依赖。

---

### 批次 1 · 数据库迁移 `003_p2.sql`

对齐 `docs/backend/数据库设计.md` §3.8/§3.9：
- **`daily_device_stats`**：`id, user_id, device_id, stat_date date, on_count int, on_duration_sec bigint, energy_kwh numeric(12,4), extra jsonb`；唯一 `(device_id, stat_date)`；索引 `(user_id, stat_date)`。
- **`telemetry_samples`**：`device_id, entity_id, ts timestamptz, state text, num_value double precision, attributes jsonb`；索引 `(entity_id, ts desc)`、`(device_id, ts desc)`。用 `CREATE TABLE IF NOT EXISTS`。

---

### 批次 2 · telemetry 采集与同步入库

**采集层**（与批次 3 共建）：后端订阅 HA WS `state_changed` 事件（见批次 3），每条已纳入 entity 的变更落 `telemetry_samples`（瘦身 attributes）。
**日聚合**：`module/analytics/repo.go` 提供 `Aggregate(ctx, date)` —— 遍历用户设备，按 telemetry_samples 当天统计：`on_count`（state 由非 on→on 的次数）、`on_duration_sec`（on 持续秒数，按相邻样本 ts 差累加）、`energy_kwh`（有 power/energy 实体时从 samples num_value 累加，否则 null）。upsert 进 `daily_device_stats`。
触发方式：`GET /api/v1/analytics/*` 时按需聚合缺失的最近 N 天（懒聚合，避免定时任务复杂度），并在 WS 事件写入后异步更新当天累计（轻量增量）。

---

### 批次 3 · WS 实时推送

**后端 HA 订阅** — `adapter/hass/subscribe.go`：用 `coder/websocket` 连 `{base}/api/websocket`，鉴权（access_token），`subscribe_events` 过 type=`state_changed`；收到事件后回调 `func(entityID string, state State)`。断线指数退避重连，重连后全量 `GetStates` 补偿一次。
**HA Hub** — `adapter/hass/hub.go`：维护「已纳入 entity_id 集合」（Server 启动 + 设备增删时刷新），把 HA 订阅事件过滤后分发给本系统前端 WS 连接。
**本系统 WS** — `handlers_ws.go` `GET /api/v1/ws`（Cookie 鉴权）：升级连接，订阅 hub；下行 `{type:"state_changed", device: DeviceView}`（复用 deviceView，查设备表 + 实时 state）；上行 `{type:"ping"}`→pong。前端断线降级短轮询。
集成：Server 启动时若 HA 配置则 `hass.StartHub(ctx, ha, devicesRepo, log)`；设备增删后通知 hub 刷新 entity 集合。

**前端** — `composables/useWS.ts`：连 `/api/v1/ws`，收到 state_changed 时更新对应设备缓存（通过事件总线或直接 patch OverviewView/DevicesView 的 devices 数组）。OverviewView 现有 15s 轮询保留作降级，WS 连上后可拉长到 60s；断线回退 15s。`lib/http` 已是 cookie，WS 同源无需额外。

---

### 批次 4 · 使用分析看板

**后端** `handlers_analytics.go` + `module/analytics/repo.go`：
- `GET /api/v1/analytics/summary?range=7d|30d`：activation_count（区间 on_count 总和）、runtime_hours（on_duration_sec 总和/3600）、energy_kwh、avg_temperature/humidity（temperature/humidity 类 sensor 当天均值）、on_count（当前在线且 on 的可开关设备数，拉 HA states）、online_count。无数据 null。
- `GET /api/v1/analytics/runtime`：每日每设备 on 时长 → `[{date, device_id, entity_id, hours}]`。
- `GET /api/v1/analytics/ranking`：按区间 on 时长/次数排行的设备 Top N。
- `GET /api/v1/analytics/type-mix`：domain → 设备数/总时长占比。
- `GET /api/v1/analytics/heatmap`：device×hour 的 on 次数矩阵（供热力图）。
- `GET /api/v1/analytics/environment`：温湿度 sensor 按日均值序列。

区间聚合优先查 `daily_device_stats`（已聚合的天）；今天（未聚合）实时算 telemetry_samples。

**前端** — 新增 `views/AnalyticsView.vue`（路由 `/analytics`，侧栏「分析」入口），用现有 `HistoryChart` + 新增轻量 SVG 条形/占比环/热力网格。KPI 卡片复用 `.stat-card`。无新图表库。

---

### 批次 5 · scene/script 激活 + media/vacuum 控件

**scene/script**：后端 action.go 已支持 `activate`(scene)/`run`(script)。前端在 `AddDeviceView` 的 domain chip 已含 scene/script；`DeviceControls.vue` 增分支：scene/script → 单个大按钮「激活/运行」（`activate`/`run`）；`DeviceCard` 对 action 级 domain 显示「运行」而非 toggle。`isOn` 对 scene 不适用——卡片只显示「执行」按钮。
**media_player**：`DeviceControls` 增 on_off（已有）+ `dc-slider` 音量（attributes.volume_level 0–1 → set_volume，action.go 补 set_volume 映射 media_player.set_volume）+ 播放/暂停（play_pause → media_player.media_play_pause）。
**vacuum**：start/stop（start 已有；补 stop → vacuum.stop/`pause`）+ return_to_base（已有）。
后端 action.go 补：`set_volume`、`play_pause`、`pause`(vacuum) 映射 + `ActionAllowed` 放行。

---

### 验证
- 每批 `GOTOOLCHAIN=go1.24.3 go build ./internal/... && go vet ./internal/...`；前端 `pnpm build`。
- WS / 分析需 HA + 已采集数据，以人工联调为准，不写集成测试。

### 不做（边界）
- 自动化规则引擎（用 HA）、视频监控、多 HA 切换 UI、CSV 导出、WS 消息压缩。