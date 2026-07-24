import { ref, readonly } from 'vue'
import { fetchHAStatus } from '@/api'
import type { HAStatus } from '@/types/api'

const status = ref<HAStatus | null>(null)
const loading = ref(false)

export function useHA() {
  async function refresh() {
    loading.value = true
    try {
      const res = await fetchHAStatus()
      status.value = res.data
      return res.data
    } catch {
      status.value = {
        configured: false,
        online: false,
        message: 'status unavailable',
      }
      return status.value
    } finally {
      loading.value = false
    }
  }

  return {
    status: readonly(status),
    loading: readonly(loading),
    refresh,
  }
}
