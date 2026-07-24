<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { deleteDevice, deviceAction, fetchDevice, fetchRooms, patchDevice } from '@/api'
import type { DeviceView, Room } from '@/types/api'
import { canToggle, compositeMemberCount, domainEmoji, isOn, isComposite, toggleAction } from '@/lib/device'
import DeviceControls from '@/components/DeviceControls.vue'
import HistoryChart from '@/components/HistoryChart.vue'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'
import { fetchDeviceHistory } from '@/api'
import type { HistoryPoint } from '@/api'

const route = useRoute()
const router = useRouter()
const { toast } = useToast()
const device = ref<DeviceView | null>(null)
const rooms = ref<Room[]>([])
const loading = ref(true)
const busy = ref(false)
const range = ref<'24h' | '7d'>('24h')
const history = ref<HistoryPoint[]>([])
const histLoading = ref(false)

const id = computed(() => route.params.id as string)

const histKind = computed<'value' | 'onoff'>(() => {
  if (!device.value) return 'value'
  const d = device.value.domain
  return d === 'light' || d === 'switch' || d === 'input_boolean' || d === 'cover' || d === 'binary_sensor'
    ? 'onoff'
    : 'value'
})
const histUnit = computed<string>(() => {
  const u = device.value?.attributes?.unit_of_measurement
  return typeof u === 'string' ? u : ''
})

async function loadHistory() {
  if (!device.value) return
  histLoading.value = true
  try {
    const res = await fetchDeviceHistory(device.value.id, range.value)
    history.value = res.data.points || []
  } catch (e) {
    if (!(e instanceof ApiError && e.httpStatus === 502)) {
      toast(e instanceof ApiError ? e.message : '历史加载失败', 'err')
    }
    history.value = []
  } finally {
    histLoading.value = false
  }
}

async function load() {
  loading.value = true
  try {
    const [res, rr] = await Promise.all([fetchDevice(id.value), fetchRooms()])
    device.value = res.data
    rooms.value = rr.data.items || []
    void loadHistory()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '加载失败', 'err')
    router.replace('/devices')
  } finally {
    loading.value = false
  }
}

async function setRoom(e: Event) {
  if (!device.value) return
  const v = (e.target as HTMLSelectElement).value
  const roomID = v || null
  try {
    const res = await patchDevice(device.value.id, { room_id: roomID })
    device.value = res.data
    toast(roomID ? '已归属房间' : '已移出房间')
  } catch {
    toast('更新房间失败', 'err')
  }
}

async function doAction(action: string, params?: Record<string, unknown>) {
  if (!device.value || busy.value) return
  busy.value = true
  try {
    const res = await deviceAction(device.value.id, action, params)
    device.value = res.data
    toast('已执行')
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '控制失败', 'err')
  } finally {
    busy.value = false
  }
}

async function onToggle() {
  if (!device.value) return
  await doAction(toggleAction(device.value))
}

async function onFav() {
  if (!device.value) return
  try {
    const res = await patchDevice(device.value.id, { favorite: !device.value.favorite })
    device.value = res.data
  } catch {
    toast('更新失败', 'err')
  }
}

async function onRemove() {
  if (!device.value) return
  if (!confirm(`从本系统移除「${device.value.name}」？不影响 HA。`)) return
  try {
    await deleteDevice(device.value.id)
    toast('已移除')
    router.replace('/devices')
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '移除失败', 'err')
  }
}

onMounted(load)

