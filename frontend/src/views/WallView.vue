<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  fetchAnalyticsEnvironment,
  fetchAnalyticsHeatmap,
  fetchAnalyticsRanking,
  fetchAnalyticsRuntime,
  fetchAnalyticsSummary,
  fetchAnalyticsTypeMix,
} from '@/api'
import type {
  AnalyticsEnv,
  AnalyticsHeatmap,
  AnalyticsRanking,
  AnalyticsRuntime,
  AnalyticsSummary,
  AnalyticsTypeMix,
} from '@/api'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'

const router = useRouter()
const { toast } = useToast()
const range = ref<'7d' | '30d'>('7d')
const loading = ref(true)
const summary = ref<AnalyticsSummary | null>(null)
const runtime = ref<AnalyticsRuntime[]>([])
const ranking = ref<AnalyticsRanking[]>([])
const typeMix = ref<AnalyticsTypeMix[]>([])
const heatmap = ref<AnalyticsHeatmap[]>([])
const env = ref<AnalyticsEnv[]>([])

async function load() {
  loading.value = true
  try {
    const [s, rt, rk, tm, hm, ev] = await Promise.all([
      fetchAnalyticsSummary(range.value),
      fetchAnalyticsRuntime(range.value),
      fetchAnalyticsRanking(range.value),
      fetchAnalyticsTypeMix(range.value),
      fetchAnalyticsHeatmap(range.value),
      fetchAnalyticsEnvironment(range.value),
    ])
    summary.value = s.data
    runtime.value = rt.data.items || []
    ranking.value = rk.data.items || []
    typeMix.value = tm.data.items || []
    heatmap.value = hm.data.items || []
    env.value = ev.data.items || []
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '加载失败', 'err')
  } finally {
    loading.value = false
  }
}

// runtime per device summed by date → daily total hours
const runtimeDaily = computed(() => {
  const m = new Map<string, number>()
  for (const r of runtime.value) {
    m.set(r.date, (m.get(r.date) || 0) + r.hours)
  }
  return Array.from(m.entries())
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([t, h]) => ({ t, h }))
})

const maxDailyHours = computed(() =>
  Math.max(0.01, ...runtimeDaily.value.map((d) => d.h)),
)

function barHeight(h: number) {
  return { height: Math.max(2, (h / maxDailyHours.value) * 100) + '%' }
}
function barLabel(t: string) {
  const m = t.slice(5)
  return m.replace('-', '/')
}

const maxRankHours = computed(() => Math.max(1, ...ranking.value.map((r) => r.hours)))
function rankPct(h: number) {
  return { width: (h / maxRankHours.value) * 100 + '%' }
}

const totalTypeHours = computed(() => typeMix.value.reduce((a, b) => a + b.hours, 0) || 1)

// donut: typeMix → SVG arcs
const donutColors = ['#22d3ee', '#a78bfa', '#34d399', '#fbbf24', '#f472b6', '#60a5fa', '#fb7185']
const donutR = 42
const donutC = 2 * Math.PI * donutR
const donutSegments = computed(() => {
  const total = totalTypeHours.value
  let offset = 0
  return typeMix.value.map((t, i) => {
    const frac = t.hours / total
    const len = frac * donutC
    const seg = {
      color: donutColors[i % donutColors.length],
      dash: `${len} ${donutC - len}`,
      offset: -offset,
      domain: t.domain,
      hours: t.hours,
      pct: Math.round(frac * 100),
    }
    offset += len
    return seg
  })
})

// heatmap: rows = entities (cap 7), cols = 24 hours
const heatRows = computed(() => Array.from(new Set(heatmap.value.map((h) => h.entity_id))).slice(0, 7))
function heatValue(eid: string, hour: number) {
  return heatmap.value.find((h) => h.entity_id === eid && h.hour === hour)?.count || 0
}
const heatMax = computed(() => Math.max(1, ...heatmap.value.map((h) => h.count)))
function heatOpacity(eid: string, hour: number) {
  return heatValue(eid, hour) / heatMax.value
}
const heatGridStyle = computed(() => ({
  gridTemplateRows: `repeat(${heatRows.value.length || 1}, 14px)`,
}))

// env line chart as SVG path (first env series)
const envPoints = computed(() => {
  if (!env.value.length) return [] as { x: number; y: number; v: number | null }[]
  const series = env.value[0].series || []
  const vals = series.map((p) => p.value).filter((v): v is number => v != null)
  if (!vals.length) return []
  const min = Math.min(...vals)
  const max = Math.max(...vals)
  const span = max - min || 1
  const W = 320
  const H = 120
  const PAD = 8
  return series.map((p, i) => {
    const x = PAD + (i / Math.max(1, series.length - 1)) * (W - 2 * PAD)
    const v = p.value
    const y = v == null ? H - PAD : H - PAD - ((v - min) / span) * (H - 2 * PAD)
    return { x, y, v }
  })
})
const envLinePath = computed(() => {
  const pts = envPoints.value
  if (!pts.length) return ''
  return pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
})
const envAreaPath = computed(() => {
  const pts = envPoints.value
  if (!pts.length) return ''
  const top = pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
  return `${top} L${pts[pts.length - 1].x.toFixed(1)},120 L${pts[0].x.toFixed(1)},120 Z`
})
const envMeta = computed(() => (env.value.length ? env.value[0].entity_id : '—'))

