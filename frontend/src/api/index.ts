import { apiDelete, apiFetch, apiGet, apiPatch, apiPost } from '@/lib/http'
import type { DeviceView, DiscoverEntity, HAStatus, User } from '@/types/api'

export function fetchMe() {
  return apiGet<User>('/api/v1/me', { skipAuthRedirect: true })
}

export function logout(opts?: { global?: boolean }) {
  const q = opts?.global ? '?global=1' : ''
  return apiFetch<{ ok: boolean; logout_url?: string }>(
    `/oauth/logout${q}`,
    { method: 'POST' },
    { skipAuthRedirect: true },
  )
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
