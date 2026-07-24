<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { batchCreateDevices, createCompositeDevice, createDevice, discoverEntities } from '@/api'
import type { DiscoverEntity } from '@/types/api'
import { useToast } from '@/composables/useToast'
import { domainEmoji } from '@/lib/device'
import { ApiError } from '@/lib/http'

const { toast } = useToast()
const loading = ref(false)
const items = ref<DiscoverEntity[]>([])
const total = ref(0)
const q = ref('')
const domain = ref('')
const onlyNew = ref(true)
const selected = ref<Set<string>>(new Set())
const adding = ref(false)

const domains = [
  '',
  'light',
  'switch',
  'sensor',
  'binary_sensor',
  'climate',
  'cover',
  'fan',
  'media_player',
  'lock',
  'scene',
  'script',
  'button',
]

const selectedCount = computed(() => selected.value.size)

async function load() {
  loading.value = true
  try {
    const res = await discoverEntities({
      q: q.value.trim() || undefined,
      domain: domain.value || undefined,
      only_new: onlyNew.value,
      page_size: 100,
    })
    items.value = res.data.items || []
    total.value = res.data.total
    selected.value = new Set()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '发现失败', 'err')
  } finally {
    loading.value = false
  }
}

function toggleSelect(eid: string) {
  const s = new Set(selected.value)
  if (s.has(eid)) s.delete(eid)
  else s.add(eid)
  selected.value = s
}

async function addOne(ent: DiscoverEntity) {
  if (ent.already_added) return
  adding.value = true
  try {
    await createDevice({ entity_id: ent.entity_id, name: ent.name })
    toast(`已添加 ${ent.name}`)
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '添加失败', 'err')
  } finally {
    adding.value = false
  }
}

async function batchAdd() {
  const ids = Array.from(selected.value)
  if (!ids.length) return
  adding.value = true
  try {
    const res = await batchCreateDevices(ids)
    const n = res.data.created?.length || 0
    toast(`已添加 ${n} 个设备`)
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '批量添加失败', 'err')
  } finally {
    adding.value = false
  }
}

async function compositeAdd() {
  const ids = Array.from(selected.value)
  if (ids.length < 2) return
  adding.value = true
  try {
    await createCompositeDevice({ entity_ids: ids })
    toast(`已组合添加 ${ids.length} 个实体`)
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '组合添加失败', 'err')
  } finally {
    adding.value = false
  }
}

let debounce: number | undefined
watch([q, domain, onlyNew], () => {
  window.clearTimeout(debounce)
  debounce = window.setTimeout(load, 280)
})

onMounted(load)
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <h1>添加设备</h1>
        <p>从 Home Assistant 发现实体 · 勾选纳入「我的设备」（非配网）</p>
      </div>
      <div class="page-actions">
        <button type="button" class="btn btn-outline btn-sm" :disabled="loading" @click="load">
          同步 HA
        </button>
        <button
          type="button"
          class="btn btn-outline btn-sm"
          :disabled="selectedCount < 2 || adding"
          title="将选中实体组合为单个设备（多 entity）"
          @click="compositeAdd"
        >
          组合添加{{ selectedCount > 1 ? ` (${selectedCount})` : '' }}
        </button>
        <button
          type="button"
          class="btn btn-primary btn-sm"
          :disabled="!selectedCount || adding"
          @click="batchAdd"
        >
          批量添加{{ selectedCount ? ` (${selectedCount})` : '' }}
        </button>
      </div>
    </div>

    <div class="toolbar">
      <input v-model="q" class="input" type="search" placeholder="筛选 entity / 名称…" />
      <label class="chip" style="cursor: pointer">
        <input v-model="onlyNew" type="checkbox" style="margin-right: 0.35rem" />
        仅未添加
      </label>
    </div>

    <div class="chip-row" style="margin-bottom: 1rem">
      <button
        v-for="d in domains"
        :key="d || 'all'"
        type="button"
        class="chip"
        :class="{ active: domain === d }"
        @click="domain = d"
      >
        {{ d || '全部' }}
      </button>
    </div>

    <p class="hint-line">共 {{ total }} 项 · 当前页 {{ items.length }}</p>

    <div v-if="loading" class="muted">同步中…</div>
    <div v-else-if="!items.length" class="muted">无实体（检查 HA 配置或筛选条件）</div>
    <div v-else class="discover-list">
      <div v-for="ent in items" :key="ent.entity_id" class="discover-row">
        <label class="discover-check">
          <input
            type="checkbox"
            :disabled="ent.already_added"
            :checked="selected.has(ent.entity_id)"
            @change="toggleSelect(ent.entity_id)"
          />
        </label>
        <span class="discover-ico">{{ domainEmoji(ent.domain) }}</span>
        <div class="discover-body">
          <div class="discover-name">
            {{ ent.name }}
            <span v-if="ent.already_added" class="badge">已添加</span>
          </div>
          <div class="discover-meta">
            {{ ent.entity_id }} · {{ ent.state }} · {{ ent.control_level }}
          </div>
        </div>
        <button
          type="button"
          class="btn btn-sm btn-outline"
          :disabled="ent.already_added || adding"
          @click="addOne(ent)"
        >
          添加
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.hint-line {
  font-size: 0.8rem;
  color: hsl(var(--muted-foreground));
  margin: 0 0 0.75rem;
}
.muted {
  color: hsl(var(--muted-foreground));
  padding: 1rem 0;
}
.discover-list {
  display: grid;
  gap: 0.5rem;
}
.discover-row {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.65rem 0.75rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius, 0.625rem);
  background: hsl(var(--card));
}
.discover-ico {
  font-size: 1.1rem;
}
.discover-body {
  flex: 1;
  min-width: 0;
}
.discover-name {
  font-weight: 500;
  font-size: 0.9rem;
}
.discover-meta {
  font-size: 0.75rem;
  color: hsl(var(--muted-foreground));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.badge {
  margin-left: 0.35rem;
  font-size: 0.7rem;
  padding: 0.1rem 0.35rem;
  border-radius: 999px;
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
  font-weight: 500;
}
</style>
