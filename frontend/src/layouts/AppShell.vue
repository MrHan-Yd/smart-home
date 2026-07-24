<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useTheme } from '@/composables/useTheme'
import { useHA } from '@/composables/useHA'
import AppToast from '@/components/AppToast.vue'

const route = useRoute()
const { user, loadMe, logoutLocal } = useAuth()
const { theme, toggleTheme } = useTheme()
const { status: ha, refresh: refreshHA } = useHA()

const portalUrl = import.meta.env.VITE_AUTH_PORTAL_URL || 'http://127.0.0.1:5173'

const haPillClass = computed(() => {
  if (!ha.value?.configured) return 'offline'
  return ha.value.online ? '' : 'offline'
})

const haPillText = computed(() => {
  if (!ha.value) return 'HA …'
  if (!ha.value.configured) return 'HA 未配置'
  return ha.value.online ? 'HA 已连接' : 'HA 离线'
})

const nav = [
  { to: '/', name: 'overview', label: '总览', ico: '⌂' },
  { to: '/devices', name: 'devices', label: '设备', ico: '▦' },
  { to: '/add', name: 'add', label: '添加', ico: '＋' },
  { to: '/settings', name: 'settings', label: '设置', ico: '⚙' },
]

function navActive(path: string) {
  if (path === '/') return route.path === '/'
  return route.path === path || route.path.startsWith(path + '/')
}

onMounted(async () => {
  if (!user.value) {
    try {
      await loadMe()
    } catch {
      /* guard */
    }
  }
  await refreshHA()
})
</script>

<template>
  <div class="view-app">
    <header class="topbar">
      <div class="topbar-brand">
        <span class="logo-mark sm">SH</span>
        <span class="topbar-title">Home Hub</span>
        <span class="ha-pill" :class="haPillClass" :title="ha?.message || ''">
          <span class="dot" />
          <span>{{ haPillText }}</span>
        </span>
      </div>
      <div class="topbar-actions">
        <a
          :href="portalUrl"
          class="btn btn-outline btn-sm btn-portal"
          title="切换系统"
          aria-label="切换系统"
        >
          <span class="portal-icon" aria-hidden="true">⇄</span>
          <span class="portal-label">切换系统</span>
        </a>
        <span v-if="user" class="user-chip hide-sm">
          <span class="avatar">{{ (user.name || user.email || '?').slice(0, 1) }}</span>
          <span>{{ user.name || user.email }}</span>
        </span>
        <button type="button" class="btn btn-ghost btn-sm" title="登出" @click="logoutLocal">
          <span class="btn-label">登出</span>
          <span class="btn-label-short" aria-hidden="true">⎋</span>
        </button>
        <button type="button" class="btn btn-icon btn-outline" title="主题" @click="toggleTheme">
          {{ theme === 'dark' ? '☀' : '☾' }}
        </button>
      </div>
    </header>

    <div class="app-body">
      <aside class="sidebar">
        <RouterLink
          v-for="n in nav"
          :key="n.to"
          :to="n.to"
          class="nav-item"
          :class="{ active: navActive(n.to) }"
        >
          <span class="nav-ico">{{ n.ico }}</span>{{ n.label }}
        </RouterLink>
      </aside>

      <main class="main">
        <RouterView />
      </main>
    </div>

    <nav class="bottom-nav">
      <RouterLink
        v-for="n in nav"
        :key="n.to"
        :to="n.to"
        :class="{ active: navActive(n.to) }"
      >
        <span class="nav-ico">{{ n.ico }}</span>
        {{ n.label }}
      </RouterLink>
    </nav>

    <AppToast />
  </div>
</template>

<style scoped>
.nav-item {
  text-decoration: none;
}
.bottom-nav a {
  flex: 1;
  border: none;
  background: transparent;
  color: hsl(var(--muted-foreground));
  font-size: 0.65rem;
  padding: 0.35rem 0.15rem;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.15rem;
  text-decoration: none;
}
.bottom-nav a.active {
  color: hsl(var(--primary));
  font-weight: 600;
}
.bottom-nav .nav-ico {
  font-size: 1.1rem;
}
</style>
