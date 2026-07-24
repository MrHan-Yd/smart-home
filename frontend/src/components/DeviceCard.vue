<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import type { DeviceView } from '@/types/api'
import { canToggle, domainEmoji, isOn, toggleAction } from '@/lib/device'
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
    <div class="ico">{{ domainEmoji(device.domain) }}</div>
    <div class="name">{{ device.name }}</div>
    <div class="meta">{{ device.domain }}</div>
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
      <span v-else class="badge ro">只读</span>
    </div>
  </article>
</template>
