export type Home = {
  id: string
  name: string
}

export type User = {
  id: string
  sub: string
  email: string
  name: string
  home?: Home
}

export type HAStatus = {
  configured: boolean
  online: boolean
  base_url_host?: string
  latency_ms?: number | null
  message?: string
}

export type DeviceView = {
  id: string
  entity_id: string
  domain: string
  name: string
  room_id?: string | null
  room_name?: string | null
  favorite: boolean
  hidden: boolean
  state: string
  available: boolean
  primary_display: string
  capabilities: string[]
  control_level: string
  attributes?: Record<string, unknown>
  ha?: {
    last_changed?: string
    last_updated?: string
  }
}

export type DiscoverEntity = {
  entity_id: string
  domain: string
  name: string
  state: string
  available: boolean
  already_added: boolean
  capabilities: string[]
  control_level: string
  device_class?: unknown
  area?: unknown
}
