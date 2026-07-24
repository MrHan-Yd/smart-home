<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  createScenario,
  deleteScenario,
  fetchDevices,
  fetchScenarios,
  patchScenario,
  replaceScenarioSteps,
  runScenario,
} from '@/api'
import type { Scenario, ScenarioStep } from '@/types/api'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'

const { toast } = useToast()
const loading = ref(true)
const scenarios = ref<Scenario[]>([])

// device catalog for step editor
type DevLite = { id: string; name: string; entity_id: string; domain: string; capabilities: string[] }
const devices = ref<DevLite[]>([])

// editor state
const editing = ref<Scenario | null>(null)
const draftName = ref('')
const draftSteps = ref<Array<{ device_id: string; action: string; params: string; delay_ms: number }>>([])
const saving = ref(false)
const running = ref<string | null>(null)

// new scenario
const newName = ref('')
const adding = ref(false)

async function load() {
  loading.value = true
  try {
    const [res, dr] = await Promise.all([fetchScenarios(), fetchDevices()])
    scenarios.value = res.data.items || []
    devices.value = (dr.data.items || []).map((d) => ({
      id: d.id,
      name: d.name,
      entity_id: d.entity_id,
      domain: d.domain,
      capabilities: d.capabilities || [],
    }))
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '加载失败', 'err')
  } finally {
    loading.value = false
  }
}

function actionsFor(deviceID: string): string[] {
  const d = devices.value.find((x) => x.id === deviceID)
  if (!d) return ['turn_on', 'turn_off', 'toggle']
  const caps = d.capabilities
  const out: string[] = []
  if (caps.includes('on_off')) out.push('turn_on', 'turn_off', 'toggle')
  if (caps.includes('brightness')) out.push('set_brightness')
  if (caps.includes('color_temp')) out.push('set_color_temp')
  if (caps.includes('open_close')) out.push('open', 'close', 'stop')
  if (caps.includes('position')) out.push('set_position')
  if (caps.includes('hvac_mode')) out.push('set_hvac_mode')
  if (caps.includes('temperature')) out.push('set_temperature')
  if (caps.includes('lock')) out.push('lock', 'unlock')
  if (caps.includes('activate')) out.push(d.domain === 'scene' ? 'activate' : 'run')
  if (caps.includes('play_pause')) out.push('play_pause', 'play', 'pause')
  if (caps.includes('volume')) out.push('set_volume')
  if (caps.includes('start_stop')) out.push('start', 'stop', 'return_to_base')
  if (out.length === 0) out.push('turn_on', 'turn_off', 'toggle')
  return out
}

async function add() {
  if (!newName.value.trim()) return
  adding.value = true
  try {
    await createScenario({ name: newName.value.trim() })
    newName.value = ''
    toast('已创建场景')
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '创建失败', 'err')
  } finally {
    adding.value = false
  }
}

function startEdit(sc: Scenario) {
  editing.value = sc
  draftName.value = sc.name
  draftSteps.value = (sc.steps || []).map((st) => ({
    device_id: st.device_id,
    action: st.action,
    params: st.params ? JSON.stringify(st.params) : '{}',
    delay_ms: st.delay_ms,
  }))
}

function cancelEdit() {
  editing.value = null
  draftSteps.value = []
}

function addStep() {
  const first = devices.value[0]
  draftSteps.value.push({
    device_id: first?.id || '',
    action: 'turn_on',
    params: '{}',
    delay_ms: 0,
  })
}

function removeStep(i: number) {
  draftSteps.value.splice(i, 1)
}

function moveStep(i: number, dir: -1 | 1) {
  const j = i + dir
  if (j < 0 || j >= draftSteps.value.length) return
  const arr = draftSteps.value
  ;[arr[i], arr[j]] = [arr[j], arr[i]]
}

async function saveDraft() {
  if (!editing.value || !draftName.value.trim()) return
  saving.value = true
  try {
    await patchScenario(editing.value.id, { name: draftName.value.trim() })
    const steps = draftSteps.value.map((st) => {
      let params: Record<string, unknown> = {}
      try {
        params = JSON.parse(st.params || '{}')
      } catch {
        params = {}
      }
      return { device_id: st.device_id, action: st.action, params, delay_ms: st.delay_ms }
    })
    await replaceScenarioSteps(editing.value.id, steps)
    toast('已保存')
    editing.value = null
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '保存失败', 'err')
  } finally {
    saving.value = false
  }
}

async function run(sc: Scenario) {
  running.value = sc.id
  try {
    const res = await runScenario(sc.id)
    const results = res.data.results || []
    const ok = results.filter((r) => r.success).length
    const fail = results.length - ok
    if (fail === 0) toast(`已执行 ${ok} 步`)
    else toast(`成功 ${ok} · 失败 ${fail}`, 'err')
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '执行失败', 'err')
  } finally {
    running.value = null
  }
}

async function remove(sc: Scenario) {
  if (!confirm(`删除场景「${sc.name}」？`)) return
  try {
    await deleteScenario(sc.id)
    toast('已删除')
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '删除失败', 'err')
  }
}

function devName(id: string) {
  return devices.value.find((x) => x.id === id)?.name || '已删除设备'
}

const editingId = computed(() => editing.value?.id)

