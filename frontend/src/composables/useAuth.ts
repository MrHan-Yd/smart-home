import { ref, readonly } from 'vue'
import { fetchMe } from '@/api'
import type { User } from '@/types/api'
import { ApiError } from '@/lib/http'

const user = ref<User | null>(null)
const loading = ref(false)
const checked = ref(false)

export function useAuth() {
  async function loadMe() {
    loading.value = true
    try {
      const res = await fetchMe()
      user.value = res.data
      return res.data
    } catch (e) {
      user.value = null
      if (!(e instanceof ApiError && e.httpStatus === 401)) {
        console.error(e)
      }
      throw e
    } finally {
      loading.value = false
      checked.value = true
    }
  }

  // 未登录：直接跳转统一认证中心（authorize），由 IdP 处理登录后回跳。
  // 本系统不维护本地登录页，也不提供登出（清 SSO 会话由认证中心负责）。
  function goLogin(returnTo?: string) {
    const path = returnTo ?? window.location.pathname + window.location.search
    window.location.href = `/oauth/login?return_to=${encodeURIComponent(path || '/')}`
  }

  return {
    user: readonly(user),
    loading: readonly(loading),
    checked: readonly(checked),
    loadMe,
    goLogin,
  }
}