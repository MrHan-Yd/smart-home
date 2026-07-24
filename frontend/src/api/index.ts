import { apiDelete, apiFetch, apiGet, apiPatch, apiPost, apiPut } from '@/lib/http'
import type { DeviceView, DiscoverEntity, HAInstance, HAStatus, Room, Scenario, User } from '@/types/api'

export function fetchMe() {
  return apiGet<User>('/api/v1/me', { skipAuthRedirect: true })
}

export function fetchMeta() {
  return apiGet<{
    service: string
    auth_portal_url: string
    app_base_url: string
    session_ttl_sec?: number
    session_idle_sec?: number
  }>('/api/v1/meta', { skipAuthRedirect: true })
}

export function fetchHAStatus() {
  return apiGet<HAStatus>('/api/v1/ha/status', { skipAuthRedirect: true })
}

export function fetchDevices(params?: {
  q?: string
  domain?: string
  favorite?: boolean
  controllable?: boolean
  include_hidden?: boolean
}) {
  const qs = new URLSearchParams()
  if (params?.q) qs.set('q', params.q)
  if (params?.domain) qs.set('domain', params.domain)
  if (params?.favorite) qs.set('favorite', 'true')
  if (params?.controllable) qs.set('controllable', 'true')
  if (params?.include_hidden) qs.set('include_hidden', 'true')
  const q = qs.toString()
  return apiGet<{ items: DeviceView[] }>(`/api/v1/devices${q ? `?${q}` : ''}`)
}

export function fetchDevice(id: string) {
  return apiGet<DeviceView>(`/api/v1/devices/${id}`)
}

export function createDevice(body: { entity_id: string; name?: string }) {
  return apiPost<DeviceView>('/api/v1/devices', body)
}

export function createCompositeDevice(body: {
  entity_ids: string[]
  primary_entity_id?: string
  name?: string
  room_id?: string
}) {
  return apiPost<DeviceView>('/api/v1/devices/composite', body)
}

export function batchCreateDevices(entity_ids: string[]) {
  return apiPost<{
    created: { id: string; entity_id: string; domain: string }[]
    skipped: { entity_id: string; reason: string }[]
  }>('/api/v1/devices/batch', { entity_ids })
}

export function patchDevice(
  id: string,
  body: {
    name?: string
    room_id?: string | null
    favorite?: boolean
    hidden?: boolean
    sort_order?: number
  },
) {
  return apiPatch<DeviceView>(`/api/v1/devices/${id}`, body)
}

export function deleteDevice(id: string) {
  return apiDelete<{ ok: boolean }>(`/api/v1/devices/${id}`)
}

