import { readonly, ref } from 'vue'
import type { DeviceView } from '@/types/api'

// Module-level WS connection + subscriber registry.
const ws = ref<WebSocket | null>(null)
const connected = ref(false)
let reconnectTimer: number | undefined
let pingTimer: number | undefined
const subscribers = new Map<number, (d: DeviceView) => void>()
let nextSub = 1
let manualClose = false

function connect() {
  if (ws.value && (ws.value.readyState === WebSocket.OPEN || ws.value.readyState === WebSocket.CONNECTING)) {
    return
  }
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const url = `${proto}://${window.location.host}/api/v1/ws`
  manualClose = false
  try {
    const c = new WebSocket(url)
    ws.value = c
    c.onopen = () => {
      connected.value = true
      startPing()
    }
    c.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.type === 'state_changed' && msg.device) {
          notify(msg.device as DeviceView)
        }
      } catch {
        /* ignore */
      }
    }
    c.onclose = () => {
      connected.value = false
      stopPing()
      if (!manualClose) scheduleReconnect()
    }
    c.onerror = () => {
      c.close()
    }
  } catch {
    connected.value = false
    scheduleReconnect()
  }
}

function scheduleReconnect() {
  if (reconnectTimer) return
  reconnectTimer = window.setTimeout(() => {
    reconnectTimer = undefined
    connect()
  }, 5000)
}

function startPing() {
  stopPing()
  pingTimer = window.setInterval(() => {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify({ type: 'ping' }))
    }
  }, 30000)
}

function stopPing() {
  if (pingTimer) {
    window.clearInterval(pingTimer)
    pingTimer = undefined
  }
}

function notify(d: DeviceView) {
  subscribers.forEach((fn) => fn(d))
}

export function useWS() {
  function subscribe(fn: (d: DeviceView) => void): () => void {
    const id = nextSub++
    subscribers.set(id, fn)
    if (!ws.value || ws.value.readyState === WebSocket.CLOSED) {
      connect()
    }
    return () => {
      subscribers.delete(id)
    }
  }

  function disconnect() {
    manualClose = true
    stopPing()
    if (reconnectTimer) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = undefined
    }
    ws.value?.close()
    ws.value = null
    connected.value = false
  }

  return { connected: readonly(connected), subscribe, disconnect }
}