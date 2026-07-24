<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  activateHAInstance,
  createHAInstance,
  deleteHAInstance,
  fetchHAInstances,
  fetchHAStatus,
  fetchMeta,
  probeHAInstance,
  updateHAInstance,
} from '@/api'
import type { HAInstance, HAStatus } from '@/types/api'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'

const { user } = useAuth()
const { toast } = useToast()
const ha = ref<HAStatus | null>(null)
const meta = ref<{ auth_portal_url?: string; app_base_url?: string; service?: string } | null>(null)
const instances = ref<HAInstance[]>([])
const showForm = ref(false)
const form = ref({ name: '', base_url: '', token: '' })
const editing = ref<Record<string, { base_url: string; token: string }>>({})
const probing = ref<Record<string, boolean>>({})
const saving = ref(false)

async function refreshHA() {
  try {
    const [h, m, ins] = await Promise.all([fetchHAStatus(), fetchMeta(), fetchHAInstances()])
    ha.value = h.data
    meta.value = m.data
    instances.value = ins.data.items || []
  } catch {
    toast('加载设置失败', 'err')
  }
}

async function add() {
  if (!form.value.base_url.trim() || !form.value.token.trim()) {
    toast('base_url 与 token 必填', 'err')
    return
  }
  saving.value = true
  try {
    await createHAInstance({
      name: form.value.name.trim() || undefined,
      base_url: form.value.base_url.trim(),
      token: form.value.token.trim(),
    })
    toast('已添加 HA 实例')
    form.value = { name: '', base_url: '', token: '' }
    showForm.value = false
    await refreshHA()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '添加失败', 'err')
  } finally {
    saving.value = false
  }
}

async function saveEdit(id: string) {
  const e = editing.value[id]
  if (!e) return
  try {
    await updateHAInstance(id, {
      base_url: e.base_url || undefined,
      token: e.token || undefined,
    })
    toast('已更新')
    delete editing.value[id]
    await refreshHA()
  } catch (err) {
    toast(err instanceof ApiError ? err.message : '更新失败', 'err')
  }
}

async function remove(id: string) {
  if (!confirm('删除该 HA 实例？将回退到环境变量配置。')) return
  try {
    await deleteHAInstance(id)
    toast('已删除')
    await refreshHA()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '删除失败', 'err')
  }
}

async function probe(id: string) {
  probing.value[id] = true
  try {
    await probeHAInstance(id)
    toast('探测成功')
    await refreshHA()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '探测失败', 'err')
    await refreshHA()
  } finally {
    probing.value[id] = false
  }
}

function edit(ins: HAInstance) {
  editing.value[ins.id] = { base_url: '', token: '' }
}

async function activate(id: string) {
  try {
    await activateHAInstance(id)
    toast('已切换活跃实例')
    await refreshHA()
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '切换失败', 'err')
  }
}

