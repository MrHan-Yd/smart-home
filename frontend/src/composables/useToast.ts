import { ref } from 'vue'

export type ToastItem = {
  id: number
  message: string
  type: 'ok' | 'err'
}

const items = ref<ToastItem[]>([])
let seq = 0

export function useToast() {
  function toast(message: string, type: 'ok' | 'err' = 'ok') {
    const id = ++seq
    items.value = [...items.value, { id, message, type }]
    window.setTimeout(() => {
      items.value = items.value.filter((t) => t.id !== id)
    }, 2400)
  }

  return { items, toast }
}