export function deviceAction(
  id: string,
  action: string,
  params?: Record<string, unknown>,
) {
  const key =
    typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random()}`
  return apiPost<DeviceView>(
    `/api/v1/devices/${id}/actions`,
    { action, params },
    { 'Idempotency-Key': key },
  )
}

export function discoverEntities(params?: {
  q?: string
  domain?: string
  only_new?: boolean
  only_controllable?: boolean
  page?: number
  page_size?: number
}) {
  const qs = new URLSearchParams()
  if (params?.q) qs.set('q', params.q)
  if (params?.domain) qs.set('domain', params.domain)
  if (params?.only_new === false) qs.set('only_new', 'false')
  if (params?.only_controllable) qs.set('only_controllable', 'true')
  if (params?.page) qs.set('page', String(params.page))
  if (params?.page_size) qs.set('page_size', String(params.page_size))
  const q = qs.toString()
  return apiGet<{
    items: DiscoverEntity[]
    page: number
    page_size: number
    total: number
  }>(`/api/v1/discover/entities${q ? `?${q}` : ''}`)
}

export function fetchRooms() {
  return apiGet<{ items: Room[] }>('/api/v1/rooms')
}

export function createRoom(body: { name: string; sort_order?: number }) {
  return apiPost<{ id: string; name: string; sort_order: number }>('/api/v1/rooms', body)
}

export function patchRoom(id: string, body: { name?: string; sort_order?: number }) {
  return apiPatch<{ id: string; name: string; sort_order: number }>(`/api/v1/rooms/${id}`, body)
}

export function deleteRoom(id: string) {
  return apiDelete<{ ok: boolean }>(`/api/v1/rooms/${id}`)
}

export type HistoryPoint = { t: string; state: string; num: number | null }
export type HistoryResp = {
  entity_id: string
  points: HistoryPoint[]
  count: number
}

export function fetchDeviceHistory(id: string, range: '24h' | '7d') {
  const now = new Date()
  const end = now.toISOString()
  const start = new Date(now.getTime() - (range === '7d' ? 7 : 1) * 86400000).toISOString()
  const qs = new URLSearchParams({ start, end, significant_only: 'true' })
  return apiGet<HistoryResp>(`/api/v1/devices/${id}/history?${qs.toString()}`)
}

export type ActionLogItem = {
  id: string
  device_id: string
  entity_id: string
  action: string
  success: boolean
  error_message: string
  ha_domain: string
  ha_service: string
  duration_ms: number
  created_at: string
}

export function fetchActionLogs(limit = 50, offset = 0) {
  const qs = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  return apiGet<{ items: ActionLogItem[] }>(`/api/v1/action-logs?${qs.toString()}`)
}

export function fetchHAInstances() {
  return apiGet<{ items: HAInstance[] }>('/api/v1/ha/instances')
}

export function createHAInstance(body: { name?: string; base_url: string; token: string }) {
  return apiPost<{ id: string; name: string; base_url_host: string }>('/api/v1/ha/instances', body)
}

export function updateHAInstance(id: string, body: { base_url?: string; token?: string }) {
  return apiPatch<{ id: string }>(`/api/v1/ha/instances/${id}`, body)
}

export function deleteHAInstance(id: string) {
  return apiDelete<{ ok: boolean }>(`/api/v1/ha/instances/${id}`)
}

export function probeHAInstance(id: string) {
  return apiPost<{ ok: boolean }>(`/api/v1/ha/instances/${id}/probe`, undefined)
}

export function activateHAInstance(id: string) {
  return apiPost<{ ok: boolean }>(`/api/v1/ha/instances/${id}/activate`, undefined)
}

export type AnalyticsSummary = {
  activation_count: number
  runtime_hours: number
  energy_kwh: number | null
  avg_temperature: number | null
  avg_humidity: number | null
  on_count: number
  online_count: number
}
export type AnalyticsRuntime = {
  date: string
  device_id: string
  hours: number
  on_count: number
  energy_kwh: number | null
}
export type AnalyticsRanking = { device_id: string; name: string; hours: number; on_count: number }
export type AnalyticsTypeMix = { domain: string; count: number; hours: number }
export type AnalyticsHeatmap = { entity_id: string; hour: number; count: number }
export type AnalyticsEnv = { device_id: string; entity_id: string; series: { date: string; value: number | null }[] }

function rng(r: string) {
  const qs = new URLSearchParams({ range: r })
  return qs.toString()
}
export function fetchAnalyticsSummary(range = '7d') {
  return apiGet<AnalyticsSummary>(`/api/v1/analytics/summary?${rng(range)}`)
}
export function fetchAnalyticsRuntime(range = '7d') {
  return apiGet<{ items: AnalyticsRuntime[] }>(`/api/v1/analytics/runtime?${rng(range)}`)
}
export function fetchAnalyticsRanking(range = '7d', limit = 10) {
  return apiGet<{ items: AnalyticsRanking[] }>(`/api/v1/analytics/ranking?${rng(range)}&limit=${limit}`)
}
export function fetchAnalyticsTypeMix(range = '7d') {
  return apiGet<{ items: AnalyticsTypeMix[] }>(`/api/v1/analytics/type-mix?${rng(range)}`)
}
export function fetchAnalyticsHeatmap(range = '7d') {
  return apiGet<{ items: AnalyticsHeatmap[] }>(`/api/v1/analytics/heatmap?${rng(range)}`)
}
export function fetchAnalyticsEnvironment(range = '7d') {
  return apiGet<{ items: AnalyticsEnv[] }>(`/api/v1/analytics/environment?${rng(range)}`)
}

// ---- scenarios ----

export function fetchScenarios() {
  return apiGet<{ items: Scenario[] }>('/api/v1/scenarios')
}

export function fetchScenario(id: string) {
  return apiGet<Scenario>(`/api/v1/scenarios/${id}`)
}

export function createScenario(body: {
  name: string
  icon?: string
  room_id?: string | null
  steps?: Array<{ device_id: string; action: string; params?: Record<string, unknown>; delay_ms?: number }>
}) {
  return apiPost<Scenario>('/api/v1/scenarios', body)
}

export function patchScenario(id: string, body: { name?: string; icon?: string; room_id?: string | null; sort_order?: number; enabled?: boolean }) {
  return apiPatch<Scenario>(`/api/v1/scenarios/${id}`, body)
}

export function replaceScenarioSteps(id: string, steps: Array<{ device_id: string; action: string; params?: Record<string, unknown>; delay_ms?: number }>) {
  return apiPut<{ ok: boolean }>(`/api/v1/scenarios/${id}/steps`, { steps })
}

export function deleteScenario(id: string) {
  return apiDelete<{ ok: boolean }>(`/api/v1/scenarios/${id}`)
}

export type ScenarioRunResult = { device_id: string; action: string; success: boolean; error?: string }

export function runScenario(id: string) {
  return apiPost<{ results: ScenarioRunResult[] }>(`/api/v1/scenarios/${id}/run`, undefined)
}
