<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchActionLogs } from '@/api'
import type { ActionLogItem } from '@/api'
import { useToast } from '@/composables/useToast'
import { ApiError } from '@/lib/http'

const { toast } = useToast()
const loading = ref(true)
const logs = ref<ActionLogItem[]>([])

function fmtTime(t: string): string {
  try {
    return new Date(t).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return t
  }
}

async function load() {
  loading.value = true
  try {
    const res = await fetchActionLogs(100)
    logs.value = res.data.items || []
  } catch (e) {
    toast(e instanceof ApiError ? e.message : '加载失败', 'err')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="page active">
    <div class="page-header">
      <div>
        <h1>操作日志</h1>
        <p>设备控制审计 · 最近 100 条</p>
      </div>
      <div class="page-actions">
        <button type="button" class="btn btn-outline btn-sm" :disabled="loading" @click="load">
          刷新
        </button>
      </div>
    </div>

    <div v-if="loading" class="muted">加载中…</div>
    <div v-else-if="!logs.length" class="muted">暂无操作记录</div>
    <div v-else class="log-table-wrap">
      <table class="log-table">
        <thead>
          <tr>
            <th>时间</th>
            <th>设备</th>
            <th>动作</th>
            <th>HA 服务</th>
            <th>结果</th>
            <th>耗时</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="l in logs" :key="l.id" :class="{ fail: !l.success }">
            <td class="mono">{{ fmtTime(l.created_at) }}</td>
            <td class="mono">{{ l.entity_id }}</td>
            <td>{{ l.action }}</td>
            <td class="mono">{{ l.ha_domain }}.{{ l.ha_service }}</td>
            <td :class="l.success ? 'ok' : 'bad'">
              {{ l.success ? '成功' : `失败${l.error_message ? ': ' + l.error_message : ''}` }}
            </td>
            <td class="mono">{{ l.duration_ms }}ms</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<style scoped>
.muted {
  color: hsl(var(--muted-foreground));
  padding: 1rem 0;
}
.log-table-wrap {
  overflow-x: auto;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius, 0.625rem);
  background: hsl(var(--card));
}
.log-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.82rem;
}
.log-table th,
.log-table td {
  text-align: left;
  padding: 0.5rem 0.7rem;
  border-bottom: 1px solid hsl(var(--border) / 0.5);
  white-space: nowrap;
}
.log-table th {
  color: hsl(var(--muted-foreground));
  font-weight: 600;
  background: hsl(var(--muted) / 0.5);
}
.log-table tr.fail td {
  background: hsl(var(--danger) / 0.06);
}
.mono {
  font-family: ui-monospace, monospace;
  font-size: 0.78rem;
}
.ok {
  color: hsl(var(--success));
}
.bad {
  color: hsl(var(--destructive));
}
</style>