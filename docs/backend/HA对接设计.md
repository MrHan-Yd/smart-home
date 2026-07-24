# 智能家居子系统 — Home Assistant 对接设计

> 依据：[Home Assistant 官方 Integrations](https://www.home-assistant.io/integrations/)（building block domains）  
> 关联：[设备类型与接入设计.md](./设备类型与接入设计.md) · [架构设计.md](./架构设计.md) · [接口设计.md](./接口设计.md)

---

## 1. 目标与边界

| 要做 | 不做 |
|------|------|
| 用 **一套** HA REST（+ 后续 WS）对接所有品牌设备 | 直连米家/涂鸦等厂商协议 |
| 按 **domain** 映射 state / attributes / services | 替代 HA 配网与集成安装 |
| 服务端持有 Long-Lived Token | 前端持有 HA Token 或直连 `:8123` |
| 能力裁剪：设备不支持的 attr/service 不暴露 | 假设所有灯都有彩色 |

**权威来源：** HA 各 domain 文档中的 **State / Attributes / Actions（services）**。本系统逻辑 action 再映射到 HA service。

---

## 2. 连接与鉴权

### 2.1 配置

| 项 | 说明 | 示例 |
|----|------|------|
| `HASS_BASE_URL` | HA 根，无尾斜杠 | `http://192.168.1.10:8123` |
| `HASS_TOKEN` | 用户资料 → 长期访问令牌 | `eyJ...` |
| `HASS_TIMEOUT` | HTTP 超时 | `10s` |
| `HASS_TLS_INSECURE` | 仅开发自签证书 | 默认 false |

P0：全局环境变量。P1+：可按 `homes` 表存多实例（token 加密）。

### 2.2 请求约定

```http
Authorization: Bearer {HASS_TOKEN}
Content-Type: application/json
```

| HA API | 方法 | 本系统用途 |
|--------|------|------------|
| `/api/` | GET | 探活（返回 `API running.`） |
| `/api/config` | GET | 版本、location 等（可选） |
| `/api/states` | GET | 发现 + 列表状态 |
| `/api/states/{entity_id}` | GET | 单实体详情 |
| `/api/services/{domain}/{service}` | POST | 控制 |
| `/api/history/period/{timestamp}` | GET | 历史 |
| `/api/logbook` | GET | 可选审计补充 |
| WebSocket `/api/websocket` | WS | P2 实时 |

### 2.3 控制调用体

```http
POST {HASS_BASE_URL}/api/services/{domain}/{service}
Authorization: Bearer ...
Content-Type: application/json

{
  "entity_id": "light.living_room",
  "brightness": 180
}
```

- `entity_id` 可为 string 或 list。  
- 其余字段为该 service 的 data（与 HA 文档一致）。  
- 成功：HA 常返回变更后的 state 列表；本系统统一再 `GET states/{id}` 或用返回体刷新视图。

### 2.4 错误映射

| HA / 网络 | HTTP（本系统） | 说明 |
|-----------|----------------|------|
| 连接失败 / 超时 | 502 | `ha_unreachable` |
| 401/403 | 502 | `ha_auth_failed`（检查 token） |
| 404 entity | 404 | `entity_not_found` |
| 400 参数 | 400 | `ha_bad_request` + 摘要 |
| 实体 unavailable | 409 或 200+提示 | 控制前校验 state |

### 2.5 缓存与限流

| 数据 | 策略 |
|------|------|
| `/api/states` 全量 | 内存/Redis 短缓存 2～5s；控制后对该 entity 主动失效 |
| 单 entity | 控制后拉取；列表可合并缓存 |
| 并发 | 同一 entity 控制串行（可选队列） |
| History | 按请求透传，P2 再入库 |

---

## 3. Entity 模型（HA → 本系统）

### 3.1 State 对象（HA 原始）

```json
{
  "entity_id": "light.living_room",
  "state": "on",
  "attributes": {
    "friendly_name": "客厅灯",
    "brightness": 180,
    "supported_color_modes": ["color_temp", "hs"],
    "...": "..."
  },
  "last_changed": "2026-07-24T10:00:00.000Z",
  "last_updated": "2026-07-24T10:00:00.000Z",
  "context": { "id": "...", "parent_id": null, "user_id": null }
}
```

### 3.2 本系统 DeviceView（统一 DTO）

```json
{
  "id": "uuid",
  "entity_id": "light.living_room",
  "domain": "light",
  "name": "客厅灯",
  "room": "客厅",
  "state": "on",
  "available": true,
  "primary_display": "开启 · 70%",
  "capabilities": ["on_off", "brightness", "color_temp", "color_hs", "effect"],
  "attributes": { },
  "control_level": "full",
  "ha": {
    "last_changed": "...",
    "last_updated": "..."
  }
}
```

| 字段 | 规则 |
|------|------|
| `available` | `state ∉ {unavailable, unknown}`（unknown 可标半可用） |
| `capabilities` | TypeProfile + attributes 裁剪（§4～§5） |
| `control_level` | `full` / `partial` / `read_only` |
| `name` | 本地 display_name 优先，否则 `friendly_name`，否则 object_id |

### 3.3 Domain 解析

```
entity_id = "{domain}.{object_id}"
domain = 第一段
```

---

## 4. 能力解析算法

```
function resolveCapabilities(state):
  domain = parseDomain(state.entity_id)
  profile = Registry[domain] ?? { caps: [raw_state], services: {} }
  attrs = state.attributes || {}
  caps = []

  for cap in profile.default_caps:
    if capRequires(cap, attrs, state):  # 见各 domain 节
      caps.append(cap)

  if caps 为空: caps = [raw_state]
  return caps
```

**原则：** attributes 没有对应字段 / supported 列表不含 → **不暴露**该 capability（避免 UI 调了 HA 报错）。

---

## 5. 分 Domain 对接明细

> 文档链接为 HA 官方 building block 页。实现以当前 HA 版本为准；字段名以运行时 attributes 为准。

---

### 5.1 `light` — [文档](https://www.home-assistant.io/integrations/light/)

| 项 | 内容 |
|----|------|
| **State** | `on` / `off` / `unavailable` / `unknown` |
| **常用 attributes** | `friendly_name`, `brightness` (0–255), `color_temp_kelvin` / `color_temp` (mired), `min/max_color_temp_kelvin`, `hs_color` `[h,s]`, `rgb_color`, `xy_color`, `effect`, `effect_list`, `supported_color_modes`, `color_mode`, `brightness_pct` |
| **Services** | `light.turn_on`, `light.turn_off`, `light.toggle` |

**turn_on 可选 data（节选）：**

| 字段 | 说明 |
|------|------|
| `brightness` | 0–255 |
| `brightness_pct` | 0–100（与 brightness 二选一即可） |
| `color_temp_kelvin` | 色温 K |
| `color_temp` | mired（旧） |
| `hs_color` | `[hue 0–360, sat 0–100]` |
| `rgb_color` | `[r,g,b]` |
| `xy_color` | `[x,y]` |
| `effect` | 须在 `effect_list` |
| `transition` | 秒，渐变 |
| `flash` | `short` / `long` |
| `profile` | light profile 名 |

**能力裁剪：**

| capability | 条件 |
|------------|------|
| `on_off` | 始终（可控 light） |
| `brightness` | 存在 `brightness` 或 color_modes 含 brightness 相关 |
| `color_temp` | `color_temp_kelvin` 或 modes 含 `color_temp` |
| `color_hs` | `hs_color` 或 modes 含 `hs` |
| `color_rgb` | modes 含 `rgb` / 有 `rgb_color` |
| `effect` | 非空 `effect_list` |

**逻辑 action → HA：**

| action | service | data |
|--------|---------|------|
| `turn_on` | `light.turn_on` | entity_id + 可选字段 |
| `turn_off` | `light.turn_off` | entity_id, transition? |
| `toggle` | `light.toggle` | entity_id |
| `set_brightness` | `light.turn_on` | brightness / brightness_pct |
| `set_color_temp` | `light.turn_on` | color_temp_kelvin |
| `set_hs_color` | `light.turn_on` | hs_color |
| `set_effect` | `light.turn_on` | effect |
| `set_transition` | 附带在 on/off/toggle | transition |

**实现阶段：** P0 `on_off`；P1 亮度/色温/HS/effect/transition。

---

### 5.2 `switch` — [文档](https://www.home-assistant.io/integrations/switch/)

| 项 | 内容 |
|----|------|
| **State** | `on` / `off` / unavailable / unknown |
| **attributes** | `friendly_name`, `device_class` (`outlet` / `switch` / none), 可选功率等自定义 |
| **Services** | `switch.turn_on`, `switch.turn_off`, `switch.toggle` |

| action | service | data |
|--------|---------|------|
| `turn_on` / `turn_off` / `toggle` | 对应 switch.* | `{ "entity_id": "..." }` |

**能力：** 仅 `on_off`。  
**阶段：** P0。

同构：`input_boolean` 使用 `input_boolean.turn_on/off/toggle`（domain 不同，mapper 分支）。

---

### 5.3 `climate` — [文档](https://www.home-assistant.io/integrations/climate/)

| 项 | 内容 |
|----|------|
| **State（hvac mode）** | `off`, `heat`, `cool`, `heat_cool`, `auto`, `dry`, `fan_only`, unavailable, unknown |
| **关键 attributes** | `hvac_modes`, `hvac_mode`, `hvac_action` (heating/cooling/idle/…), `temperature`, `current_temperature`, `target_temp_high/low`, `min_temp`, `max_temp`, `fan_mode`, `fan_modes`, `swing_mode`, `swing_modes`, `swing_horizontal_mode(s)`, `preset_mode`, `preset_modes`, `current_humidity`, `humidity`, `min/max_humidity` |

**Services：**

| Service | 用途 |
|---------|------|
| `climate.turn_on` / `turn_off` / `toggle` | 电源 |
| `climate.set_hvac_mode` | `{ hvac_mode }` |
| `climate.set_temperature` | `{ temperature }` 或 high/low |
| `climate.set_fan_mode` | `{ fan_mode }` |
| `climate.set_swing_mode` | `{ swing_mode }` |
| `climate.set_swing_horizontal_mode` | 水平摆风 |
| `climate.set_preset_mode` | `{ preset_mode }` |
| `climate.set_humidity` | `{ humidity }` |

**能力裁剪：**

| capability | 条件 |
|------------|------|
| `on_off` | 支持 turn_on/off 或 hvac 含 off |
| `climate_hvac` | 有 `hvac_modes` |
| `climate_temp` | 有 temperature 相关 attr |
| `climate_fan` | 有 `fan_modes` |
| `climate_swing` | 有 `swing_modes` |
| `climate_preset` | 有 `preset_modes` |
| `climate_humidity` | 有 humidity 设定相关 |

**逻辑 action → HA：**

| action | service | data |
|--------|---------|------|
| `turn_on` / `turn_off` / `toggle` | climate.* | entity_id |
| `set_hvac_mode` | set_hvac_mode | hvac_mode |
| `set_temperature` | set_temperature | temperature |
| `set_fan_mode` | set_fan_mode | fan_mode |
| `set_swing_mode` | set_swing_mode | swing_mode |
| `set_preset_mode` | set_preset_mode | preset_mode |
| `set_humidity` | set_humidity | humidity |

**阶段：** P1（电源+模式+温度优先，其余按 attr 有则显示）。

---

### 5.4 `cover` — [文档](https://www.home-assistant.io/integrations/cover/)

| 项 | 内容 |
|----|------|
| **State** | `opening`, `open`, `closing`, `closed`, unavailable, unknown |
| **device_class** | awning, blind, curtain, damper, door, garage, gate, shade, shutter, window… |
| **attributes** | `current_position` 0–100, `current_tilt_position` 0–100, `device_class` |

**Services：**

| Service | 说明 |
|---------|------|
| `cover.open_cover` / `close_cover` / `stop_cover` / `toggle` | 开合 |
| `cover.set_cover_position` | `{ position: 0–100 }` |
| `cover.open_cover_tilt` / `close_cover_tilt` / `stop_cover_tilt` / `toggle_cover_tilt` | 倾角 |
| `cover.set_cover_tilt_position` | `{ tilt_position: 0–100 }` |

| capability | 条件 |
|------------|------|
| `open_close` | 默认 |
| `position` | 有 `current_position` 或支持 set position |
| `tilt` | 有 tilt 相关 attr |

| action | service |
|--------|---------|
| `open` / `close` / `stop` / `toggle` | open/close/stop/toggle_cover |
| `set_position` | set_cover_position |
| `open_tilt` / `close_tilt` / `stop_tilt` | *_tilt |
| `set_tilt_position` | set_cover_tilt_position |

**阶段：** P1。

---

### 5.5 `fan` — [文档](https://www.home-assistant.io/integrations/fan/)

| 项 | 内容 |
|----|------|
| **State** | `on` / `off` |
| **attributes** | `percentage`, `percentage_step`, `preset_mode`, `preset_modes`, `oscillating`, `direction` (`forward`/`reverse`) |

**Services：** `fan.turn_on`, `turn_off`, `toggle`, `set_percentage`, `set_preset_mode`, `oscillate` (`oscillating: true/false`), `set_direction`, `increase_speed`, `decrease_speed`。

| action | service | data |
|--------|---------|------|
| `turn_on/off/toggle` | fan.* | |
| `set_percentage` | set_percentage | percentage |
| `set_preset_mode` | set_preset_mode | preset_mode |
| `set_oscillating` | oscillate | oscillating |
| `set_direction` | set_direction | direction |
| `increase_speed` / `decrease_speed` | 对应 | |

**阶段：** P1～P2。

---

### 5.6 `media_player` — [文档](https://www.home-assistant.io/integrations/media_player/)

| 项 | 内容 |
|----|------|
| **State** | `off`, `on`, `idle`, `playing`, `paused`, `buffering`, … |
| **attributes** | `media_title`, `media_artist`, `volume_level` 0–1, `is_volume_muted`, `source`, `source_list`, `media_content_id`, … |

**常用 services：**  
`turn_on/off/toggle`, `media_play`, `media_pause`, `media_stop`, `media_next_track`, `media_previous_track`, `volume_set` (`volume_level`), `volume_mute` (`is_volume_muted`), `select_source` (`source`), `volume_up/down` 等。

| action | service |
|--------|---------|
| `turn_on/off/toggle` | media_player.* |
| `play` / `pause` / `stop` | media_play/pause/stop |
| `next` / `previous` | media_next/previous_track |
| `set_volume` | volume_set |
| `set_mute` | volume_mute |
| `select_source` | select_source |

**阶段：** P2。

---

### 5.7 `lock` — [文档](https://www.home-assistant.io/integrations/lock/)

| State | `locked`, `unlocked`, `locking`, `unlocking`, `jammed`, … |
| Services | `lock.lock`, `lock.unlock`, `lock.open`（部分） |

| action | service | 注意 |
|--------|---------|------|
| `lock` | lock.lock | |
| `unlock` | lock.unlock | **建议二次确认** |
| `open` | lock.open | 若支持 |

**阶段：** P1；UI 危险操作确认。

---

### 5.8 `vacuum` — [文档](https://www.home-assistant.io/integrations/vacuum/)

| attributes | `battery_level`, `fan_speed`, `fan_speed_list`, `status` |
| Services | `start`, `pause`, `stop`, `return_to_base`, `locate`, `set_fan_speed`, `clean_spot` 等（随集成变化） |

| action | service（常见） |
|--------|-----------------|
| `start` / `pause` / `stop` | vacuum.start/pause/stop |
| `return_to_base` | return_to_base |
| `set_fan_speed` | set_fan_speed |
| `locate` | locate |

**阶段：** P2。实现前用目标实体 `GET /api/states` + Developer Tools 核对可用 service。

---

### 5.9 `scene` / `script`

| Domain | Service | action |
|--------|---------|--------|
| scene | `scene.turn_on` | `activate` |
| script | `script.turn_on` / `script.turn_off` / 或 `script.{object_id}` | `run` / `stop` |

**能力：** `scene_activate` / `script_run`。  
**阶段：** P2。本系统不编辑场景步骤。

---

### 5.10 `button` / `input_button`

| Service | `button.press` / `input_button.press` |
| action | `press` |
| 特点 | 瞬时，无持续 on |

**阶段：** P1。

---

### 5.11 电耗 / 功率（sensor 子集）

HA **可以**提供电耗，但依赖集成是否暴露实体：

| device_class | 含义 | 常见单位 |
|--------------|------|----------|
| `power` | 瞬时功率 | W / kW |
| `energy` | 累计电量 | kWh |

本系统经 `GET /api/states` 读取；**不**直连电表。分期与任务见 **[电耗与功率.md](./电耗与功率.md)**。

### 5.12 `sensor` / `binary_sensor` — 只读

| Domain | State | 控制 |
|--------|-------|------|
| sensor | 数值或枚举字符串 | **无** service 控制 |
| binary_sensor | `on` / `off` | **无** |

**attributes：** `unit_of_measurement`, `device_class`, `state_class`, `friendly_name` 等。

| capability | `numeric_sensor` / `binary_sensor` / `raw_state` |
| control_level | `read_only` |
| 历史 | history API 画曲线（P1） |

**阶段：** P0 展示与纳入。

---

### 5.13 其它 domain（默认策略）

| Domain | 策略 |
|--------|------|
| `number` / `input_number` | P2：`set_value` |
| `select` / `input_select` | P2：`select_option` |
| `remote` | P3：按键 |
| `camera` | P2 可选：取 snapshot URL，不做录像中台 |
| `person` / `zone` / `sun` / `event` 等 | 发现可黑名单；或只读 |
| **未注册** | `raw_state`，可添加，只读 |

---

## 6. ActionMapper 总表（实现清单）

统一入口：`POST /api/v1/devices/{id}/actions` → 校验 capability → Mapper → HA。

| 逻辑 action | 适用 domain（主） | HA service 模式 |
|-------------|-------------------|-----------------|
| turn_on / turn_off / toggle | light, switch, fan, climate, media_player, input_boolean | `{domain}.{action}` |
| set_brightness | light | light.turn_on |
| set_color_temp | light | light.turn_on |
| set_hs_color | light | light.turn_on |
| set_effect | light | light.turn_on |
| set_hvac_mode | climate | climate.set_hvac_mode |
| set_temperature | climate | climate.set_temperature |
| set_fan_mode | climate | climate.set_fan_mode |
| set_swing_mode | climate | climate.set_swing_mode |
| set_preset_mode | climate / fan | *.set_preset_mode |
| set_humidity | climate | climate.set_humidity |
| open / close / stop | cover | cover.open/close/stop_cover |
| set_position | cover | set_cover_position |
| set_tilt_position | cover | set_cover_tilt_position |
| set_percentage | fan | fan.set_percentage |
| set_oscillating | fan | fan.oscillate |
| set_direction | fan | fan.set_direction |
| play / pause / stop / next / previous | media_player | media_* |
| set_volume | media_player | volume_set |
| set_mute | media_player | volume_mute |
| select_source | media_player | select_source |
| lock / unlock | lock | lock.lock / unlock |
| start / pause / stop / return_to_base | vacuum | vacuum.* |
| activate | scene | scene.turn_on |
| run | script | script.turn_on |
| press | button | button.press |

未支持 action → `400 unsupported_action`。  
无 capability → `400 capability_required`。

---

## 7. 发现与过滤

```
GET /api/states
  → 解析全部 entity
  → 去掉 denylist domains
  → 去掉 entity_id regex
  → 标记 already_added（对照本地 devices 表）
  → 计算 capabilities + control_level
  → 返回发现列表
```

**建议默认 denylist domains：**  
`zone`, `person`, `persistent_notification`, `tts`, `stt`, `conversation`, `update`（可配置）, `device_tracker`（可选）

**建议保留：** 全部 `light/switch/sensor/binary_sensor/climate/cover/fan/media_player/lock/vacuum/scene/script/button/...`

---

## 8. History

```
GET /api/history/period/{start}?filter_entity_id={id}&end_time={end}&minimal_response
```

| 参数 | 说明 |
|------|------|
| start | ISO8601，路径或 query 以 HA 版本文档为准 |
| filter_entity_id | 逗号分隔 |
| end_time | 结束时间 |
| significant_changes_only | 可选 |

本系统 P1：代理查询并转成 `{ t, state, attributes? }[]`。  
P2：增量写入自有库。

**注意：** HA recorder 默认保留约 10 天；长期分析需调 HA 或自建同步。

---

## 9. WebSocket（P2）

1. 连接 `ws(s)://{host}/api/websocket`  
2. `auth` + access_token  
3. `subscribe_events` → `state_changed`  
4. 过滤已纳入 entity_id → 推送到本系统前端 WS  

断线重连 + 全量 states 补偿。

---

## 10. Go 适配层结构

```
internal/adapter/hass/
  client.go       # HTTP：GetStates, GetState, CallService, GetHistory, Ping
  types.go        # State, ServicePayload
  mapper/
    registry.go   # domain → profile
    light.go
    switch.go
    climate.go
    cover.go
    fan.go
    media_player.go
    lock.go
    vacuum.go
    sensor.go
    fallback.go
  action.go       # LogicalAction → domain/service/data
```

**CallService 签名示意：**

```go
CallService(ctx, domain, service string, data map[string]any) error
// data 必含 entity_id
```

---

## 11. 安全

1. Token 仅服务端；日志禁止打印 Token 与完整 Authorization  
2. 仅允许操作 **当前用户已纳入** 的 entity_id  
3. 禁止客户端传入任意 HA service 字符串（只认白名单 action）  
4. lock.unlock 等可加确认头 `X-Confirm: true`（可选）  
5. HA 内网优先；公网须 TLS + 强密码/VPN  

---

## 12. 联调检查清单

- [ ] `GET /api/` 探活成功  
- [ ] `GET /api/states` 有实体  
- [ ] light/switch turn_on/off 生效  
- [ ] 无 brightness 的灯不出现亮度滑条  
- [ ] climate 仅显示 hvac_modes 内模式  
- [ ] cover 无 position 时不显示位置滑条  
- [ ] sensor 不可调用 turn_on  
- [ ] 错误 token → 明确 502  
- [ ] 控制审计写入（P1）  

---

## 13. 参考链接

| Domain | URL |
|--------|-----|
| Light | https://www.home-assistant.io/integrations/light/ |
| Switch | https://www.home-assistant.io/integrations/switch/ |
| Climate | https://www.home-assistant.io/integrations/climate/ |
| Cover | https://www.home-assistant.io/integrations/cover/ |
| Fan | https://www.home-assistant.io/integrations/fan/ |
| Media Player | https://www.home-assistant.io/integrations/media_player/ |
| Lock | https://www.home-assistant.io/integrations/lock/ |
| Vacuum | https://www.home-assistant.io/integrations/vacuum/ |
| Sensor | https://www.home-assistant.io/integrations/sensor/ |
| Binary Sensor | https://www.home-assistant.io/integrations/binary_sensor/ |
| Scene | https://www.home-assistant.io/integrations/scene/ |
| REST API | https://developers.home-assistant.io/docs/api/rest/ |

---

## 14. 变更记录

| 日期 | 说明 |
|------|------|
| 2026-07-24 | 初版：连接、分 domain 状态/服务/能力/action 映射、历史与安全 |