watch(range, loadHistory)
watch(
  () => device.value?.id,
  (nid, oid) => {
    if (nid && nid !== oid) loadHistory()
  },
)
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <button type="button" class="btn btn-ghost btn-sm" @click="router.back()">← 返回</button>
        <h1 v-if="device" style="margin-top: 0.5rem">
          <span style="margin-right: 0.35rem">{{ isComposite(device) ? '⛦' : domainEmoji(device.domain) }}</span>
          {{ device.name }}
          <span v-if="isComposite(device)" class="comp-tag">组合 · {{ compositeMemberCount(device) }} 件</span>
        </h1>
        <p v-if="device">{{ device.entity_id }}</p>
      </div>
      <div v-if="device" class="page-actions">
        <button type="button" class="btn btn-outline btn-sm" @click="onFav">
          {{ device.favorite ? '★ 已收藏' : '☆ 收藏' }}
        </button>
        <button type="button" class="btn btn-danger btn-sm" @click="onRemove">移除</button>
      </div>
    </div>

    <div v-if="loading" class="muted">加载中…</div>
    <template v-else-if="device">
      <div class="detail-card">
        <div class="detail-row">
          <span class="k">状态</span>
          <span class="v">{{ device.primary_display || device.state || '—' }}</span>
        </div>
        <div class="detail-row">
          <span class="k">可用性</span>
          <span class="v">{{ device.available ? '正常' : '不可用' }}</span>
        </div>
        <div class="detail-row">
          <span class="k">类型</span>
          <span class="v">{{ device.domain }} · {{ device.control_level }}</span>
        </div>
        <div class="detail-row">
          <span class="k">能力</span>
          <span class="v">{{ (device.capabilities || []).join(', ') || '—' }}</span>
        </div>
        <div class="detail-row">
          <span class="k">房间</span>
          <span class="v">
            <select class="select room-select" :value="device.room_id || ''" @change="setRoom">
              <option value="">未分配</option>
              <option v-for="r in rooms" :key="r.id" :value="r.id">{{ r.name }}</option>
            </select>
          </span>
        </div>
      </div>

      <div v-if="canToggle(device)" class="control-panel">
        <div class="section-title">开关</div>
        <button
          type="button"
          class="btn btn-primary btn-lg"
          :disabled="busy || !device.available"
          @click="onToggle"
        >
          {{ isOn(device) ? '关闭' : '开启' }}
        </button>
        <button
          type="button"
          class="btn btn-outline"
          style="margin-left: 0.5rem"
          :disabled="busy || !device.available"
          @click="doAction('toggle')"
        >
          切换
        </button>
      </div>

      <DeviceControls v-if="device" :device="device" @updated="device = $event" />

      <div v-if="isComposite(device) && device.members?.length" class="members-block">
        <div class="section-title">成员设备 ({{ device.members.length }})</div>
        <table class="members-table">
          <thead>
            <tr><th>Entity</th><th>状态</th><th>可用</th></tr>
          </thead>
          <tbody>
            <tr v-for="m in device.members" :key="m.entity_id">
              <td class="mono">{{ m.entity_id }}</td>
              <td>{{ m.state || '—' }}</td>
              <td>{{ m.available ? '正常' : '不可用' }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="device.control_level === 'read_only'" class="control-panel muted">
        只读设备 · 无控制动作
      </div>

      <div class="history-block">
        <div class="section-title">
          历史
          <span class="hist-range">
            <button
              v-for="r in (['24h','7d'] as const)"
              :key="r"
              type="button"
              class="chip"
              :class="{ active: range === r }"
              @click="range = r"
            >{{ r }}</button>
          </span>
        </div>
        <div v-if="histLoading" class="muted">加载历史…</div>
        <HistoryChart v-else :points="history" :unit="histUnit" :kind="histKind" />
      </div>

      <div v-if="device.attributes" class="attr-block">
        <div class="section-title">Attributes</div>
        <pre class="attr-pre">{{ JSON.stringify(device.attributes, null, 2) }}</pre>
      </div>
    </template>
  </section>
</template>

<style scoped>
.muted {
  color: hsl(var(--muted-foreground));
}
.detail-card {
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius, 0.625rem);
  background: hsl(var(--card));
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
}
.detail-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.4rem 0;
  font-size: 0.9rem;
  border-bottom: 1px solid hsl(var(--border) / 0.6);
}
.detail-row:last-child {
  border-bottom: none;
}
.k {
  color: hsl(var(--muted-foreground));
}
.v {
  text-align: right;
  word-break: break-all;
}
.room-select {
  min-width: 8rem;
}
.control-panel {
  margin-bottom: 1.25rem;
}
.history-block {
  margin-bottom: 1.25rem;
}
.hist-range {
  display: inline-flex;
  gap: 0.35rem;
  margin-left: 0.75rem;
}
.attr-pre {
  font-size: 0.75rem;
  overflow: auto;
  padding: 0.75rem;
  border-radius: var(--radius, 0.625rem);
  background: hsl(var(--muted));
  border: 1px solid hsl(var(--border));
  max-height: 16rem;
}
.comp-tag {
  font-size: 0.7rem;
  font-weight: 500;
  padding: 0.1rem 0.45rem;
  border-radius: 999px;
  background: hsl(var(--primary) / 0.15);
  color: hsl(var(--primary));
  vertical-align: middle;
  margin-left: 0.3rem;
}
.members-block {
  margin-bottom: 1.25rem;
}
.members-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}
.members-table th,
.members-table td {
  text-align: left;
  padding: 0.4rem 0.5rem;
  border-bottom: 1px solid hsl(var(--border) / 0.6);
}
.members-table th {
  color: hsl(var(--muted-foreground));
  font-weight: 500;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.78rem;
}
</style>
