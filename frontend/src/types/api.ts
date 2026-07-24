export type Home = {
  id: string
  name: string
}

export type Room = {
  id: string
  home_id: string
  name: string
  sort_order: number
  ha_area_id?: string | null
  device_count?: number
  created_at?: string
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
  active_instance_id?: string | null
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
  meta?: { kind?: string }
  entity_ids?: string[]
  members?: { entity_id: string; state: string; available: boolean }[]
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

export type HAInstance = {
  id: string
  name: string
  base_url_host: string
  is_active: boolean
  last_ok_at?: string | null
  last_error?: string | null
  has_token: boolean
}

export type ScenarioStep = {
  id?: string
  scenario_id?: string
  sort_order: number
  device_id: string
  action: string
  params?: Record<string, unknown>
  delay_ms: number
}

export type Scenario = {
  id: string
  home_id: string
  user_id: string
  name: string
  icon?: string | null
  room_id?: string | null
  sort_order: number
  enabled: boolean
  last_run_at?: string | null
  run_count: number
  created_at: string
  updated_at: string
  steps?: ScenarioStep[]
}
