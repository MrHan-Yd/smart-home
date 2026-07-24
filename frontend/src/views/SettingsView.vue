<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchHAStatus, fetchMeta } from '@/api'
import type { HAStatus } from '@/types/api'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'

const { user, logoutLocal, logoutGlobal } = useAuth()
const { toast } = useToast()
const ha = ref<HAStatus | null>(null)
const meta = ref<{ auth_portal_url?: string; app_base_url?: string; service?: string } | null>(null)

onMounted(async () => {
  try {
    const [h, m] = await Promise.all([fetchHAStatus(), fetchMeta()])
    ha.value = h.data
    meta.value = m.data
  } catch {
    toast('加载设置失败', 'err')
  }
})
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
      <div class="actions">
        <button type="button" class="btn btn-outline btn-sm" @click="logoutLocal">退出本系统</button>
        <button type="button" class="btn btn-ghost btn-sm" @click="logoutGlobal">全局登出</button>
      </div>
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
      <p class="note">P0 通过环境变量配置 HA；设置页不修改 Token。</p>
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
</style>
