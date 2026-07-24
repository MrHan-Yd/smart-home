import { ref, readonly } from 'vue'
import { fetchMe, logout as apiLogout } from '@/api'
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

  function goLogin(returnTo?: string) {
    const path = returnTo ?? window.location.pathname + window.location.search
    window.location.href = `/oauth/login?return_to=${encodeURIComponent(path || '/')}`
  }

  async function logoutLocal() {
    user.value = null
    try {
      await apiLogout({ global: false })
    } catch {
      /* ignore */
    }
    window.location.replace('/login')
  }

  async function logoutGlobal() {
    user.value = null
    let logoutUrl = import.meta.env.VITE_AUTH_PORTAL_URL || 'http://127.0.0.1:5173'
    try {
      const res = await apiLogout({ global: true })
      if (res.data.logout_url) logoutUrl = res.data.logout_url
    } catch {
      /* ignore */
    }
    window.location.replace(logoutUrl)
  }

  return {
    user: readonly(user),
    loading: readonly(loading),
    checked: readonly(checked),
    loadMe,
    goLogin,
    logoutLocal,
    logoutGlobal,
    logout: logoutLocal,
  }
}
