<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
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
import HistoryChart from '@/components/HistoryChart.vue'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'

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

// runtime per device summed by date → daily total hours line
const runtimeDaily = computed(() => {
  const m = new Map<string, number>()
  for (const r of runtime.value) {
    m.set(r.date, (m.get(r.date) || 0) + r.hours)
  }
  return Array.from(m.entries())
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([t, h]) => ({ t, state: String(h), num: h }))
})

const maxRankHours = () => Math.max(1, ...ranking.value.map((r) => r.hours))

function barWidth(h: number) {
  const pct = (h / maxRankHours()) * 100
  return { width: pct + '%' }
}

const totalTypeHours = () => typeMix.value.reduce((a, b) => a + b.hours, 0) || 1

// heatmap: matrix entity × hour
const heatEntities = () => Array.from(new Set(heatmap.value.map((h) => h.entity_id)))
const heatValue = (eid: string, hour: number) =>
  heatmap.value.find((h) => h.entity_id === eid && h.hour === hour)?.count || 0
const heatMax = () => Math.max(1, ...heatmap.value.map((h) => h.count))

function heatStyle(eid: string, hour: number) {
  return { opacity: heatValue(eid, hour) / heatMax() }
}

const envPoints = () => {
  if (!env.value.length) return []
  const series = env.value[0].series || []
  return series.map((p) => ({ t: p.date, state: String(p.value ?? ''), num: p.value ?? null }))
}

onMounted(load)
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <h1>使用分析</h1>
        <p>设备运行时长 · 类型占比 · 热力图 · 环境趋势</p>
      </div>
      <div class="page-actions">
        <button
          v-for="r in (['7d','30d'] as const)"
          :key="r"
          type="button"
          class="chip"
          :class="{ active: range === r }"
          @click="range = r"
        >{{ r }}</button>
        <button type="button" class="btn btn-outline btn-sm" :disabled="loading" @click="load">刷新</button>
      </div>
    </div>

    <div v-if="loading" class="muted">加载中…</div>
    <template v-else>
      <div class="stat-row">
        <div class="stat-card">
          <div class="label">激活次数</div>
          <div class="value">{{ summary?.activation_count ?? 0 }}</div>
          <div class="hint">ACTIVATIONS</div>
        </div>
        <div class="stat-card">
          <div class="label">运行时长</div>
          <div class="value">{{ (summary?.runtime_hours ?? 0).toFixed(1) }}h</div>
          <div class="hint">RUNTIME</div>
        </div>
        <div class="stat-card">
          <div class="label">用电量</div>
          <div class="value">{{ summary?.energy_kwh != null ? summary.energy_kwh.toFixed(1) + 'kWh' : '—' }}</div>
          <div class="hint">ENERGY</div>
        </div>
        <div class="stat-card">
          <div class="label">均温</div>
          <div class="value">{{ summary?.avg_temperature != null ? summary.avg_temperature.toFixed(1) + '°' : '—' }}</div>
          <div class="hint">AVG TEMP</div>
        </div>
        <div class="stat-card">
          <div class="label">在线设备</div>
          <div class="value">{{ summary?.online_count ?? 0 }}</div>
          <div class="hint">ONLINE</div>
        </div>
      </div>

      <div class="section-title">每日运行时长</div>
      <HistoryChart :points="runtimeDaily" kind="value" unit="小时" />

      <div class="section-title">设备排行</div>
      <div v-if="!ranking.length" class="muted">无数据</div>
      <div v-else class="bar-list">
        <div v-for="r in ranking" :key="r.device_id" class="bar-row">
          <span class="bar-name">{{ r.name }}</span>
          <div class="bar-track">
            <div class="bar-fill" :style="barWidth(r.hours)" />
          </div>
          <span class="bar-val">{{ r.hours.toFixed(1) }}h</span>
        </div>
      </div>

      <div class="section-title">类型占比</div>
      <div v-if="!typeMix.length" class="muted">无数据</div>
      <div v-else class="type-list">
        <div v-for="t in typeMix" :key="t.domain" class="type-row">
          <span class="type-domain">{{ t.domain }}</span>
          <span class="type-count">{{ t.count }} 台</span>
          <span class="type-hours">{{ (t.hours / totalTypeHours() * 100).toFixed(0) }}%</span>
        </div>
      </div>

      <div class="section-title">活动热力（entity × 时段）</div>
      <div v-if="!heatmap.length" class="muted">无数据</div>
      <div v-else class="heat-wrap">
        <div v-for="eid in heatEntities()" :key="eid" class="heat-row">
          <span class="heat-name">{{ eid }}</span>
          <div class="heat-cells">
            <span
              v-for="h in 24"
              :key="h"
              class="heat-cell"
              :style="heatStyle(eid, h - 1)"
              :title="`${eid} ${h - 1}:00 → ${heatValue(eid, h - 1)}`"
            />
          </div>
        </div>
      </div>

      <div class="section-title">环境趋势（{{ env[0]?.entity_id || '—' }}）</div>
      <HistoryChart v-if="envPoints().length" :points="envPoints()" kind="value" />
      <div v-else class="muted">无温湿度传感器数据</div>
    </template>
  </section>
</template>

<style scoped>
.muted {
  color: hsl(var(--muted-foreground));
  padding: 1rem 0;
}
.bar-list {
  display: grid;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
}
.bar-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}
.bar-name {
  flex: 0 0 8rem;
  font-size: 0.82rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bar-track {
  flex: 1;
  height: 0.7rem;
  background: hsl(var(--muted));
  border-radius: 999px;
  overflow: hidden;
}
.bar-fill {
  height: 100%;
  background: hsl(var(--primary));
  border-radius: 999px;
}
.bar-val {
  flex: 0 0 4rem;
  text-align: right;
  font-size: 0.8rem;
  font-variant-numeric: tabular-nums;
}
.type-list {
  display: grid;
  gap: 0.4rem;
  margin-bottom: 1.25rem;
}
.type-row {
  display: flex;
  gap: 1rem;
  padding: 0.35rem 0.5rem;
  border-bottom: 1px solid hsl(var(--border) / 0.5);
  font-size: 0.85rem;
}
.type-domain {
  flex: 1;
  font-weight: 500;
}
.type-count {
  color: hsl(var(--muted-foreground));
}
.heat-wrap {
  display: grid;
  gap: 0.3rem;
  margin-bottom: 1.25rem;
  overflow-x: auto;
}
.heat-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.heat-name {
  flex: 0 0 12rem;
  font-size: 0.72rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: ui-monospace, monospace;
}
.heat-cells {
  display: flex;
  gap: 2px;
}
.heat-cell {
  width: 1.1rem;
  height: 1.1rem;
  border-radius: 3px;
  background: hsl(var(--primary));
}
</style>