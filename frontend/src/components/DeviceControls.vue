<script setup lang="ts">
import { ref } from 'vue'
import type { DeviceView } from '@/types/api'
import { deviceAction } from '@/api'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'
import {
  brightnessPct,
  colorTempKelvin,
  coverPosition,
  currentTemp,
  hasCap,
  hvacMode,
  hvacModes,
  isButton,
  isMediaPlayer,
  isScene,
  isVacuum,
  pctStyle,
  targetTemp,
  volumeLevel,
} from '@/lib/device'

const props = defineProps<{ device: DeviceView }>()
const emit = defineEmits<{ updated: [DeviceView] }>()

const { toast } = useToast()
const busy = ref(false)
let debounce: number | undefined

async function run(action: string, params?: Record<string, unknown>) {
  if (busy.value || !props.device.available) return
  busy.value = true
  try {
    const res = await deviceAction(props.device.id, action, params)
    emit('updated', res.data)
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '控制失败', 'err')
  } finally {
    busy.value = false
  }
}

// debounced slider commit (brightness / temp / position)
function debounceRun(action: string, params: Record<string, unknown>) {
  if (debounce) window.clearTimeout(debounce)
  debounce = window.setTimeout(() => run(action, params), 300)
}

const d = props.device
const isLight = d.domain === 'light'
const isCover = d.domain === 'cover'
const isClimate = d.domain === 'climate'
const isMedia = isMediaPlayer(d)
const isVac = isVacuum(d)
const scene = isScene(d)
const showBrightness = isLight && hasCap(d, 'brightness')
const showColorTemp = isLight && hasCap(d, 'color_temp')
const showCover = isCover && hasCap(d, 'open_close')
const showClimate = isClimate && hasCap(d, 'temperature')
const showMedia = isMedia
const showVacuum = isVac
const modes = hvacModes(d)

const brightness = ref(brightnessPct(d) ?? 0)
const cTemp = ref(colorTempKelvin(d) ?? 3000)
const position = ref(coverPosition(d) ?? 0)
const tSet = ref(targetTemp(d) ?? 22)
const volume = ref(volumeLevel(d) ?? 0)
</script>

