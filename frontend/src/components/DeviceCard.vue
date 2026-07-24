<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import type { DeviceView } from '@/types/api'
import { canRun, canToggle, compositeMemberCount, domainEmoji, isOn, isComposite, toggleAction } from '@/lib/device'
import { deviceAction, patchDevice } from '@/api'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'

const props = defineProps<{ device: DeviceView }>()
const emit = defineEmits<{ updated: [DeviceView] }>()

const router = useRouter()
const { toast } = useToast()
const busy = ref(false)

function openDetail() {
  router.push(`/devices/${props.device.id}`)
}

async function onToggle(e: Event) {
  e.stopPropagation()
  if (!canToggle(props.device) || busy.value) return
  if (!props.device.available) {
    toast('设备不可用', 'err')
    return
  }
  busy.value = true
  try {
    const action = toggleAction(props.device)
    const res = await deviceAction(props.device.id, action)
    emit('updated', res.data)
  } catch (err) {
    toast(err instanceof ApiError ? err.message : '控制失败', 'err')
  } finally {
    busy.value = false
  }
}

async function onRun(e: Event) {
  e.stopPropagation()
  if (!canRun(props.device) || busy.value) return
  busy.value = true
  try {
    const action =
      props.device.domain === 'scene' ? 'activate' : props.device.domain === 'script' ? 'run' : 'press'
    const res = await deviceAction(props.device.id, action)
    if (res.data) emit('updated', res.data)
    else toast('已执行')
  } catch (err) {
    toast(err instanceof ApiError ? err.message : '执行失败', 'err')
  } finally {
    busy.value = false
  }
}

async function onStar(e: Event) {
  e.stopPropagation()
  try {
    const res = await patchDevice(props.device.id, { favorite: !props.device.favorite })
    emit('updated', res.data)
  } catch {
    toast('更新收藏失败', 'err')
  }
}
</script>

<template>
  <article
    class="device-card"
    :class="{ on: isOn(device), unavailable: !device.available }"
    role="button"
    tabindex="0"
    @click="openDetail"
    @keydown.enter="openDetail"
  >
    <button
      type="button"
      class="fav"
      :class="{ active: device.favorite }"
      title="收藏"
      @click="onStar"
    >
      {{ device.favorite ? '★' : '☆' }}
    </button>
    <div class="ico">{{ isComposite(device) ? '⛦' : domainEmoji(device.domain) }}</div>
    <div class="name">
      {{ device.name }}
      <span v-if="isComposite(device)" class="badge comp" :title="`组合设备 · ${compositeMemberCount(device)} 个成员`">
        {{ compositeMemberCount(device) }}件
      </span>
    </div>
    <div class="meta">{{ isComposite(device) ? '组合设备' : device.domain }}</div>
    <div class="state-line">
      <span class="state-text" :class="{ on: isOn(device) }">
        {{ device.primary_display || device.state || '—' }}
      </span>
      <button
        v-if="canToggle(device)"
        type="button"
        class="toggle"
        :class="{ on: isOn(device) }"
        :disabled="busy || !device.available"
        role="switch"
        :aria-checked="isOn(device)"
        @click="onToggle"
      />
      <button
        v-else-if="canRun(device)"
        type="button"
        class="btn btn-sm btn-primary"
        :disabled="busy || !device.available"
        @click="onRun"
      >执行</button>
      <span v-else class="badge ro">只读</span>
    </div>
  </article>
</template>

<style scoped>
.badge.comp {
  margin-left: 0.3rem;
  font-size: 0.65rem;
  padding: 0.05rem 0.35rem;
  border-radius: 999px;
  background: hsl(var(--primary) / 0.15);
  color: hsl(var(--primary));
  font-weight: 600;
  vertical-align: middle;
}
</style>