onMounted(refreshHA)
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <h1>设置</h1>
        <p>账号与 Home Assistant 连接状态</p>
      </div>
    </div>

    <div class="settings-card">
      <div class="section-title" style="margin-top: 0">账号</div>
      <div class="row">
        <span class="k">名称</span>
        <span>{{ user?.name || '—' }}</span>
      </div>
      <div class="row">
        <span class="k">邮箱</span>
        <span>{{ user?.email || '—' }}</span>
      </div>
      <div class="row">
        <span class="k">家庭</span>
        <span>{{ user?.home?.name || '—' }}</span>
      </div>
      <p class="note">登录与登出由统一认证中心管理，本系统不提供单独登出。</p>
    </div>

    <div class="settings-card">
      <div class="section-title" style="margin-top: 0">Home Assistant</div>
      <div class="row">
        <span class="k">配置</span>
        <span>{{ ha?.configured ? '已配置' : '未配置' }}</span>
      </div>
      <div class="row">
        <span class="k">状态</span>
        <span :class="ha?.online ? 'ok' : 'bad'">
          {{ ha?.online ? '在线' : ha?.configured ? '离线' : '—' }}
        </span>
      </div>
      <div class="row">
        <span class="k">主机</span>
        <span>{{ ha?.base_url_host || '—' }}</span>
      </div>
      <div class="row">
        <span class="k">延迟</span>
        <span>{{ ha?.latency_ms != null ? `${ha.latency_ms} ms` : '—' }}</span>
      </div>
      <div class="row">
        <span class="k">Token</span>
        <span>仅服务端持有 · 不展示</span>
      </div>
      <p class="note">P0 通过环境变量配置 HA；DB 实例优先于环境变量。设置页不展示 Token。</p>
    </div>

    <div class="settings-card">
      <div class="section-title" style="margin-top: 0">HA 实例（DB 配置）</div>
      <div v-if="!instances.length" class="row"><span class="k">无</span><span>使用环境变量或点下方添加</span></div>
      <div v-for="ins in instances" :key="ins.id" class="row stack">
        <div class="ins-row">
          <span class="k">{{ ins.name }}</span>
          <span>{{ ins.base_url_host }}</span>
          <span :class="ins.is_active ? 'ok' : 'muted'">{{ ins.is_active ? '活跃' : '未启用' }}</span>
          <span v-if="ins.last_error" class="bad">{{ ins.last_error }}</span>
        </div>
        <div class="ins-actions">
          <button
            v-if="!ins.is_active"
            type="button"
            class="btn btn-sm btn-primary"
            @click="activate(ins.id)"
          >启用</button>
          <button type="button" class="btn btn-sm btn-outline" :disabled="probing[ins.id]" @click="probe(ins.id)">
            {{ probing[ins.id] ? '探测中…' : '测活' }}
          </button>
          <button type="button" class="btn btn-sm btn-outline" @click="edit(ins)">修改</button>
          <button type="button" class="btn btn-sm btn-danger" @click="remove(ins.id)">删除</button>
        </div>
        <div v-if="editing[ins.id]" class="ins-edit">
          <input v-model="editing[ins.id].base_url" class="input" type="text" placeholder="新 base_url（留空不改）" />
          <input v-model="editing[ins.id].token" class="input" type="password" placeholder="新 Token（留空不改）" />
          <button type="button" class="btn btn-sm btn-primary" @click="saveEdit(ins.id)">保存</button>
          <button type="button" class="btn btn-sm btn-ghost" @click="delete editing[ins.id]">取消</button>
        </div>
      </div>

      <div v-if="showForm" class="ins-form">
        <input v-model="form.name" class="input" type="text" placeholder="名称（默认 default）" />
        <input v-model="form.base_url" class="input" type="text" placeholder="HA base_url, 如 http://192.168.1.10:8123" />
        <input v-model="form.token" class="input" type="password" placeholder="Long-Lived Token" />
        <button type="button" class="btn btn-sm btn-primary" :disabled="saving" @click="add">添加</button>
        <button type="button" class="btn btn-sm btn-ghost" @click="showForm = false">取消</button>
      </div>
      <div v-else class="actions">
        <button type="button" class="btn btn-outline btn-sm" @click="showForm = true">添加 HA 实例</button>
      </div>
    </div>

    <div v-if="meta" class="settings-card">
      <div class="section-title" style="margin-top: 0">服务</div>
      <div class="row">
        <span class="k">名称</span>
        <span>{{ meta.service }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.settings-card {
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius, 0.625rem);
  background: hsl(var(--card));
  padding: 1rem;
  margin-bottom: 1rem;
}
.row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.4rem 0;
  font-size: 0.9rem;
  border-bottom: 1px solid hsl(var(--border) / 0.5);
}
.row:last-of-type {
  border-bottom: none;
}
.k {
  color: hsl(var(--muted-foreground));
}
.ok {
  color: hsl(var(--success));
}
.bad {
  color: hsl(var(--destructive));
}
.actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  flex-wrap: wrap;
}
.note {
  margin: 0.75rem 0 0;
  font-size: 0.8rem;
  color: hsl(var(--muted-foreground));
}
.row.stack {
  flex-direction: column;
  align-items: stretch;
  gap: 0.4rem;
}
.ins-row {
  display: flex;
  gap: 1rem;
  align-items: center;
}
.ins-actions,
.ins-edit,
.ins-form {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
  align-items: center;
}
.ins-edit .input,
.ins-form .input {
  min-width: 10rem;
}
.muted {
  color: hsl(var(--muted-foreground));
}
</style>