<template>
  <div class="dc-card">
    <!-- light brightness -->
    <div v-if="showBrightness" class="dc-slider">
      <div class="dc-slider-top">
        <span class="dc-slider-label">亮度</span>
        <span class="dc-slider-val"><b>{{ brightness }}</b>%</span>
      </div>
      <input
        v-model.number="brightness"
        type="range"
        min="0"
        max="100"
        :style="pctStyle(brightness, 0, 100)"
        :disabled="busy || !device.available"
        @change="debounceRun('set_brightness', { brightness: Math.round((brightness / 100) * 255) })"
      />
      <div class="dc-service">set_brightness</div>
    </div>

    <!-- light color temp -->
    <div v-if="showColorTemp" class="dc-slider">
      <div class="dc-slider-top">
        <span class="dc-slider-label">色温</span>
        <span class="dc-slider-val"><b>{{ cTemp }}</b>K</span>
      </div>
      <input
        v-model.number="cTemp"
        type="range"
        min="2000"
        max="6500"
        step="100"
        :style="pctStyle(cTemp, 2000, 6500)"
        :disabled="busy || !device.available"
        @change="debounceRun('set_color_temp', { kelvin: cTemp })"
      />
      <div class="dc-service">set_color_temp</div>
    </div>

    <!-- cover open/stop/close + position -->
    <div v-if="showCover" class="dc-block">
      <div class="dc-power">
        <button
          type="button"
          class="dc-power-btn"
          :class="{ 'is-on': device.state === 'open' }"
          :disabled="busy || !device.available"
          @click="run('open')"
        >
          开
        </button>
        <button
          type="button"
          class="dc-power-btn"
          :disabled="busy || !device.available"
          @click="run('stop')"
        >
          停
        </button>
        <button
          type="button"
          class="dc-power-btn"
          :class="{ 'is-on': device.state === 'closed' }"
          :disabled="busy || !device.available"
          @click="run('close')"
        >
          合
        </button>
      </div>
      <div class="dc-slider">
        <div class="dc-slider-top">
          <span class="dc-slider-label">位置</span>
          <span class="dc-slider-val"><b>{{ position }}</b>%</span>
        </div>
        <input
          v-model.number="position"
          type="range"
          min="0"
          max="100"
          :style="pctStyle(position, 0, 100)"
          :disabled="busy || !device.available"
          @change="debounceRun('set_position', { position })"
        />
      </div>
    </div>

    <!-- climate temperature + mode -->
    <div v-if="showClimate" class="dc-block">
      <div class="dc-temp-hero">
        <div class="dc-temp-now">
          当前
          <b>{{ currentTemp(device) ?? '—' }}°</b>
        </div>
        <div class="dc-temp-set">
          目标
          <b>{{ tSet }}°</b>
        </div>
      </div>
      <div class="dc-slider">
        <div class="dc-slider-top">
          <span class="dc-slider-label">目标温度</span>
          <span class="dc-slider-val"><b>{{ tSet }}</b>°C</span>
        </div>
        <input
          v-model.number="tSet"
          type="range"
          min="16"
          max="30"
          step="0.5"
          :style="pctStyle(tSet, 16, 30)"
          :disabled="busy || !device.available"
          @change="debounceRun('set_temperature', { temperature: tSet })"
        />
      </div>
      <div v-if="modes.length" class="dc-chips">
        <button
          v-for="m in modes"
          :key="m"
          type="button"
          class="dc-chip"
          :class="{ active: hvacMode(device) === m }"
          :disabled="busy || !device.available"
          @click="run('set_hvac_mode', { hvac_mode: m })"
        >
          {{ m }}
        </button>
      </div>
    </div>

    <!-- scene / script activate -->
    <div v-if="scene" class="dc-block">
      <button
        type="button"
        class="btn btn-primary btn-lg"
        :disabled="busy || !device.available"
        @click="run(d.domain === 'scene' ? 'activate' : 'run')"
      >
        {{ d.domain === 'scene' ? '激活场景' : '运行脚本' }}
      </button>
    </div>

    <!-- media_player: play/pause + volume -->
    <div v-if="showMedia" class="dc-block">
      <div class="dc-power">
        <button
          type="button"
          class="dc-power-btn"
          :class="{ 'is-on': device.state === 'playing' }"
          :disabled="busy || !device.available"
          @click="run('play_pause')"
        >
          {{ device.state === 'playing' ? '暂停' : '播放' }}
        </button>
      </div>
      <div v-if="volumeLevel(device) !== null" class="dc-slider">
        <div class="dc-slider-top">
          <span class="dc-slider-label">音量</span>
          <span class="dc-slider-val"><b>{{ volume }}</b>%</span>
        </div>
        <input
          v-model.number="volume"
          type="range"
          min="0"
          max="100"
          :style="pctStyle(volume, 0, 100)"
          :disabled="busy || !device.available"
          @change="debounceRun('set_volume', { volume_level: volume / 100 })"
        />
      </div>
    </div>

    <!-- vacuum: start / pause / stop / return -->
    <div v-if="showVacuum" class="dc-block">
      <div class="dc-power">
        <button type="button" class="dc-power-btn" :disabled="busy || !device.available" @click="run('start')">清扫</button>
        <button type="button" class="dc-power-btn" :disabled="busy || !device.available" @click="run('pause')">暂停</button>
        <button type="button" class="dc-power-btn" :disabled="busy || !device.available" @click="run('stop')">停止</button>
      </div>
      <div class="actions">
        <button type="button" class="btn btn-outline btn-sm" :disabled="busy || !device.available" @click="run('return_to_base')">
          回充
        </button>
      </div>
    </div>

    <!-- button press -->
    <div v-if="isButton(device)" class="dc-block">
      <button
        type="button"
        class="btn btn-primary btn-lg"
        :disabled="busy || !device.available"
        @click="run('press')"
      >
        按下
      </button>
    </div>
  </div>
</template>

<style scoped>
.dc-card {
  border: 1px solid hsl(var(--glass-border-strong));
  border-radius: var(--radius, 0.625rem);
  background: hsl(var(--glass));
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  padding: 1rem;
  margin-bottom: 1.25rem;
  box-shadow: var(--shadow-sm);
}
.dc-block {
  margin-bottom: 0.75rem;
}
.dc-block:last-child {
  margin-bottom: 0;
}
</style>