onMounted(load)
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <h1>场景</h1>
        <p>本系统编排的多步一键执行 · 串行 · 步骤含前置延迟</p>
      </div>
      <div class="page-actions">
        <input v-model="newName" class="input" placeholder="新场景名称…" @keyup.enter="add" />
        <button type="button" class="btn btn-primary btn-sm" :disabled="adding || !newName.trim()" @click="add">
          新建
        </button>
      </div>
    </div>

    <div v-if="loading" class="muted">加载中…</div>
    <div v-else-if="!scenarios.length" class="muted">暂无场景</div>
    <div v-else class="scen-list">
      <article v-for="sc in scenarios" :key="sc.id" class="scen-card">
        <template v-if="editingId === sc.id">
          <div class="editor">
            <div class="editor-head">
              <input v-model="draftName" class="input" placeholder="场景名称" />
              <span class="muted small">共 {{ draftSteps.length }} 步</span>
            </div>
            <div v-if="!draftSteps.length" class="muted small">还没有步骤</div>
            <div v-for="(st, i) in draftSteps" :key="i" class="step-row">
              <span class="step-idx">{{ i + 1 }}</span>
              <select v-model="st.device_id" class="select step-dev">
                <option v-for="d in devices" :key="d.id" :value="d.id">{{ d.name }}</option>
              </select>
              <select v-model="st.action" class="select step-act">
                <option v-for="a in actionsFor(st.device_id)" :key="a" :value="a">{{ a }}</option>
              </select>
              <input
                v-model.number="st.delay_ms"
                type="number"
                min="0"
                class="input step-delay"
                placeholder="延迟ms"
                title="前置延迟（毫秒）"
              />
              <button type="button" class="btn btn-ghost btn-sm" title="上移" @click="moveStep(i, -1)">↑</button>
              <button type="button" class="btn btn-ghost btn-sm" title="下移" @click="moveStep(i, 1)">↓</button>
              <button type="button" class="btn btn-danger btn-sm" title="删除步骤" @click="removeStep(i)">✕</button>
            </div>
            <div class="editor-foot">
              <button type="button" class="btn btn-outline btn-sm" @click="addStep">+ 添加步骤</button>
              <div class="foot-right">
                <button type="button" class="btn btn-ghost btn-sm" @click="cancelEdit">取消</button>
                <button type="button" class="btn btn-primary btn-sm" :disabled="saving" @click="saveDraft">保存</button>
              </div>
            </div>
          </div>
        </template>
        <template v-else>
          <div class="scen-head">
            <span class="scen-ico">✦</span>
            <div class="scen-body">
              <div class="scen-name">{{ sc.name }}</div>
              <div class="scen-meta">{{ (sc.steps || []).length }} 步 · 执行 {{ sc.run_count }} 次</div>
            </div>
            <div class="scen-actions">
              <button
                type="button"
                class="btn btn-primary btn-sm"
                :disabled="running === sc.id"
                @click="run(sc)"
              >{{ running === sc.id ? '执行中…' : '立即执行' }}</button>
              <button type="button" class="btn btn-outline btn-sm" @click="startEdit(sc)">编辑</button>
              <button type="button" class="btn btn-danger btn-sm" @click="remove(sc)">删除</button>
            </div>
          </div>
          <div v-if="sc.steps?.length" class="scen-steps">
            <div v-for="(st, i) in sc.steps" :key="i" class="mini-step">
              <span class="step-idx">{{ i + 1 }}</span>
              <span class="mini-dev">{{ devName((st as ScenarioStep).device_id) }}</span>
              <span class="mini-act">{{ (st as ScenarioStep).action }}</span>
              <span v-if="(st as ScenarioStep).delay_ms > 0" class="mini-delay">+{{ (st as ScenarioStep).delay_ms }}ms</span>
            </div>
          </div>
        </template>
      </article>
    </div>
  </section>
</template>

<style scoped>
.muted {
  color: hsl(var(--muted-foreground));
}
.small {
  font-size: 0.78rem;
}
.scen-list {
  display: grid;
  gap: 0.75rem;
}
.scen-card {
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius, 0.625rem);
  background: hsl(var(--card));
  padding: 0.85rem 1rem;
}
.scen-head {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.scen-ico {
  font-size: 1.3rem;
}
.scen-body {
  flex: 1;
  min-width: 0;
}
.scen-name {
  font-weight: 600;
  font-size: 0.95rem;
}
.scen-meta {
  font-size: 0.75rem;
  color: hsl(var(--muted-foreground));
}
.scen-actions {
  display: flex;
  gap: 0.4rem;
}
.scen-steps {
  margin-top: 0.6rem;
  border-top: 1px solid hsl(var(--border) / 0.6);
  padding-top: 0.5rem;
  display: grid;
  gap: 0.3rem;
}
.mini-step {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
}
.mini-dev {
  font-weight: 500;
}
.mini-act {
  color: hsl(var(--primary));
  font-family: ui-monospace, monospace;
  font-size: 0.75rem;
}
.mini-delay {
  color: hsl(var(--muted-foreground));
  font-size: 0.72rem;
}
.editor {
  display: grid;
  gap: 0.5rem;
}
.editor-head {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.editor-head .input {
  flex: 1;
}
.step-row {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  flex-wrap: wrap;
}
.step-idx {
  width: 1.5rem;
  text-align: center;
  font-size: 0.78rem;
  color: hsl(var(--muted-foreground));
}
.step-dev {
  min-width: 9rem;
}
.step-act {
  min-width: 8rem;
}
.step-delay {
  width: 5rem;
}
.editor-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.foot-right {
  display: flex;
  gap: 0.4rem;
}
</style>