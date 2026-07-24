<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useTheme } from '@/composables/useTheme'

const { user, loadMe, goLogin } = useAuth()
const { theme, toggleTheme } = useTheme()
const route = useRoute()
const router = useRouter()
const status = ref('正在检查登录状态…')
const ready = ref(false)

onMounted(async () => {
  try {
    await loadMe()
    if (user.value) {
      status.value = '已登录，正在进入…'
      await router.replace((route.query.return_to as string) || '/')
      return
    }
  } catch {
    /* 未登录 */
  }
  ready.value = true
  status.value = ''
})

function onSSO() {
  status.value = '正在跳转统一认证中心…'
  ready.value = false
  const returnTo = (route.query.return_to as string) || '/'
  goLogin(returnTo)
}
</script>

<template>
  <div class="view-login">
    <button type="button" class="theme-fab" title="切换主题" @click="toggleTheme">
      <span v-if="theme === 'dark'">☀</span>
      <span v-else>☾</span>
    </button>

    <div class="login-panel">
      <div class="login-logo">
        <span class="logo-mark">SH</span>
        <span class="logo-text">Home Hub</span>
      </div>
      <h1 class="login-title">你的设备，一张网管住</h1>
      <p class="login-lead">对接 Home Assistant · 统一认证登录 · 监控、控制与使用分析。</p>
      <ul class="login-features">
        <li>
          <span class="feat-icon">01</span>
          <div>
            <strong>统一认证 SSO</strong>
            <span>本系统不存密码，票只来自认证中心</span>
          </div>
        </li>
        <li>
          <span class="feat-icon">02</span>
          <div>
            <strong>从 HA 添加设备</strong>
            <span>多类型发现与纳入，能力驱动控制</span>
          </div>
        </li>
        <li>
          <span class="feat-icon">03</span>
          <div>
            <strong>监控 · 控制</strong>
            <span>状态墙、开关与设备详情</span>
          </div>
        </li>
      </ul>
      <button
        type="button"
        class="btn btn-primary btn-lg btn-block"
        :disabled="!ready && !status"
        @click="onSSO"
      >
        {{ ready ? '继续使用统一认证中心' : status || '请稍候…' }}
      </button>
      <p class="login-footnote">OAuth · Cookie 会话 · 不直连 Home Assistant</p>
    </div>
  </div>
</template>
