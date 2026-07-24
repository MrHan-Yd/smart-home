<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createRoom, deleteRoom, fetchRooms, patchRoom } from '@/api'
import type { Room } from '@/types/api'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'

const { toast } = useToast()
const loading = ref(true)
const rooms = ref<Room[]>([])
const newName = ref('')
const adding = ref(false)
const editing = ref<Record<string, string>>({})

async function load() {
  loading.value = true
  try {
    const res = await fetchRooms()
    rooms.value = res.data.items || []
    editing.value = {}
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '加载失败', 'err')
  } finally {
    loading.value = false
  }
}

async function add() {
  if (!newName.value.trim()) return
  adding.value = true
  try {
    await createRoom({ name: newName.value.trim() })
    newName.value = ''
    toast('已添加')
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '添加失败', 'err')
  } finally {
    adding.value = false
  }
}

async function save(r: Room) {
  const v = (editing.value[r.id] ?? '').trim()
  if (!v || v === r.name) {
    delete editing.value[r.id]
    return
  }
  try {
    await patchRoom(r.id, { name: v })
    toast('已改名')
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '改名失败', 'err')
  }
}

async function remove(r: Room) {
  if (!confirm(`删除房间「${r.name}」？其中设备的房间归属将被清空。`)) return
  try {
    await deleteRoom(r.id)
    toast('已删除')
    await load()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '删除失败', 'err')
  }
}

function editName(r: Room) {
  editing.value[r.id] = r.name
}

onMounted(load)
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <h1>房间</h1>
        <p>整理设备归属 · 删除房间后设备 room_id 置空</p>
      </div>
    </div>

    <div class="add-row">
      <input
        v-model="newName"
        class="input"
        type="text"
        placeholder="新房间名（如 客厅）"
        @keydown.enter="add"
      />
      <button type="button" class="btn btn-primary btn-sm" :disabled="adding || !newName.trim()" @click="add">
        添加
      </button>
    </div>

    <div v-if="loading" class="muted">加载中…</div>
    <div v-else-if="!rooms.length" class="muted">还没有房间</div>
    <div v-else class="room-list">
      <div v-for="r in rooms" :key="r.id" class="room-row">
        <span class="r-ico">▢</span>
        <input
          v-if="editing[r.id] !== undefined"
          v-model="editing[r.id]"
          class="input r-name-input"
          type="text"
          @keydown.enter="save(r)"
          @keydown.esc="delete editing[r.id]"
        />
        <span v-else class="r-name" @click="editName(r)">{{ r.name }}</span>
        <span class="r-count">{{ r.device_count ?? 0 }} 个设备</span>
        <div class="r-actions">
          <button v-if="editing[r.id] !== undefined" type="button" class="btn btn-sm btn-primary" @click="save(r)">
            保存
          </button>
          <button v-else type="button" class="btn btn-sm btn-outline" @click="editName(r)">改名</button>
          <button type="button" class="btn btn-sm btn-danger" @click="remove(r)">删除</button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.muted {
  color: hsl(var(--muted-foreground));
  padding: 1rem 0;
}
.add-row {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
}
.room-list {
  display: grid;
  gap: 0.5rem;
}
.room-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.7rem 0.9rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius, 0.625rem);
  background: hsl(var(--card));
}
.r-ico {
  font-size: 1.1rem;
}
.r-name {
  flex: 1;
  font-weight: 500;
  cursor: text;
}
.r-name-input {
  flex: 1;
  min-width: 8rem;
}
.r-count {
  font-size: 0.8rem;
  color: hsl(var(--muted-foreground));
  white-space: nowrap;
}
.r-actions {
  display: flex;
  gap: 0.4rem;
}
</style>