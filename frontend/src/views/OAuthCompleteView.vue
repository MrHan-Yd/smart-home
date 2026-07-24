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
        <span class="logo-mark">SH</span>
      </div>
      <p class="login-lead">{{ status }}</p>
    </div>
  </div>
</template>
