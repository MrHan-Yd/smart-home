<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiGet } from '@/api/http'

type Meta = {
  service: string
  auth_portal_url: string
  app_base_url: string
}

type HaStatus = {
  configured: boolean
  online: boolean
  base_url_host: string
  latency_ms: number | null
  message: string
}

const meta = ref<Meta | null>(null)
const ha = ref<HaStatus | null>(null)
const error = ref('')
const loading = ref(true)

const portalUrl = import.meta.env.VITE_AUTH_PORTAL_URL || 'http://127.0.0.1:5173'

onMounted(async () => {
  try {
    const [m, h] = await Promise.all([
      apiGet<Meta>('/api/v1/meta'),
      apiGet<HaStatus>('/api/v1/ha/status'),
    ])
    if (m.code === 0) meta.value = m.data ?? null
    if (h.code === 0) ha.value = h.data ?? null
    if (m.code !== 0) error.value = m.message
  } catch (e) {
    error.value = e instanceof Error ? e.message : '后端未连通（请先启动 :3002）'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="min-h-dvh px-4 py-8 sm:px-8">
    <header class="mx-auto flex max-w-3xl items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <span
          class="grid h-8 w-8 place-items-center rounded-lg bg-gradient-to-br from-sky-500 to-emerald-500 text-xs font-bold text-white"
        >
          SH
        </span>
        <div>
          <h1 class="text-base font-semibold tracking-tight">Home Hub</h1>
          <p class="text-xs text-[hsl(var(--muted-foreground))]">智能家居子系统 · 骨架页</p>
        </div>
      </div>
      <a
        :href="portalUrl"
        title="切换系统"
        class="inline-flex min-h-8 min-w-8 items-center justify-center rounded-md border border-[hsl(var(--border))] px-2 text-xs no-underline hover:bg-[hsl(var(--muted))] sm:min-w-0 sm:px-3"
      >
        <span class="sm:hidden" aria-hidden="true">⇄</span>
        <span class="hidden sm:inline">切换系统</span>
      </a>
    </header>

    <main class="mx-auto mt-8 max-w-3xl space-y-4">
      <section
        class="rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--card))] p-5 shadow-sm"
      >
        <h2 class="text-sm font-semibold">工程状态</h2>
        <p class="mt-1 text-sm text-[hsl(var(--muted-foreground))]">
          前端 Vue 3 + Vite · 后端 Go :3002 · 设计见
          <code class="text-xs">docs/</code>
        </p>

        <div v-if="loading" class="mt-4 text-sm text-[hsl(var(--muted-foreground))]">加载中…</div>
        <div v-else-if="error" class="mt-4 rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-400">
          {{ error }}
        </div>
        <dl v-else class="mt-4 grid gap-3 text-sm sm:grid-cols-2">
          <div class="rounded-lg border border-[hsl(var(--border))] p-3">
            <dt class="text-xs text-[hsl(var(--muted-foreground))]">服务</dt>
            <dd class="mt-1 font-medium">{{ meta?.service ?? '—' }}</dd>
          </div>
          <div class="rounded-lg border border-[hsl(var(--border))] p-3">
            <dt class="text-xs text-[hsl(var(--muted-foreground))]">HA</dt>
            <dd class="mt-1 font-medium">
              <span v-if="!ha?.configured">未配置</span>
              <span v-else-if="ha.online" class="text-emerald-400">在线</span>
              <span v-else class="text-amber-400">离线</span>
              <span
                v-if="ha?.base_url_host"
                class="mt-0.5 block text-xs text-[hsl(var(--muted-foreground))]"
              >
                {{ ha.base_url_host }}
                <template v-if="ha.latency_ms != null"> · {{ ha.latency_ms }}ms</template>
              </span>
            </dd>
          </div>
        </dl>
      </section>

      <section
        class="rounded-xl border border-dashed border-[hsl(var(--border))] p-5 text-sm text-[hsl(var(--muted-foreground))]"
      >
        <p class="font-medium text-[hsl(var(--foreground))]">下一步</p>
        <ul class="mt-2 list-inside list-disc space-y-1">
          <li>后端 OAuth（对齐统一认证中心）</li>
          <li>发现 / 设备 / actions API</li>
          <li>按 ui-prototype 铺页面</li>
        </ul>
      </section>
    </main>
  </div>
</template>
