<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { fetchDevices } from '@/api'
import type { DeviceView } from '@/types/api'
import DeviceCard from '@/components/DeviceCard.vue'
import { useToast } from '@/composables/useToast'
import { canToggle, isOn } from '@/lib/device'
import { ApiError } from '@/lib/http'

const router = useRouter()
const { toast } = useToast()
const loading = ref(true)
const devices = ref<DeviceView[]>([])
let timer: number | undefined

const stats = computed(() => {
  const list = devices.value
  const total = list.length
  const on = list.filter((d) => canToggle(d) && isOn(d)).length
  const bad = list.filter((d) => !d.available).length
  const temp = list.find(
    (d) =>
      d.domain === 'sensor' &&
      (d.entity_id.includes('temp') ||
        String(d.attributes?.device_class || '') === 'temperature'),
  )
  return {
    total,
    on,
    bad,
    temp: temp ? temp.primary_display || temp.state : '—',
  }
})

const favorites = computed(() => devices.value.filter((d) => d.favorite))

async function load() {
  loading.value = true
  try {
    const res = await fetchDevices()
    devices.value = res.data.items || []
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : '加载失败'
    toast(msg, 'err')
  } finally {
    loading.value = false
  }
}

function onUpdated(d: DeviceView) {
  const i = devices.value.findIndex((x) => x.id === d.id)
  if (i >= 0) devices.value[i] = d
  else devices.value = [...devices.value, d]
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 15000)
})
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <h1>总览</h1>
        <p>家庭状态墙 · 快捷控制常用设备</p>
      </div>
      <div class="page-actions">
        <button type="button" class="btn btn-outline btn-sm" :disabled="loading" @click="load">
          刷新
        </button>
        <button type="button" class="btn btn-primary btn-sm" @click="router.push('/add')">
          添加设备
        </button>
      </div>
    </div>

    <div class="stat-row">
      <div class="stat-card">
        <div class="label">我的设备</div>
        <div class="value">{{ stats.total }}</div>
        <div class="hint">已从 HA 纳入</div>
      </div>
      <div class="stat-card">
        <div class="label">开启中</div>
        <div class="value">{{ stats.on }}</div>
        <div class="hint">可开关类设备</div>
      </div>
      <div class="stat-card">
        <div class="label">异常 / 离线</div>
        <div class="value">{{ stats.bad }}</div>
        <div class="hint">unavailable</div>
      </div>
      <div class="stat-card">
        <div class="label">环境</div>
        <div class="value" style="font-size: 1.25rem">{{ stats.temp }}</div>
        <div class="hint">温度传感器（若有）</div>
      </div>
    </div>

    <div v-if="loading && !devices.length" class="empty-hint">加载中…</div>
    <template v-else>
      <template v-if="favorites.length">
        <div class="section-title">收藏</div>
        <div class="device-grid">
          <DeviceCard
            v-for="d in favorites"
            :key="d.id"
            :device="d"
            @updated="onUpdated"
          />
        </div>
      </template>
      <div class="section-title">全部设备</div>
      <div v-if="!devices.length" class="empty-hint">
        还没有设备。
        <button type="button" class="btn btn-primary btn-sm" style="margin-left: 0.5rem" @click="router.push('/add')">
          去添加
        </button>
      </div>
      <div v-else class="device-grid">
        <DeviceCard v-for="d in devices" :key="d.id" :device="d" @updated="onUpdated" />
      </div>
    </template>
  </section>
</template>

<style scoped>
.empty-hint {
  color: hsl(var(--muted-foreground));
  padding: 1.5rem 0;
  font-size: 0.9rem;
}
</style>
