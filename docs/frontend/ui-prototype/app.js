/* Smart Home UI prototype — mock data & interactions */
(function () {
  const DOMAIN_ICON = {
    light: "💡",
    switch: "🔌",
    sensor: "📡",
    climate: "❄️",
    cover: "🪟",
    binary_sensor: "🚪",
    scene: "🎬",
    script: "▶️",
    media_player: "🎵",
    button: "⏺",
    fan: "🌀",
    lock: "🔒",
  };

  /**
   * mock 设备 · attrs 对齐 HA 常见 attributes / services
   * light: turn_on/off/toggle + brightness/color_temp/hs_color/effect/transition
   * climate: turn_on/off/toggle + set_hvac_mode/set_temperature/set_fan_mode/set_swing_mode/set_preset_mode/set_humidity
   * cover: open/close/stop + set_cover_position/set_cover_tilt_position
   * fan: turn_on/off + set_percentage/set_preset_mode/oscillate/direction
   * media_player: turn_on/off + media_* + volume
   * lock / vacuum / scene / script / button ...
   */
  let myDevices = [
    {
      id: "d1",
      entity_id: "light.living_room",
      domain: "light",
      name: "客厅灯",
      room: "客厅",
      state: "on",
      favorite: true,
      available: true,
      capabilities: ["on_off", "brightness", "color_temp", "color_hs", "effect"],
      attrs: {
        brightness: 180,
        color_temp_kelvin: 3200,
        min_color_temp_kelvin: 2000,
        max_color_temp_kelvin: 6500,
        hs_color: [40, 60],
        effect: "none",
        effect_list: ["none", "colorloop", "candle", "breathe"],
        supported_color_modes: ["color_temp", "hs"],
        transition: 0.5,
      },
    },
    {
      id: "d2",
      entity_id: "switch.desk_plug",
      domain: "switch",
      name: "书桌插座",
      room: "卧室",
      state: "off",
      favorite: true,
      available: true,
      capabilities: ["on_off"],
      attrs: { device_class: "outlet", current_power_w: 0 },
    },
    {
      id: "d3",
      entity_id: "sensor.living_temp",
      domain: "sensor",
      name: "客厅温度",
      room: "客厅",
      state: "24.5",
      favorite: false,
      available: true,
      capabilities: ["numeric_sensor"],
      attrs: {
        unit_of_measurement: "°C",
        device_class: "temperature",
        state_class: "measurement",
        friendly_name: "客厅温度",
      },
    },
    {
      id: "d4",
      entity_id: "climate.bedroom_ac",
      domain: "climate",
      name: "卧室空调",
      room: "卧室",
      state: "cool",
      favorite: true,
      available: true,
      capabilities: ["on_off", "climate_hvac", "climate_temp", "climate_fan", "climate_swing", "climate_preset"],
      attrs: {
        hvac_mode: "cool",
        hvac_modes: ["off", "heat", "cool", "heat_cool", "auto", "dry", "fan_only"],
        hvac_action: "cooling",
        temperature: 26,
        current_temperature: 27.5,
        target_temp_high: 28,
        target_temp_low: 24,
        min_temp: 16,
        max_temp: 30,
        fan_mode: "auto",
        fan_modes: ["auto", "low", "medium", "high"],
        swing_mode: "off",
        swing_modes: ["off", "vertical", "horizontal", "both"],
        preset_mode: "none",
        preset_modes: ["none", "eco", "away", "boost", "sleep"],
        humidity: 50,
      },
    },
    {
      id: "d5",
      entity_id: "cover.balcony",
      domain: "cover",
      name: "阳台窗帘",
      room: "客厅",
      state: "open",
      favorite: false,
      available: true,
      capabilities: ["open_close", "position", "tilt"],
      attrs: {
        current_position: 80,
        current_tilt_position: 30,
        device_class: "curtain",
      },
    },
    {
      id: "d6",
      entity_id: "binary_sensor.door",
      domain: "binary_sensor",
      name: "入户门",
      room: "未分配",
      state: "off",
      favorite: false,
      available: false,
      capabilities: ["binary_sensor"],
      attrs: { device_class: "door" },
    },
    {
      id: "d7",
      entity_id: "fan.living_fan",
      domain: "fan",
      name: "客厅风扇",
      room: "客厅",
      state: "on",
      favorite: false,
      available: true,
      capabilities: ["on_off", "fan_speed", "fan_oscillate", "fan_direction"],
      attrs: {
        percentage: 60,
        percentage_step: 20,
        preset_mode: "normal",
        preset_modes: ["normal", "nature", "sleep"],
        oscillating: true,
        direction: "forward",
      },
    },
    {
      id: "d8",
      entity_id: "media_player.tv",
      domain: "media_player",
      name: "客厅电视",
      room: "客厅",
      state: "playing",
      favorite: false,
      available: true,
      capabilities: ["on_off", "media_play", "volume"],
      attrs: {
        media_title: "演示影片",
        media_artist: "Home Hub",
        volume_level: 0.42,
        is_volume_muted: false,
        source: "HDMI1",
        source_list: ["HDMI1", "HDMI2", "Netflix", "YouTube"],
      },
    },
    {
      id: "d9",
      entity_id: "lock.front_door",
      domain: "lock",
      name: "大门锁",
      room: "未分配",
      state: "locked",
      favorite: false,
      available: true,
      capabilities: ["lock"],
      attrs: { changed_by: "user" },
    },
    {
      id: "d10",
      entity_id: "vacuum.roborock",
      domain: "vacuum",
      name: "扫地机",
      room: "客厅",
      state: "docked",
      favorite: false,
      available: true,
      capabilities: ["vacuum"],
      attrs: {
        battery_level: 96,
        fan_speed: "balanced",
        fan_speed_list: ["quiet", "balanced", "turbo", "max"],
        status: "充电中",
      },
    },
    {
      id: "d11",
      entity_id: "scene.movie",
      domain: "scene",
      name: "观影模式",
      room: "客厅",
      state: "scening",
      favorite: true,
      available: true,
      capabilities: ["scene_activate"],
      attrs: {},
    },
  ];

  /** HA 发现池（含未添加） */
  let discoverPool = [
    ...myDevices.map((d) => ({
      entity_id: d.entity_id,
      domain: d.domain,
      name: d.name,
      state: d.state,
      capabilities: d.capabilities,
    })),
    {
      entity_id: "light.bedroom",
      domain: "light",
      name: "卧室主灯",
      state: "off",
      capabilities: ["on_off", "brightness", "color_temp", "color_hs", "effect"],
    },
    {
      entity_id: "switch.kitchen_hood",
      domain: "switch",
      name: "厨房插座",
      state: "off",
      capabilities: ["on_off"],
    },
    {
      entity_id: "sensor.humidity",
      domain: "sensor",
      name: "客厅湿度",
      state: "52",
      capabilities: ["numeric_sensor"],
    },
    {
      entity_id: "button.doorbell",
      domain: "button",
      name: "门铃触发",
      state: "unknown",
      capabilities: ["button"],
    },
  ];

  let haOnline = true;
  let currentPage = "overview";
  let detailId = null;
  let filterDomain = "";
  let filterFlag = "all";
  let discoverDomain = "";
  let selectedDiscover = new Set();

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];

  function toast(msg) {
    const wrap = $("#toasts");
    const el = document.createElement("div");
    el.className = "toast";
    el.textContent = msg;
    wrap.appendChild(el);
    setTimeout(() => el.remove(), 2400);
  }

  function isOn(d) {
    if (d.domain === "climate") return d.state !== "off";
    if (d.domain === "cover") return d.state === "open";
    if (d.domain === "lock") return d.state === "unlocked";
    if (d.domain === "media_player") return d.state !== "off" && d.state !== "idle";
    if (d.domain === "vacuum") return d.state === "cleaning" || d.state === "on";
    if (d.domain === "scene" || d.domain === "script") return false;
    if (d.domain === "sensor" || d.domain === "binary_sensor") return false;
    return d.state === "on";
  }

  function canControl(d) {
    return d.capabilities.some((c) =>
      [
        "on_off",
        "brightness",
        "color_temp",
        "color_hs",
        "open_close",
        "position",
        "climate_hvac",
        "climate_temp",
        "fan_speed",
        "media_play",
        "volume",
        "lock",
        "vacuum",
        "scene_activate",
        "script_run",
        "button",
      ].includes(c)
    );
  }

  function stateLabel(d) {
    if (!d.available) return "不可用";
    if (d.capabilities.includes("numeric_sensor")) {
      const u = d.attrs.unit_of_measurement || "";
      return `${d.state}${u}`;
    }
    if (d.domain === "cover") {
      if (d.state === "open") return `打开 ${d.attrs.current_position ?? ""}%`.trim();
      if (d.state === "closed") return "已关闭";
      return d.state;
    }
    if (d.domain === "climate") {
      const mode = hvacLabel(d.attrs.hvac_mode || d.state);
      return `${mode} · ${d.attrs.temperature ?? "—"}°`;
    }
    if (d.domain === "binary_sensor") return d.state === "on" ? "触发" : "正常";
    if (d.domain === "lock") return d.state === "locked" ? "已上锁" : "已解锁";
    if (d.domain === "media_player") {
      const m = { playing: "播放中", paused: "已暂停", idle: "待机", off: "关闭" };
      return m[d.state] || d.state;
    }
    if (d.domain === "vacuum") {
      const m = { cleaning: "清扫中", docked: "充电座", paused: "暂停", idle: "空闲" };
      return m[d.state] || d.state;
    }
    if (d.domain === "scene") return "场景";
    if (d.state === "on") return "开启";
    if (d.state === "off") return "关闭";
    return d.state;
  }

  function icon(domain) {
    return DOMAIN_ICON[domain] || "📟";
  }

  function renderDeviceCard(d, opts = {}) {
    const on = isOn(d);
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className =
      "device-card" +
      (on ? " on" : "") +
      (!d.available ? " unavailable" : "");
    btn.dataset.id = d.id;

    const showToggle = d.capabilities.includes("on_off") && d.available;
    btn.innerHTML = `
      <span class="fav ${d.favorite ? "active" : ""}" data-fav="${d.id}" title="收藏">${d.favorite ? "★" : "☆"}</span>
      <div class="ico">${icon(d.domain)}</div>
      <div class="name">${escapeHtml(d.name)}</div>
      <div class="meta">${escapeHtml(d.room)} · ${escapeHtml(d.domain)}${!canControl(d) ? ' · <span class="badge ro">仅监控</span>' : ""}</div>
      <div class="state-line">
        <span class="state-text ${on ? "on" : ""}">${escapeHtml(stateLabel(d))}</span>
        ${showToggle ? `<span class="toggle ${on ? "on" : ""}" data-toggle="${d.id}" role="switch" aria-checked="${on}"></span>` : ""}
      </div>
    `;

    btn.addEventListener("click", (e) => {
      const fav = e.target.closest("[data-fav]");
      const tog = e.target.closest("[data-toggle]");
      if (fav) {
        e.stopPropagation();
        toggleFavorite(d.id);
        return;
      }
      if (tog) {
        e.stopPropagation();
        togglePower(d.id);
        return;
      }
      openDetail(d.id);
    });

    return btn;
  }

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function updateStats() {
    $("#stat-total").textContent = String(myDevices.length);
    $("#stat-on").textContent = String(myDevices.filter((d) => isOn(d) && d.available).length);
    $("#stat-bad").textContent = String(myDevices.filter((d) => !d.available).length);
    const temp = myDevices.find((d) => d.entity_id === "sensor.living_temp");
    $("#stat-temp").textContent = temp ? `${temp.state}°` : "—";
  }

  function renderOverview() {
    const fav = $("#fav-grid");
    const all = $("#overview-grid");
    fav.innerHTML = "";
    all.innerHTML = "";
    const favs = myDevices.filter((d) => d.favorite);
    if (!favs.length) {
      fav.innerHTML = `<div class="empty" style="grid-column:1/-1">暂无收藏 · 在卡片点星标</div>`;
    } else {
      favs.forEach((d) => fav.appendChild(renderDeviceCard(d)));
    }
    myDevices.forEach((d) => all.appendChild(renderDeviceCard(d)));
    updateStats();
  }

  function renderDevices() {
    const q = ($("#filter-q").value || "").trim().toLowerCase();
    const room = $("#filter-room").value;
    const grid = $("#devices-grid");
    grid.innerHTML = "";

    let list = myDevices.slice();
    if (filterDomain) list = list.filter((d) => d.domain === filterDomain);
    if (room) list = list.filter((d) => d.room === room);
    if (filterFlag === "controllable") list = list.filter(canControl);
    if (filterFlag === "favorite") list = list.filter((d) => d.favorite);
    if (q) {
      list = list.filter(
        (d) =>
          d.name.toLowerCase().includes(q) ||
          d.entity_id.toLowerCase().includes(q)
      );
    }

    if (!list.length) {
      grid.innerHTML = `<div class="empty" style="grid-column:1/-1">没有匹配的设备</div>`;
      return;
    }
    list.forEach((d) => grid.appendChild(renderDeviceCard(d)));
  }

  function renderDomainChips() {
    const domains = [...new Set(myDevices.map((d) => d.domain))];
    const row = $("#filter-domain");
    row.innerHTML = "";
    const all = document.createElement("button");
    all.type = "button";
    all.className = "chip" + (!filterDomain ? " active" : "");
    all.textContent = "全部类型";
    all.addEventListener("click", () => {
      filterDomain = "";
      renderDomainChips();
      renderDevices();
    });
    row.appendChild(all);
    domains.forEach((dom) => {
      const b = document.createElement("button");
      b.type = "button";
      b.className = "chip" + (filterDomain === dom ? " active" : "");
      b.textContent = dom;
      b.addEventListener("click", () => {
        filterDomain = dom;
        renderDomainChips();
        renderDevices();
      });
      row.appendChild(b);
    });
  }

  function renderDiscover() {
    const q = ($("#discover-q").value || "").trim().toLowerCase();
    const onlyNew = $("#discover-only-new").checked;
    const added = new Set(myDevices.map((d) => d.entity_id));
    const list = $("#discover-list");
    list.innerHTML = "";

    let items = discoverPool.slice();
    if (discoverDomain) items = items.filter((x) => x.domain === discoverDomain);
    if (onlyNew) items = items.filter((x) => !added.has(x.entity_id));
    if (q) {
      items = items.filter(
        (x) =>
          x.name.toLowerCase().includes(q) ||
          x.entity_id.toLowerCase().includes(q)
      );
    }

    // domain chips
    const doms = [...new Set(discoverPool.map((x) => x.domain))];
    const drow = $("#discover-domain");
    drow.innerHTML = "";
    const makeChip = (label, val) => {
      const b = document.createElement("button");
      b.type = "button";
      b.className = "chip" + (discoverDomain === val ? " active" : "");
      b.textContent = label;
      b.addEventListener("click", () => {
        discoverDomain = val;
        renderDiscover();
      });
      drow.appendChild(b);
    };
    makeChip("全部", "");
    doms.forEach((d) => makeChip(d, d));

    if (!items.length) {
      list.innerHTML = `<div class="empty">没有可添加的实体 · 试试取消「仅未添加」或同步 HA</div>`;
      $("#btn-batch-add").disabled = true;
      return;
    }

    items.forEach((x) => {
      const already = added.has(x.entity_id);
      const row = document.createElement("div");
      row.className = "discover-item";
      const controllable = (x.capabilities || []).some((c) =>
        ["on_off", "open_close", "climate_hvac"].includes(c)
      );
      row.innerHTML = `
        <input type="checkbox" data-eid="${escapeHtml(x.entity_id)}" ${already ? "disabled" : ""} ${selectedDiscover.has(x.entity_id) ? "checked" : ""} />
        <div class="ico" style="width:2rem;height:2rem;border-radius:0.45rem;display:grid;place-items:center;background:hsl(var(--muted))">${icon(x.domain)}</div>
        <div class="info">
          <strong>${escapeHtml(x.name)}</strong>
          <span>${escapeHtml(x.entity_id)} · ${escapeHtml(x.state)}${!controllable ? " · 仅监控" : ""}</span>
        </div>
        ${
          already
            ? `<span class="badge">已添加</span>`
            : `<button type="button" class="btn btn-primary btn-sm" data-add="${escapeHtml(x.entity_id)}">添加</button>`
        }
      `;
      const cb = row.querySelector("input[type=checkbox]");
      if (cb && !already) {
        cb.addEventListener("change", () => {
          if (cb.checked) selectedDiscover.add(x.entity_id);
          else selectedDiscover.delete(x.entity_id);
          $("#btn-batch-add").disabled = selectedDiscover.size === 0;
        });
      }
      const addBtn = row.querySelector("[data-add]");
      if (addBtn) {
        addBtn.addEventListener("click", () => addEntities([x.entity_id]));
      }
      list.appendChild(row);
    });
    $("#btn-batch-add").disabled = selectedDiscover.size === 0;
  }

  function addEntities(entityIds) {
    let n = 0;
    entityIds.forEach((eid) => {
      if (myDevices.some((d) => d.entity_id === eid)) return;
      const src = discoverPool.find((x) => x.entity_id === eid);
      if (!src) return;
      myDevices.push({
        id: "d" + Date.now() + Math.random().toString(16).slice(2, 6),
        entity_id: src.entity_id,
        domain: src.domain,
        name: src.name,
        room: "未分配",
        state: src.state,
        favorite: false,
        available: true,
        capabilities: src.capabilities || ["raw_state"],
        attrs: {},
      });
      n++;
      selectedDiscover.delete(eid);
    });
    if (n) {
      toast(`已添加 ${n} 个设备`);
      refreshAll();
      renderDiscover();
    }
  }

  function toggleFavorite(id) {
    const d = myDevices.find((x) => x.id === id);
    if (!d) return;
    d.favorite = !d.favorite;
    toast(d.favorite ? `已收藏 ${d.name}` : `已取消收藏`);
    refreshAll();
  }

  function togglePower(id) {
    if (!haOnline) {
      toast("HA 已断开，无法控制");
      return;
    }
    const d = myDevices.find((x) => x.id === id);
    if (!d || !d.capabilities.includes("on_off")) return;
    d.state = d.state === "on" ? "off" : d.state === "off" ? "on" : d.state === "cool" ? "off" : "cool";
    if (d.domain === "climate" && d.state === "off") {
      /* ok */
    } else if (d.domain === "climate") {
      d.state = "cool";
    }
    toast(`${d.name} → ${stateLabel(d)}`);
    refreshAll();
    if (detailId === id) renderDetail();
  }

  function openDetail(id) {
    detailId = id;
    goPage("detail");
    renderDetail();
  }

  function guardHa() {
    if (!haOnline) {
      toast("HA 已断开，无法控制");
      return false;
    }
    return true;
  }

  function ensureOn(d) {
    if (d.state === "off") d.state = d.domain === "climate" ? "cool" : "on";
  }

  function domainLabel(domain) {
    const map = {
      light: "灯光",
      switch: "开关 / 插座",
      climate: "空调 / 温控",
      cover: "窗帘 / 遮盖",
      fan: "风扇",
      media_player: "媒体播放器",
      lock: "门锁",
      vacuum: "扫地机",
      scene: "场景",
      script: "脚本",
      button: "按钮",
      sensor: "传感器",
      binary_sensor: "二元传感",
    };
    return map[domain] || domain;
  }

  function hvacLabel(mode) {
    const m = {
      off: "关闭",
      heat: "制热",
      cool: "制冷",
      heat_cool: "自动热冷",
      auto: "自动",
      dry: "除湿",
      fan_only: "仅送风",
    };
    return m[mode] || mode;
  }

  function section(title, hint, bodyHtml) {
    return `
      <section class="dc-card">
        <div class="dc-card-head">
          <h3>${title}</h3>
          ${hint ? `<span class="dc-hint">${hint}</span>` : ""}
        </div>
        <div class="dc-card-body">${bodyHtml}</div>
      </section>`;
  }

  function powerButtons(d) {
    const on = isOn(d) || (d.domain === "media_player" && d.state !== "off");
    return `
      <div class="dc-power">
        <button type="button" class="dc-power-btn ${on ? "is-on" : ""}" data-act="on">
          <span class="dc-power-ico">⏻</span>
          <span>打开</span>
          <small>turn_on</small>
        </button>
        <button type="button" class="dc-power-btn" data-act="off">
          <span class="dc-power-ico">○</span>
          <span>关闭</span>
          <small>turn_off</small>
        </button>
        <button type="button" class="dc-power-btn" data-act="toggle">
          <span class="dc-power-ico">⇄</span>
          <span>切换</span>
          <small>toggle</small>
        </button>
      </div>`;
  }

  function sliderHtml(id, label, min, max, val, unit, service) {
    const pct = max > min ? ((val - min) / (max - min)) * 100 : 0;
    return `
      <div class="dc-slider" data-slider="${id}">
        <div class="dc-slider-top">
          <span class="dc-slider-label">${label}</span>
          <span class="dc-slider-val"><b data-val>${val}</b>${unit || ""}</span>
        </div>
        <input type="range" min="${min}" max="${max}" value="${val}" step="1"
          style="--pct:${pct}%" data-range="${id}" />
        ${service ? `<div class="dc-service">HA · ${service}</div>` : ""}
      </div>`;
  }

  function chipsHtml(name, options, current, labels) {
    return `
      <div class="dc-chips" data-chips="${name}">
        ${options
          .map((o) => {
            const lab = (labels && labels[o]) || o;
            return `<button type="button" class="dc-chip ${o === current ? "active" : ""}" data-val="${escapeHtml(o)}">${escapeHtml(lab)}</button>`;
          })
          .join("")}
      </div>`;
  }

  function bindSliders(root, d) {
    root.querySelectorAll("[data-range]").forEach((input) => {
      const wrap = input.closest(".dc-slider");
      const valEl = wrap.querySelector("[data-val]");
      const sync = () => {
        const min = Number(input.min);
        const max = Number(input.max);
        const v = Number(input.value);
        const pct = max > min ? ((v - min) / (max - min)) * 100 : 0;
        input.style.setProperty("--pct", pct + "%");
        valEl.textContent = v;
      };
      input.addEventListener("input", sync);
      input.addEventListener("change", () => {
        if (!guardHa()) return;
        const key = input.dataset.range;
        const v = Number(input.value);
        if (key === "brightness") {
          d.attrs.brightness = v;
          ensureOn(d);
          toast(`亮度 ${Math.round((v / 255) * 100)}%`);
        } else if (key === "color_temp") {
          d.attrs.color_temp_kelvin = v;
          ensureOn(d);
          toast(`色温 ${v} K`);
        } else if (key === "hue") {
          d.attrs.hs_color = [v, (d.attrs.hs_color && d.attrs.hs_color[1]) || 60];
          ensureOn(d);
          toast(`色相 ${v}°`);
        } else if (key === "sat") {
          d.attrs.hs_color = [(d.attrs.hs_color && d.attrs.hs_color[0]) || 40, v];
          ensureOn(d);
          toast(`饱和度 ${v}%`);
        } else if (key === "temperature") {
          d.attrs.temperature = v;
          ensureOn(d);
          toast(`目标温度 ${v}°`);
        } else if (key === "humidity") {
          d.attrs.humidity = v;
          toast(`目标湿度 ${v}%`);
        } else if (key === "position") {
          d.attrs.current_position = v;
          d.state = v <= 0 ? "closed" : v >= 100 ? "open" : "open";
          toast(`位置 ${v}%`);
        } else if (key === "tilt") {
          d.attrs.current_tilt_position = v;
          toast(`倾角 ${v}%`);
        } else if (key === "percentage") {
          d.attrs.percentage = v;
          ensureOn(d);
          toast(`风速 ${v}%`);
        } else if (key === "volume") {
          d.attrs.volume_level = v / 100;
          toast(`音量 ${v}%`);
        } else if (key === "transition") {
          d.attrs.transition = v / 10;
          toast(`过渡 ${v / 10}s`);
        }
        refreshAll();
        renderDetail();
      });
    });
  }

  function bindPower(root, d) {
    root.querySelectorAll("[data-act]").forEach((b) => {
      b.addEventListener("click", () => {
        if (!guardHa()) return;
        const act = b.dataset.act;
        if (act === "toggle") {
          if (d.domain === "climate") {
            d.state = d.state === "off" ? d.attrs.hvac_mode || "cool" : "off";
            if (d.state !== "off") d.attrs.hvac_mode = d.state;
          } else if (d.domain === "media_player") {
            d.state = d.state === "off" ? "idle" : "off";
          } else if (d.domain === "lock") {
            /* no */
          } else {
            d.state = d.state === "on" ? "off" : "on";
          }
        } else if (act === "on") {
          if (d.domain === "climate") {
            d.state = d.attrs.hvac_mode && d.attrs.hvac_mode !== "off" ? d.attrs.hvac_mode : "cool";
            d.attrs.hvac_mode = d.state;
            d.attrs.hvac_action = d.state === "heat" ? "heating" : "cooling";
          } else if (d.domain === "media_player") d.state = "idle";
          else if (d.domain === "cover") d.state = "open";
          else d.state = "on";
        } else if (act === "off") {
          d.state = "off";
          if (d.domain === "climate") {
            d.attrs.hvac_mode = "off";
            d.attrs.hvac_action = "idle";
          }
        }
        toast(`${d.name} · ${act}`);
        refreshAll();
        renderDetail();
      });
    });
  }

  function bindChips(root, d) {
    root.querySelectorAll("[data-chips]").forEach((group) => {
      const name = group.dataset.chips;
      group.querySelectorAll(".dc-chip").forEach((chip) => {
        chip.addEventListener("click", () => {
          if (!guardHa()) return;
          const v = chip.dataset.val;
          if (name === "hvac_mode") {
            d.attrs.hvac_mode = v;
            d.state = v;
            d.attrs.hvac_action =
              v === "off" ? "idle" : v === "heat" ? "heating" : v === "cool" ? "cooling" : "idle";
            toast(`模式 · ${hvacLabel(v)}`);
          } else if (name === "fan_mode") {
            d.attrs.fan_mode = v;
            toast(`风速模式 · ${v}`);
          } else if (name === "swing_mode") {
            d.attrs.swing_mode = v;
            toast(`摆风 · ${v}`);
          } else if (name === "preset_mode") {
            d.attrs.preset_mode = v;
            toast(`预设 · ${v}`);
          } else if (name === "effect") {
            d.attrs.effect = v;
            ensureOn(d);
            toast(`灯效 · ${v}`);
          } else if (name === "fan_preset") {
            d.attrs.preset_mode = v;
            toast(`风扇预设 · ${v}`);
          } else if (name === "direction") {
            d.attrs.direction = v;
            toast(`风向 · ${v === "forward" ? "正向" : "反向"}`);
          } else if (name === "source") {
            d.attrs.source = v;
            toast(`输入源 · ${v}`);
          } else if (name === "vac_fan") {
            d.attrs.fan_speed = v;
            toast(`清扫风力 · ${v}`);
          }
          refreshAll();
          renderDetail();
        });
      });
    });
  }

  function renderLightControls(d) {
    const bri = Number(d.attrs.brightness || 128);
    const ct = Number(d.attrs.color_temp_kelvin || 3200);
    const minK = Number(d.attrs.min_color_temp_kelvin || 2000);
    const maxK = Number(d.attrs.max_color_temp_kelvin || 6500);
    const hs = d.attrs.hs_color || [40, 60];
    const effects = d.attrs.effect_list || ["none"];
    const tr = Math.round(Number(d.attrs.transition || 0) * 10);
    return (
      section("电源", "light.turn_on / turn_off / toggle", powerButtons(d)) +
      section(
        "亮度",
        "brightness 0–255",
        sliderHtml("brightness", "亮度", 1, 255, bri, ` · ${Math.round((bri / 255) * 100)}%`, "light.turn_on")
      ) +
      section(
        "色温",
        "color_temp_kelvin",
        sliderHtml("color_temp", "色温", minK, maxK, ct, " K", "light.turn_on")
      ) +
      section(
        "彩色 HS",
        "hs_color [色相, 饱和度]",
        sliderHtml("hue", "色相 H", 0, 360, hs[0], "°", "light.turn_on") +
          sliderHtml("sat", "饱和度 S", 0, 100, hs[1], "%", "light.turn_on") +
          `<div class="dc-color-preview" style="--h:${hs[0]};--s:${hs[1]}%"></div>`
      ) +
      section("灯效", "effect", chipsHtml("effect", effects, d.attrs.effect || "none")) +
      section(
        "过渡动画",
        "transition 秒",
        sliderHtml("transition", "过渡", 0, 50, tr, " ×0.1s", "light.turn_on / toggle / turn_off")
      )
    );
  }

  function renderClimateControls(d) {
    const modes = d.attrs.hvac_modes || ["off", "heat", "cool"];
    const labels = {
      off: "关闭",
      heat: "制热",
      cool: "制冷",
      heat_cool: "热/冷",
      auto: "自动",
      dry: "除湿",
      fan_only: "送风",
    };
    const t = Number(d.attrs.temperature || 26);
    const minT = Number(d.attrs.min_temp || 16);
    const maxT = Number(d.attrs.max_temp || 30);
    const hum = Number(d.attrs.humidity || 50);
    return (
      section("电源", "climate.turn_on / turn_off / toggle", powerButtons(d)) +
      section(
        "运行模式",
        "set_hvac_mode",
        chipsHtml("hvac_mode", modes, d.attrs.hvac_mode || d.state, labels)
      ) +
      section(
        "目标温度",
        "set_temperature",
        `<div class="dc-temp-hero">
          <div class="dc-temp-now">当前 <b>${d.attrs.current_temperature ?? "—"}°</b></div>
          <div class="dc-temp-set">目标 <b data-temp-show>${t}°</b></div>
        </div>` +
          sliderHtml("temperature", "设定温度", minT, maxT, t, "°C", "climate.set_temperature")
      ) +
      section(
        "风速",
        "set_fan_mode",
        chipsHtml("fan_mode", d.attrs.fan_modes || ["auto", "low", "high"], d.attrs.fan_mode || "auto", {
          auto: "自动",
          low: "低",
          medium: "中",
          high: "高",
        })
      ) +
      section(
        "摆风",
        "set_swing_mode",
        chipsHtml("swing_mode", d.attrs.swing_modes || ["off", "vertical"], d.attrs.swing_mode || "off", {
          off: "关闭",
          vertical: "上下",
          horizontal: "左右",
          both: "双向",
        })
      ) +
      section(
        "预设",
        "set_preset_mode",
        chipsHtml("preset_mode", d.attrs.preset_modes || ["none"], d.attrs.preset_mode || "none", {
          none: "无",
          eco: "节能",
          away: "离家",
          boost: "强力",
          sleep: "睡眠",
        })
      ) +
      section(
        "目标湿度",
        "set_humidity（部分设备）",
        sliderHtml("humidity", "湿度", 30, 80, hum, "%", "climate.set_humidity")
      )
    );
  }

  function renderCoverControls(d) {
    const pos = Number(d.attrs.current_position ?? 50);
    const tilt = Number(d.attrs.current_tilt_position ?? 0);
    return (
      section(
        "开合",
        "open_cover / close_cover / stop_cover",
        `<div class="dc-power">
          <button type="button" class="dc-power-btn" data-cover="open"><span class="dc-power-ico">↑</span><span>打开</span><small>open_cover</small></button>
          <button type="button" class="dc-power-btn" data-cover="close"><span class="dc-power-ico">↓</span><span>关闭</span><small>close_cover</small></button>
          <button type="button" class="dc-power-btn" data-cover="stop"><span class="dc-power-ico">■</span><span>停止</span><small>stop_cover</small></button>
        </div>`
      ) +
      section(
        "位置",
        "set_cover_position 0–100",
        sliderHtml("position", "开合位置", 0, 100, pos, "%", "cover.set_cover_position")
      ) +
      section(
        "倾角",
        "set_cover_tilt_position",
        sliderHtml("tilt", "叶片倾角", 0, 100, tilt, "%", "cover.set_cover_tilt_position")
      )
    );
  }

  function renderFanControls(d) {
    const pct = Number(d.attrs.percentage || 50);
    return (
      section("电源", "fan.turn_on / turn_off", powerButtons(d)) +
      section(
        "风速百分比",
        "set_percentage",
        sliderHtml("percentage", "风速", 0, 100, pct, "%", "fan.set_percentage")
      ) +
      section(
        "预设",
        "set_preset_mode",
        chipsHtml("fan_preset", d.attrs.preset_modes || ["normal"], d.attrs.preset_mode || "normal", {
          normal: "普通",
          nature: "自然风",
          sleep: "睡眠",
        })
      ) +
      section(
        "摇头 / 方向",
        "oscillate · set_direction",
        `<div class="dc-toggle-row">
          <span>摇头 oscillating</span>
          <button type="button" class="toggle ${d.attrs.oscillating ? "on" : ""}" data-oscillate></button>
        </div>` +
          chipsHtml("direction", ["forward", "reverse"], d.attrs.direction || "forward", {
            forward: "正向",
            reverse: "反向",
          })
      )
    );
  }

  function renderMediaControls(d) {
    const vol = Math.round(Number(d.attrs.volume_level || 0) * 100);
    return (
      section("电源", "media_player.turn_on / turn_off", powerButtons(d)) +
      section(
        "播放控制",
        "media_play / pause / stop / next / previous",
        `<div class="dc-media">
          <div class="dc-media-art">${icon("media_player")}</div>
          <div class="dc-media-info">
            <strong>${escapeHtml(d.attrs.media_title || "未播放")}</strong>
            <span>${escapeHtml(d.attrs.media_artist || "—")}</span>
          </div>
        </div>
        <div class="dc-media-btns">
          <button type="button" class="btn btn-outline" data-media="prev">⏮</button>
          <button type="button" class="btn btn-primary" data-media="play">▶ 播放</button>
          <button type="button" class="btn btn-outline" data-media="pause">⏸ 暂停</button>
          <button type="button" class="btn btn-outline" data-media="stop">⏹</button>
          <button type="button" class="btn btn-outline" data-media="next">⏭</button>
        </div>`
      ) +
      section(
        "音量",
        "volume_set · volume_mute",
        sliderHtml("volume", "音量", 0, 100, vol, "%", "media_player.volume_set") +
          `<div class="dc-toggle-row">
            <span>静音</span>
            <button type="button" class="toggle ${d.attrs.is_volume_muted ? "on" : ""}" data-mute></button>
          </div>`
      ) +
      section(
        "输入源",
        "select_source",
        chipsHtml("source", d.attrs.source_list || [], d.attrs.source || "")
      )
    );
  }

  function renderLockControls(d) {
    const locked = d.state === "locked";
    return section(
      "锁控",
      "lock.lock / lock.unlock · 远程开锁请谨慎",
      `<div class="dc-lock ${locked ? "locked" : "unlocked"}">
        <div class="dc-lock-ico">${locked ? "🔒" : "🔓"}</div>
        <div class="dc-lock-state">${locked ? "已上锁" : "已解锁"}</div>
      </div>
      <div class="dc-power">
        <button type="button" class="dc-power-btn danger" data-lock="unlock"><span>解锁</span><small>lock.unlock</small></button>
        <button type="button" class="dc-power-btn is-on" data-lock="lock"><span>上锁</span><small>lock.lock</small></button>
      </div>`
    );
  }

  function renderVacuumControls(d) {
    return (
      section(
        "清扫",
        "start / pause / stop / return_to_base",
        `<div class="dc-vac-stat">
          <div><span>电量</span><b>${d.attrs.battery_level ?? "—"}%</b></div>
          <div><span>状态</span><b>${escapeHtml(d.attrs.status || d.state)}</b></div>
        </div>
        <div class="dc-power">
          <button type="button" class="dc-power-btn is-on" data-vac="start"><span>开始</span><small>start</small></button>
          <button type="button" class="dc-power-btn" data-vac="pause"><span>暂停</span><small>pause</small></button>
          <button type="button" class="dc-power-btn" data-vac="stop"><span>停止</span><small>stop</small></button>
          <button type="button" class="dc-power-btn" data-vac="dock"><span>回充</span><small>return_to_base</small></button>
        </div>`
      ) +
      section(
        "风力",
        "set_fan_speed",
        chipsHtml("vac_fan", d.attrs.fan_speed_list || ["balanced"], d.attrs.fan_speed || "balanced", {
          quiet: "安静",
          balanced: "均衡",
          turbo: "强力",
          max: "最大",
        })
      )
    );
  }

  function renderSceneControls(d) {
    return section(
      "场景",
      "scene.turn_on",
      `<p class="dc-readonly-tip">一键执行 HA 中已配置的场景，本系统不编辑场景步骤。</p>
      <button type="button" class="btn btn-primary btn-lg btn-block" data-scene-run>▶ 激活场景</button>`
    );
  }

  function renderSensorPanel(d) {
    const unit = d.attrs.unit_of_measurement || "";
    return section(
      "只读监测",
      "无控制 service · 仅展示 state",
      `<div class="dc-sensor-hero">
        <div class="dc-sensor-val">${escapeHtml(d.state)}<small>${escapeHtml(unit)}</small></div>
        <div class="dc-sensor-meta">${escapeHtml(d.attrs.device_class || d.domain)} · ${escapeHtml(d.attrs.state_class || "—")}</div>
      </div>
      <p class="dc-readonly-tip">传感器 / 二元传感不可远程操控，可在「历史」查看曲线。</p>`
    );
  }

  function renderDetail() {
    const root = $("#detail-root");
    if (!root) return;
    const d = myDevices.find((x) => x.id === detailId);
    if (!d) {
      goPage("devices");
      return;
    }

    const on = isOn(d) || (d.domain !== "sensor" && d.domain !== "binary_sensor" && d.state !== "off" && d.state !== "closed" && d.state !== "locked" && d.state !== "docked");
    let controls = "";
    if (!d.available) {
      controls = section("不可用", "", `<p class="dc-readonly-tip">实体 unavailable · 请检查 HA 与设备在线状态。</p>`);
    } else if (d.domain === "light") controls = renderLightControls(d);
    else if (d.domain === "switch" || d.domain === "input_boolean")
      controls = section("电源", "switch.turn_on / turn_off / toggle", powerButtons(d));
    else if (d.domain === "climate") controls = renderClimateControls(d);
    else if (d.domain === "cover") controls = renderCoverControls(d);
    else if (d.domain === "fan") controls = renderFanControls(d);
    else if (d.domain === "media_player") controls = renderMediaControls(d);
    else if (d.domain === "lock") controls = renderLockControls(d);
    else if (d.domain === "vacuum") controls = renderVacuumControls(d);
    else if (d.domain === "scene" || d.domain === "script") controls = renderSceneControls(d);
    else if (d.domain === "button" || d.domain === "input_button")
      controls = section(
        "按钮",
        "button.press",
        `<p class="dc-readonly-tip">瞬时动作，无持续状态。</p>
         <button type="button" class="btn btn-primary btn-lg btn-block" data-btn-press>按下 press</button>`
      );
    else if (d.domain === "sensor" || d.domain === "binary_sensor") controls = renderSensorPanel(d);
    else
      controls = section(
        "通用",
        "raw / 未完整映射",
        powerButtons(d) + `<p class="dc-readonly-tip">该 domain 以 HA 能力为准，后续按注册表扩展专用控件。</p>`
      );

    const attrRows = Object.entries(d.attrs || {})
      .map(([k, v]) => {
        const val = Array.isArray(v) ? v.join(", ") : typeof v === "object" ? JSON.stringify(v) : String(v);
        return `<div><span>${escapeHtml(k)}</span><span>${escapeHtml(val)}</span></div>`;
      })
      .join("");

    root.innerHTML = `
      <div class="dc-shell">
        <div class="dc-hero ${on ? "is-on" : ""} ${!d.available ? "is-down" : ""}">
          <div class="dc-hero-bg"></div>
          <div class="dc-hero-top">
            <button type="button" class="btn btn-ghost btn-sm" data-go-devices>← 返回</button>
            <div class="dc-hero-actions">
              <button type="button" class="btn btn-outline btn-sm" id="btn-detail-fav">${d.favorite ? "★ 已收藏" : "☆ 收藏"}</button>
              <button type="button" class="btn btn-danger btn-sm" id="btn-detail-remove">移除</button>
            </div>
          </div>
          <div class="dc-hero-main">
            <div class="dc-hero-ico">${icon(d.domain)}</div>
            <div>
              <div class="dc-hero-type">${escapeHtml(domainLabel(d.domain))}</div>
              <h1 class="dc-hero-name">${escapeHtml(d.name)}</h1>
              <div class="dc-hero-sub">${escapeHtml(d.room)} · <code>${escapeHtml(d.entity_id)}</code></div>
            </div>
            <div class="dc-hero-state">
              <span class="dc-state-pill ${d.available ? (on ? "on" : "off") : "bad"}">${escapeHtml(stateLabel(d))}</span>
              ${d.domain === "climate" && d.attrs.hvac_action ? `<span class="dc-state-sub">动作 ${escapeHtml(d.attrs.hvac_action)}</span>` : ""}
            </div>
          </div>
        </div>

        <div class="dc-grid">
          <div class="dc-main">
            <div class="dc-section-title">操控台 <span>按 HA domain 能力渲染</span></div>
            ${controls}
          </div>
          <aside class="dc-side">
            <section class="dc-card">
              <div class="dc-card-head"><h3>设备信息</h3></div>
              <div class="dc-card-body">
                <div class="kv">
                  <div><span>类型</span><span>${escapeHtml(domainLabel(d.domain))}</span></div>
                  <div><span>Domain</span><span>${escapeHtml(d.domain)}</span></div>
                  <div><span>房间</span><span>${escapeHtml(d.room)}</span></div>
                  <div><span>可用</span><span>${d.available ? "是" : "否"}</span></div>
                  <div><span>Entity</span><span class="mono">${escapeHtml(d.entity_id)}</span></div>
                </div>
              </div>
            </section>
            <section class="dc-card">
              <div class="dc-card-head"><h3>Attributes</h3><span class="dc-hint">HA 原始属性</span></div>
              <div class="dc-card-body"><div class="kv kv-attrs">${attrRows || "<div class='muted'>无</div>"}</div></div>
            </section>
            <section class="dc-card">
              <div class="dc-card-head"><h3>能力标签</h3></div>
              <div class="dc-card-body">
                <div class="dc-cap-list">${(d.capabilities || []).map((c) => `<span class="dc-cap">${escapeHtml(c)}</span>`).join("")}</div>
              </div>
            </section>
          </aside>
        </div>
      </div>
    `;

    root.querySelector("[data-go-devices]")?.addEventListener("click", () => goPage("devices"));
    $("#btn-detail-fav")?.addEventListener("click", () => {
      toggleFavorite(d.id);
      renderDetail();
    });
    $("#btn-detail-remove")?.addEventListener("click", () => {
      if (!confirm(`从本系统移除「${d.name}」？\n不会删除 HA 中的设备。`)) return;
      myDevices = myDevices.filter((x) => x.id !== detailId);
      toast("已移除");
      detailId = null;
      goPage("devices");
      refreshAll();
    });

    bindPower(root, d);
    bindSliders(root, d);
    bindChips(root, d);

    root.querySelectorAll("[data-cover]").forEach((b) => {
      b.addEventListener("click", () => {
        if (!guardHa()) return;
        const a = b.dataset.cover;
        if (a === "open") {
          d.state = "open";
          d.attrs.current_position = 100;
        }
        if (a === "close") {
          d.state = "closed";
          d.attrs.current_position = 0;
        }
        if (a === "stop") toast("已发送 stop_cover");
        else toast(`${d.name} · ${a}`);
        refreshAll();
        renderDetail();
      });
    });

    root.querySelector("[data-oscillate]")?.addEventListener("click", () => {
      if (!guardHa()) return;
      d.attrs.oscillating = !d.attrs.oscillating;
      toast(`摇头 ${d.attrs.oscillating ? "开" : "关"}`);
      renderDetail();
    });

    root.querySelectorAll("[data-media]").forEach((b) => {
      b.addEventListener("click", () => {
        if (!guardHa()) return;
        const a = b.dataset.media;
        if (a === "play") d.state = "playing";
        if (a === "pause") d.state = "paused";
        if (a === "stop") d.state = "idle";
        toast(`media_player · ${a}`);
        refreshAll();
        renderDetail();
      });
    });

    root.querySelector("[data-mute]")?.addEventListener("click", () => {
      if (!guardHa()) return;
      d.attrs.is_volume_muted = !d.attrs.is_volume_muted;
      toast(d.attrs.is_volume_muted ? "已静音" : "取消静音");
      renderDetail();
    });

    root.querySelectorAll("[data-lock]").forEach((b) => {
      b.addEventListener("click", () => {
        if (!guardHa()) return;
        d.state = b.dataset.lock === "lock" ? "locked" : "unlocked";
        toast(d.state === "locked" ? "已上锁" : "已解锁");
        refreshAll();
        renderDetail();
      });
    });

    root.querySelectorAll("[data-vac]").forEach((b) => {
      b.addEventListener("click", () => {
        if (!guardHa()) return;
        const a = b.dataset.vac;
        if (a === "start") {
          d.state = "cleaning";
          d.attrs.status = "清扫中";
        }
        if (a === "pause") {
          d.state = "paused";
          d.attrs.status = "已暂停";
        }
        if (a === "stop") {
          d.state = "idle";
          d.attrs.status = "停止";
        }
        if (a === "dock") {
          d.state = "docked";
          d.attrs.status = "回充中";
        }
        toast(`vacuum · ${a}`);
        refreshAll();
        renderDetail();
      });
    });

    root.querySelector("[data-scene-run]")?.addEventListener("click", () => {
      if (!guardHa()) return;
      toast(`已激活场景「${d.name}」`);
    });

    root.querySelector("[data-btn-press]")?.addEventListener("click", () => {
      if (!guardHa()) return;
      toast(`button.press · ${d.name}`);
    });
  }

  function renderHistorySelect() {
    const sel = $("#history-device");
    sel.innerHTML = myDevices
      .map((d) => `<option value="${d.id}">${escapeHtml(d.name)}</option>`)
      .join("");
  }

  /** 分析大屏 mock 数据 */
  const wallData = {
    week: {
      count: 128,
      hours: 46.2,
      kwh: 18.4,
      temp: 24.1,
      bars: [5.2, 6.8, 4.1, 7.5, 8.2, 9.1, 5.3],
      axis: ["一", "二", "三", "四", "五", "六", "日"],
      rank: [
        { name: "客厅灯", hours: 16.4 },
        { name: "书桌插座", hours: 12.1 },
        { name: "卧室空调", hours: 9.8 },
        { name: "阳台窗帘", hours: 4.2 },
        { name: "厨房插座", hours: 3.7 },
      ],
      donut: [
        { name: "灯", pct: 48, color: "#22d3ee" },
        { name: "插座", pct: 28, color: "#34d399" },
        { name: "空调", pct: 16, color: "#c084fc" },
        { name: "其它", pct: 8, color: "rgba(224,242,254,0.25)" },
      ],
      /** 7 天 × 24 小时 */
      heatDays: ["周一", "周二", "周三", "周四", "周五", "周六", "周日"],
      heat: null,
    },
    month: {
      count: 512,
      hours: 186.5,
      kwh: 72.0,
      temp: 23.6,
      bars: [38, 42, 35, 48, 51, 55, 40],
      axis: ["第1周", "第2周", "第3周", "第4周", "第5周", "第6周", "第7周"],
      rank: [
        { name: "客厅灯", hours: 62.0 },
        { name: "卧室空调", hours: 48.5 },
        { name: "书桌插座", hours: 41.2 },
        { name: "厨房插座", hours: 18.0 },
        { name: "阳台窗帘", hours: 12.3 },
      ],
      donut: [
        { name: "灯", pct: 42, color: "#22d3ee" },
        { name: "空调", pct: 30, color: "#c084fc" },
        { name: "插座", pct: 20, color: "#34d399" },
        { name: "其它", pct: 8, color: "rgba(224,242,254,0.25)" },
      ],
      heatDays: ["第1周", "第2周", "第3周", "第4周", "第5周", "第6周", "第7周"],
      heat: null,
    },
  };

  function buildHeatMatrix(seedBoost) {
    const rows = [];
    for (let d = 0; d < 7; d++) {
      const row = [];
      for (let h = 0; h < 24; h++) {
        let v = 0;
        if (h >= 7 && h <= 9) v = 3 + (d % 3);
        else if (h >= 12 && h <= 13) v = 2 + (d % 2);
        else if (h >= 18 && h <= 22) v = 5 + (d % 4) + seedBoost;
        else if (h >= 0 && h <= 5) v = d > 4 ? 1 : 0;
        else v = (h + d) % 4 === 0 ? 2 : 1;
        if (d === 5 || d === 6) v = Math.min(9, v + 1);
        row.push(v);
      }
      rows.push(row);
    }
    return rows;
  }

  wallData.week.heat = buildHeatMatrix(0);
  wallData.month.heat = buildHeatMatrix(1);

  let wallRange = "week";

  function renderWall() {
    const data = wallData[wallRange] || wallData.week;
    const maxBar = Math.max(...data.bars, 1);
    const maxRank = Math.max(...data.rank.map((r) => r.hours), 1);

    $("#wall-kpi-count").textContent = String(data.count);
    $("#wall-kpi-hours").innerHTML = data.hours + '<span class="u">小时</span>';
    $("#wall-kpi-kwh").innerHTML = data.kwh + '<span class="u">度</span>';
    $("#wall-kpi-temp").innerHTML = data.temp + '<span class="u">°</span>';

    const onN = myDevices.filter((d) => isOn(d) && d.available).length;
    const online = myDevices.filter((d) => d.available).length;
    $("#wall-kpi-live").textContent = online + " / " + onN;
    $("#wall-kpi-ha").textContent = haOnline ? "HA 已连接" : "HA 已断开";

    const bars = $("#wall-bars");
    bars.innerHTML = "";
    data.bars.forEach((h) => {
      const el = document.createElement("div");
      el.className = "bar";
      el.style.height = Math.max(8, (h / maxBar) * 100) + "%";
      el.innerHTML = `<span class="tip">${h}</span>`;
      bars.appendChild(el);
    });
    const axis = $("#wall-bars-axis");
    axis.innerHTML = data.axis.map((a) => `<span>${a}</span>`).join("");

    const rank = $("#wall-rank");
    rank.innerHTML = "";
    data.rank.forEach((r, i) => {
      const row = document.createElement("div");
      row.className = "wall-rank-row";
      row.innerHTML = `
        <span class="n">${i + 1}</span>
        <div>
          <div>${escapeHtml(r.name)}</div>
          <div class="bar-bg"><div class="bar-fg" style="width:${(r.hours / maxRank) * 100}%"></div></div>
        </div>
        <span class="hrs">${r.hours} 小时</span>
      `;
      rank.appendChild(row);
    });

    const legend = $("#wall-donut-legend");
    legend.innerHTML = data.donut
      .map(
        (x) =>
          `<span><i style="background:${x.color}"></i>${escapeHtml(x.name)} ${x.pct}%</span>`
      )
      .join("");

    const days = data.heatDays || ["周一", "周二", "周三", "周四", "周五", "周六", "周日"];
    const matrix = data.heat || buildHeatMatrix(0);
    const flat = matrix.flat();
    const maxH = Math.max(...flat, 1);

    const daysEl = $("#wall-heat-days");
    if (daysEl) {
      daysEl.innerHTML = days.map((d) => `<span>${escapeHtml(d)}</span>`).join("");
    }

    const heat = $("#wall-heat");
    heat.innerHTML = matrix
      .map((row, di) =>
        row
          .map((v, hi) => {
            const a = 0.1 + (v / maxH) * 0.9;
            return `<span style="opacity:${a.toFixed(2)}" title="${days[di]} ${hi}点 · 活跃度 ${v}"></span>`;
          })
          .join("")
      )
      .join("");
  }

  function tickClock() {
    if ($("#view-wall").classList.contains("hidden")) return;
    const n = new Date();
    const el = $("#wall-clock");
    if (el) el.textContent = n.toLocaleTimeString("zh-CN", { hour12: false });
    const d = $("#wall-date");
    if (d) {
      d.textContent = n.toLocaleDateString("zh-CN", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        weekday: "short",
      });
    }
  }

  function spawnHudParticles() {
    const host = $("#hud-particles");
    if (!host || host.dataset.ready) return;
    host.dataset.ready = "1";
    for (let i = 0; i < 28; i++) {
      const p = document.createElement("span");
      p.className = "hud-particle";
      p.style.left = Math.random() * 100 + "%";
      p.style.animationDuration = 8 + Math.random() * 14 + "s";
      p.style.animationDelay = Math.random() * 10 + "s";
      p.style.width = p.style.height = 1 + Math.random() * 2 + "px";
      if (Math.random() > 0.6) p.style.background = "#a78bfa";
      if (Math.random() > 0.85) p.style.background = "#34d399";
      host.appendChild(p);
    }
  }

  function setHaPill() {
    const pill = $("#ha-pill");
    const text = $("#ha-pill-text");
    if (haOnline) {
      pill.classList.remove("offline");
      text.textContent = "HA 已连接";
      $("#set-ha-status").value = "已连接 · 模拟";
    } else {
      pill.classList.add("offline");
      text.textContent = "HA 已断开";
      $("#set-ha-status").value = "已断开 · 模拟";
    }
  }

  function refreshAll() {
    renderOverview();
    renderDomainChips();
    renderDevices();
    renderHistorySelect();
    if (currentPage === "detail") renderDetail();
    if (!$("#view-wall").classList.contains("hidden")) renderWall();
  }

  function goPage(name) {
    currentPage = name;
    $$(".page").forEach((p) => p.classList.remove("active"));
    const page = $("#page-" + name);
    if (page) page.classList.add("active");

    $$(".nav-item, .bottom-nav button").forEach((b) => {
      b.classList.toggle("active", b.dataset.page === name);
    });
    // detail / history not in bottom: highlight devices
    if (name === "detail") {
      $$(".nav-item, .bottom-nav button").forEach((b) => {
        b.classList.toggle("active", b.dataset.page === "devices");
      });
    }
    if (name === "history") {
      $$(".nav-item").forEach((b) => {
        b.classList.toggle("active", b.dataset.page === "history");
      });
    }

    if (name === "devices") renderDevices();
    if (name === "add") renderDiscover();
    if (name === "overview") renderOverview();
    if (name === "history") renderHistorySelect();
  }

  function showApp() {
    $("#view-login").classList.add("hidden");
    $("#view-app").classList.remove("hidden");
    refreshAll();
  }

  function showLogin() {
    $("#view-app").classList.add("hidden");
    $("#view-wall").classList.add("hidden");
    $("#view-login").classList.remove("hidden");
  }

  function setTheme(t) {
    document.documentElement.setAttribute("data-theme", t);
    try {
      localStorage.setItem("sh-theme", t);
    } catch (e) {}
    const appBtn = $("#btn-theme-app");
    if (appBtn) appBtn.textContent = t === "dark" ? "☾" : "☀";
  }

  function toggleTheme() {
    const cur = document.documentElement.getAttribute("data-theme") || "dark";
    setTheme(cur === "dark" ? "light" : "dark");
  }

  // events
  $("#btn-sso").addEventListener("click", () => {
    toast("模拟 SSO 成功");
    showApp();
  });
  $("#btn-logout").addEventListener("click", () => {
    toast("已登出本地会话");
    showLogin();
  });
  $("#btn-portal").addEventListener("click", () => {
    toast("原型：将跳转认证中心 /select-system");
  });
  $("#btn-theme").addEventListener("click", toggleTheme);
  $("#btn-theme-app").addEventListener("click", toggleTheme);

  $$("#sidebar .nav-item, #bottom-nav button").forEach((b) => {
    b.addEventListener("click", () => goPage(b.dataset.page));
  });
  $$("[data-go]").forEach((b) => {
    b.addEventListener("click", () => goPage(b.dataset.go));
  });

  $("#filter-q").addEventListener("input", renderDevices);
  $("#filter-room").addEventListener("change", renderDevices);
  $$("[data-flag]").forEach((b) => {
    b.addEventListener("click", () => {
      filterFlag = b.dataset.flag;
      $$("[data-flag]").forEach((x) => x.classList.toggle("active", x === b));
      renderDevices();
    });
  });

  $("#discover-q").addEventListener("input", renderDiscover);
  $("#discover-only-new").addEventListener("change", renderDiscover);
  $("#btn-sync-ha").addEventListener("click", () => {
    toast("已同步 HA 实体（模拟）");
    renderDiscover();
  });
  $("#btn-batch-add").addEventListener("click", () => {
    addEntities([...selectedDiscover]);
  });

  $("#btn-refresh").addEventListener("click", () => {
    toast("已刷新状态");
    refreshAll();
  });

  $("#btn-toggle-ha").addEventListener("click", () => {
    haOnline = !haOnline;
    setHaPill();
    toast(haOnline ? "HA 已连接" : "HA 已断开");
  });

  $("#btn-wall").addEventListener("click", () => {
    $("#view-app").classList.add("hidden");
    $("#view-wall").classList.remove("hidden");
    spawnHudParticles();
    renderWall();
    tickClock();
  });
  $("#btn-wall-exit").addEventListener("click", () => {
    $("#view-wall").classList.add("hidden");
    $("#view-app").classList.remove("hidden");
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !$("#view-wall").classList.contains("hidden")) {
      $("#btn-wall-exit").click();
    }
  });
  $$("[data-wall-range]").forEach((b) => {
    b.addEventListener("click", () => {
      wallRange = b.dataset.wallRange;
      $$("[data-wall-range]").forEach((x) => x.classList.toggle("active", x === b));
      renderWall();
    });
  });

  setInterval(tickClock, 1000);
  setHaPill();
  setTheme(document.documentElement.getAttribute("data-theme") || "dark");
})();