// clock
const clock = ref('')
let clockTimer: number | undefined
function tick() {
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, '0')
  clock.value = `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}
const dateStr = computed(() => {
  const d = new Date()
  const w = ['日', '一', '二', '三', '四', '五', '六'][d.getDay()]
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} 周${w}`
})

function exit() {
  router.back()
}

onMounted(() => {
  load()
  tick()
  clockTimer = window.setInterval(tick, 1000)
})
onUnmounted(() => {
  if (clockTimer) window.clearInterval(clockTimer)
})
</script>

<template>
  <div class="view-wall">
    <div class="hud-bg" />
    <div class="hud-grid" />
    <div class="hud-scan" />
    <div class="hud-glow hud-glow-a" />
    <div class="hud-glow hud-glow-b" />

    <div class="hud-frame">
      <!-- header -->
      <header class="hud-header">
        <div class="hud-brand">
          <div class="hud-logo">
            <div class="hud-logo-ring" />
            <div class="hud-logo-core">SH</div>
          </div>
          <div>
            <span class="hud-title-main">智能家居中枢</span>
            <span class="hud-title-sub">USAGE ANALYTICS · HUD</span>
            <div class="hud-meta">
              <span class="hud-live"><i />LIVE</span>
              <span class="hud-sep">|</span>
              <span>{{ range === '7d' ? '近 7 天' : '近 30 天' }}</span>
              <span class="hud-sep">|</span>
              <span>设备运行分析</span>
            </div>
          </div>
        </div>
        <div class="hud-header-right">
          <div class="wall-range">
            <button
              v-for="r in (['7d','30d'] as const)"
              :key="r"
              type="button"
              class="wall-chip"
              :class="{ active: range === r }"
              @click="range = r; load()"
            >{{ r }}</button>
          </div>
          <div class="hud-clock-block">
            <div class="wall-clock">{{ clock }}</div>
            <div class="hud-date">{{ dateStr }}</div>
          </div>
          <button type="button" class="hud-exit" @click="exit">退出 ←</button>
        </div>
      </header>

      <div v-if="loading" class="hud-loading">正在加载数据…</div>
      <template v-else>
        <!-- KPI -->
        <div class="wall-kpi">
          <div class="wall-kpi-card" data-accent="cyan">
            <span class="kpi-corner" />
            <div class="k">激活次数</div>
            <div class="v">{{ summary?.activation_count ?? 0 }}</div>
            <div class="d">ACTIVATIONS</div>
          </div>
          <div class="wall-kpi-card" data-accent="mint">
            <span class="kpi-corner" />
            <div class="k">运行时长</div>
            <div class="v">{{ (summary?.runtime_hours ?? 0).toFixed(1) }}<span class="u">h</span></div>
            <div class="d">RUNTIME</div>
          </div>
          <div class="wall-kpi-card" data-accent="amber">
            <span class="kpi-corner" />
            <div class="k">用电量</div>
            <div class="v">
              <template v-if="summary?.energy_kwh != null">{{ summary.energy_kwh.toFixed(1) }}<span class="u">kWh</span></template>
              <template v-else>—</template>
            </div>
            <div class="d">ENERGY</div>
          </div>
          <div class="wall-kpi-card" data-accent="violet">
            <span class="kpi-corner" />
            <div class="k">平均温度</div>
            <div class="v">
              <template v-if="summary?.avg_temperature != null">{{ summary.avg_temperature.toFixed(1) }}<span class="u">°</span></template>
              <template v-else>—</template>
            </div>
            <div class="d">AVG TEMP</div>
          </div>
          <div class="wall-kpi-card" data-accent="pink">
            <span class="kpi-corner" />
            <div class="k">在线设备</div>
            <div class="v">{{ summary?.online_count ?? 0 }}</div>
            <div class="d">ONLINE</div>
          </div>
        </div>

        <!-- board -->
        <div class="wall-board">
          <!-- 每日运行时长 柱图 -->
          <section class="wall-panel wall-panel-lg">
            <div class="wall-panel-head">
              <h2><span class="hud-h-bar" />每日运行时长</h2>
              <span class="wall-tag">DAILY · h</span>
            </div>
            <div v-if="runtimeDaily.length" class="wall-bars">
              <div
                v-for="d in runtimeDaily"
                :key="d.t"
                class="bar"
                :style="barHeight(d.h)"
              >
                <span class="tip">{{ d.h.toFixed(1) }}h</span>
              </div>
            </div>
            <div v-else class="wall-empty">无数据</div>
            <div v-if="runtimeDaily.length" class="wall-axis">
              <span v-for="d in runtimeDaily" :key="d.t">{{ barLabel(d.t) }}</span>
            </div>
          </section>

          <!-- 设备排行 -->
          <section class="wall-panel">
            <div class="wall-panel-head">
              <h2><span class="hud-h-bar" />设备排行</h2>
              <span class="wall-tag">TOP {{ ranking.length }}</span>
            </div>
            <div v-if="ranking.length" class="wall-rank">
              <div v-for="(r, i) in ranking" :key="r.device_id" class="wall-rank-row">
                <span class="n">{{ String(i + 1).padStart(2, '0') }}</span>
                <div>
                  <div>{{ r.name }}</div>
                  <div class="bar-bg"><div class="bar-fg" :style="rankPct(r.hours)" /></div>
                </div>
                <span class="hrs">{{ r.hours.toFixed(1) }}h</span>
              </div>
            </div>
            <div v-else class="wall-empty">无数据</div>
          </section>

          <!-- 类型占比 donut -->
          <section class="wall-panel">
            <div class="wall-panel-head">
              <h2><span class="hud-h-bar" />类型占比</h2>
              <span class="wall-tag">MIX</span>
            </div>
            <div v-if="donutSegments.length" class="wall-donut-wrap">
              <div class="hud-ring-wrap">
                <svg class="wall-donut" viewBox="0 0 120 120">
                  <circle cx="60" cy="60" :r="donutR" fill="none" stroke="rgba(255,255,255,0.06)" stroke-width="14" />
                  <g class="ring-spin" transform="rotate(-90 60 60)">
                    <circle
                      v-for="(s, i) in donutSegments"
                      :key="i"
                      cx="60" cy="60" :r="donutR" fill="none"
                      :stroke="s.color" stroke-width="14"
                      :stroke-dasharray="s.dash"
                      :stroke-dashoffset="s.offset"
                      stroke-linecap="butt"
                    />
                  </g>
                </svg>
              </div>
              <div class="wall-donut-legend">
                <span v-for="(s, i) in donutSegments" :key="i">
                  <i :style="{ background: s.color, color: s.color }" />{{ s.domain }} · {{ s.pct }}%
                </span>
              </div>
            </div>
            <div v-else class="wall-empty">无数据</div>
          </section>

          <!-- 环境趋势 line -->
          <section class="wall-panel">
            <div class="wall-panel-head">
              <h2><span class="hud-h-bar" />环境趋势</h2>
              <span class="wall-tag">{{ envMeta }}</span>
            </div>
            <div v-if="envLinePath" class="wall-line-chart">
              <svg class="wall-svg" viewBox="0 0 320 120" preserveAspectRatio="none">
                <defs>
                  <linearGradient id="envArea" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stop-color="#22d3ee" stop-opacity="0.35" />
                    <stop offset="100%" stop-color="#22d3ee" stop-opacity="0" />
                  </linearGradient>
                </defs>
                <path :d="envAreaPath" fill="url(#envArea)" />
                <path :d="envLinePath" fill="none" stroke="#22d3ee" stroke-width="1.8" style="filter: drop-shadow(0 0 6px rgba(34,211,238,0.6))" />
              </svg>
              <div class="wall-legend"><i style="background:#22d3ee" />环境值</div>
            </div>
            <div v-else class="wall-empty">无温湿度传感器数据</div>
          </section>

          <!-- 活动热力 full width -->
          <section class="wall-panel wall-panel-full">
            <div class="wall-panel-head">
              <h2><span class="hud-h-bar" />活动热力（entity × 时段）</h2>
              <span class="wall-tag">24H</span>
            </div>
            <div v-if="heatRows.length" class="wall-heat-wrap">
              <div class="wall-heat-days">
                <span v-for="eid in heatRows" :key="eid" :title="eid">{{ eid.slice(-8) }}</span>
              </div>
              <div class="wall-heat-main">
                <div class="wall-heat" :style="heatGridStyle">
                  <template v-for="eid in heatRows" :key="eid">
                    <span
                      v-for="h in 24"
                      :key="eid + '-' + h"
                      :style="{ opacity: heatOpacity(eid, h - 1) }"
                      :title="`${eid} ${h - 1}:00 → ${heatValue(eid, h - 1)}`"
                    />
                  </template>
                </div>
                <div class="wall-heat-labels">
                  <span>00</span><span>06</span><span>12</span><span>18</span><span>24</span>
                </div>
              </div>
              <div class="wall-heat-scale">
                <span>高</span>
                <div class="wall-heat-scale-bar" />
                <span>低</span>
              </div>
            </div>
            <div v-else class="wall-empty">无数据</div>
          </section>
        </div>

        <!-- footer -->
        <footer class="hud-footer">
          <span class="hud-footer-pulse" />
          <span>智能家居中枢 · 数据来自 Home Assistant · 仅本地分析</span>
        </footer>
      </template>
    </div>
  </div>
</template>

<style scoped>
.hud-loading {
  position: relative;
  z-index: 1;
  text-align: center;
  padding: 3rem;
  color: rgba(34, 211, 238, 0.7);
  letter-spacing: 0.1em;
  font-family: ui-monospace, Menlo, Consolas, monospace;
}
.wall-empty {
  position: relative;
  z-index: 1;
  flex: 1;
  display: grid;
  place-items: center;
  color: rgba(224, 242, 254, 0.35);
  font-size: 0.8rem;
  letter-spacing: 0.06em;
}
</style>