<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { fetchDevices } from '@/api'
import type { DeviceView } from '@/types/api'
import DeviceCard from '@/components/DeviceCard.vue'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'

const router = useRouter()
const { toast } = useToast()
const loading = ref(true)
const devices = ref<DeviceView[]>([])
const q = ref('')
const domain = ref('')
const flag = ref<'all' | 'controllable' | 'favorite'>('all')

const domains = computed(() => {
  const set = new Set(devices.value.map((d) => d.domain))
  return Array.from(set).sort()
})

const filtered = computed(() => {
  let list = devices.value
  if (domain.value) list = list.filter((d) => d.domain === domain.value)
  if (flag.value === 'favorite') list = list.filter((d) => d.favorite)
  if (flag.value === 'controllable') list = list.filter((d) => d.control_level !== 'read_only')
  const s = q.value.trim().toLowerCase()
  if (s) {
    list = list.filter(
      (d) =>
        d.name.toLowerCase().includes(s) ||
        d.entity_id.toLowerCase().includes(s) ||
        d.domain.toLowerCase().includes(s),
    )
  }
  return list
})

async function load() {
  loading.value = true
  try {
    const res = await fetchDevices()
    devices.value = res.data.items || []
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '加载失败', 'err')
  } finally {
    loading.value = false
  }
}

function onUpdated(d: DeviceView) {
  const i = devices.value.findIndex((x) => x.id === d.id)
  if (i >= 0) devices.value[i] = d
}

onMounted(load)
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <h1>设备</h1>
        <p>筛选、收藏、进入详情控制</p>
      </div>
      <div class="page-actions">
        <button type="button" class="btn btn-primary btn-sm" @click="router.push('/add')">
          添加设备
        </button>
      </div>
    </div>

    <div class="toolbar">
      <input v-model="q" class="input" type="search" placeholder="搜索名称 / entity…" />
    </div>

    <div class="chip-row" style="margin-bottom: 0.75rem">
      <button
        type="button"
        class="chip"
        :class="{ active: !domain }"
        @click="domain = ''"
      >
        全部类型
      </button>
      <button
        v-for="d in domains"
        :key="d"
        type="button"
        class="chip"
        :class="{ active: domain === d }"
        @click="domain = d"
      >
        {{ d }}
      </button>
    </div>

    <div class="chip-row" style="margin-bottom: 1rem">
      <button type="button" class="chip" :class="{ active: flag === 'all' }" @click="flag = 'all'">
        全部
      </button>
      <button
        type="button"
        class="chip"
        :class="{ active: flag === 'controllable' }"
        @click="flag = 'controllable'"
      >
        仅可控制
      </button>
      <button
        type="button"
        class="chip"
        :class="{ active: flag === 'favorite' }"
        @click="flag = 'favorite'"
      >
        仅收藏
      </button>
    </div>

    <div v-if="loading" class="muted">加载中…</div>
    <div v-else-if="!filtered.length" class="muted">无匹配设备</div>
    <div v-else class="device-grid">
      <DeviceCard v-for="d in filtered" :key="d.id" :device="d" @updated="onUpdated" />
    </div>
  </section>
</template>

<style scoped>
.muted {
  color: hsl(var(--muted-foreground));
  padding: 1rem 0;
}
</style>
