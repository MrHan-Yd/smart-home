<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { deleteDevice, deviceAction, fetchDevice, patchDevice } from '@/api'
import type { DeviceView } from '@/types/api'
import { canToggle, domainEmoji, isOn, toggleAction } from '@/lib/device'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'

const route = useRoute()
const router = useRouter()
const { toast } = useToast()
const device = ref<DeviceView | null>(null)
const loading = ref(true)
const busy = ref(false)

const id = computed(() => route.params.id as string)

async function load() {
  loading.value = true
  try {
    const res = await fetchDevice(id.value)
    device.value = res.data
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '加载失败', 'err')
    router.replace('/devices')
  } finally {
    loading.value = false
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
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <button type="button" class="btn btn-ghost btn-sm" @click="router.back()">← 返回</button>
        <h1 v-if="device" style="margin-top: 0.5rem">
          <span style="margin-right: 0.35rem">{{ domainEmoji(device.domain) }}</span>
          {{ device.name }}
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
      <div v-else class="control-panel muted">只读设备 · 无控制动作</div>

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
.control-panel {
  margin-bottom: 1.25rem;
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
</style>
