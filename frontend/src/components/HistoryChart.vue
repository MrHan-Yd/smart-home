<script setup lang="ts">
import { computed } from 'vue'
import type { HistoryPoint } from '@/api'

const props = defineProps<{
  points: HistoryPoint[]
  unit?: string
  kind: 'value' | 'onoff'
}>()

const W = 640
const H = 180
const PAD = 28

const numeric = computed(() => props.points.filter((p) => p.num !== null && Number.isFinite(p.num)))

const range = computed(() => {
  const ns = numeric.value.map((p) => p.num as number)
  if (!ns.length) return { min: 0, max: 1 }
  let min = Math.min(...ns)
  let max = Math.max(...ns)
  if (min === max) {
    min -= 1
    max += 1
  }
  const pad = (max - min) * 0.1
  return { min: min - pad, max: max + pad }
})

const tRange = computed(() => {
  const ts = props.points.map((p) => new Date(p.t).getTime()).filter((n) => Number.isFinite(n))
  if (!ts.length) return { min: 0, max: 1 }
  return { min: Math.min(...ts), max: Math.max(...ts) }
})

function x(t: string): number {
  const ms = new Date(t).getTime()
  const span = tRange.value.max - tRange.value.min || 1
  return PAD + ((ms - tRange.value.min) / span) * (W - 2 * PAD)
}

function yVal(num: number): number {
  const span = range.value.max - range.value.min || 1
  return H - PAD - ((num - range.value.min) / span) * (H - 2 * PAD)
}

const linePath = computed(() => {
  const ns = numeric.value
  if (!ns.length) return ''
  return ns.map((p, i) => `${i === 0 ? 'M' : 'L'}${x(p.t).toFixed(1)},${yVal(p.num as number).toFixed(1)}`).join(' ')
})

const areaPath = computed(() => {
  const ns = numeric.value
  if (!ns.length) return ''
  const top = ns.map((p, i) => `${i === 0 ? 'M' : 'L'}${x(p.t).toFixed(1)},${yVal(p.num as number).toFixed(1)}`).join(' ')
  const last = ns[ns.length - 1]
  const first = ns[0]
  return `${top} L${x(last.t).toFixed(1)},${H - PAD} L${x(first.t).toFixed(1)},${H - PAD} Z`
})

const onoffBars = computed(() => {
  const pts = props.points
  if (!pts.length) return [] as { x: number; w: number; y: number }[]
  const bars: { x: number; w: number; y: number }[] = []
  for (let i = 0; i < pts.length; i++) {
    const p = pts[i]
    const next = pts[i + 1]
    const on = p.state === 'on' || p.state === 'open'
    if (!on) continue
    const x0 = x(p.t)
    const x1 = next ? x(next.t) : x0 + 2
    bars.push({ x: x0, w: Math.max(1, x1 - x0), y: PAD })
  }
  return bars
})

const yTicks = computed(() => {
  const { min, max } = range.value
  const vals = [min, (min + max) / 2, max]
  return vals.map((v) => ({ v, y: yVal(v) }))
})
</script>

<template>
  <div class="chart-wrap">
    <svg :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" class="chart">
      <!-- value line chart -->
      <g v-if="kind === 'value' && numeric.length">
        <path :d="areaPath" class="area" />
        <path :d="linePath" class="line" />
        <g class="axis">
          <text
            v-for="t in yTicks"
            :key="t.v"
            :x="4"
            :y="t.y"
            class="tick"
          >{{ t.v.toFixed(1) }}</text>
        </g>
      </g>
      <!-- on/off stepped bars -->
      <g v-else-if="kind === 'onoff' && points.length">
        <line :x1="PAD" :x2="W - PAD" :y1="PAD" :y2="PAD" class="axis-line" />
        <line :x1="PAD" :x2="W - PAD" :y1="H - PAD" :y2="H - PAD" class="axis-line" />
        <rect
          v-for="(b, i) in onoffBars"
          :key="i"
          :x="b.x"
          :y="b.y"
          :width="b.w"
          :height="H - 2 * PAD"
          class="on-bar"
        />
        <text :x="4" :y="PAD + 4" class="tick">on</text>
        <text :x="4" :y="H - PAD + 4" class="tick">off</text>
      </g>
      <text v-else :x="W / 2" :y="H / 2" class="empty" text-anchor="middle">无历史数据</text>
    </svg>
    <div v-if="unit" class="chart-unit">单位: {{ unit }}</div>
  </div>
</template>

<style scoped>
.chart-wrap {
  width: 100%;
}
.chart {
  width: 100%;
  height: 12rem;
  display: block;
  background: hsl(var(--muted) / 0.5);
  border-radius: var(--radius, 0.625rem);
  border: 1px solid hsl(var(--border));
}
.line {
  fill: none;
  stroke: hsl(var(--primary));
  stroke-width: 2;
}
.area {
  fill: hsl(var(--primary) / 0.15);
  stroke: none;
}
.on-bar {
  fill: hsl(var(--primary) / 0.35);
}
.axis-line {
  stroke: hsl(var(--border));
  stroke-width: 1;
}
.tick {
  fill: hsl(var(--muted-foreground));
  font-size: 10px;
}
.empty {
  fill: hsl(var(--muted-foreground));
  font-size: 13px;
}
.chart-unit {
  font-size: 0.75rem;
  color: hsl(var(--muted-foreground));
  margin-top: 0.4rem;
}
</style>