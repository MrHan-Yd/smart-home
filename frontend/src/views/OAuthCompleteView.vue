<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const status = ref('正在完成登录…')

onMounted(async () => {
  // Vite 将 /oauth/complete 代理到后端，若落到 Vue 路由则说明未走代理
  // 开发期后端会 302 到前端同源 /oauth/complete?ticket=… 再由后端写 Cookie
  // 若 ticket 已在 query，整页导航到同源代理路径
  const ticket = route.query.ticket as string | undefined
  if (ticket) {
    status.value = '写入会话…'
    window.location.replace(`/oauth/complete?ticket=${encodeURIComponent(ticket)}`)
    return
  }
  status.value = '登录完成，进入系统…'
  await router.replace('/')
})
</script>

<template>
  <div class="view-login">
    <div class="login-panel" style="text-align: center">
      <div class="login-logo" style="justify-content: center">
        <span class="logo-mark logo-svg" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M4 11.5 12 4l8 7.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M6 10.5V19a1 1 0 0 0 1 1h10a1 1 0 0 0 1-1v-8.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
            <circle cx="12" cy="14.2" r="1.5" fill="currentColor"/>
            <path d="M12 15.7V18" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>
          </svg>
        </span>
      </div>
      <p class="login-lead">{{ status }}</p>
    </div>
  </div>
</template>
