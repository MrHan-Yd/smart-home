import type { DeviceView } from '@/types/api'

const domainIcon: Record<string, string> = {
  light: '💡',
  switch: '⏻',
  input_boolean: '⏻',
  sensor: '⌀',
  binary_sensor: '◉',
  climate: '❄',
  cover: '▥',
  fan: '֍',
  media_player: '▶',
  lock: '🔒',
  vacuum: '⊙',
  scene: '✦',
  script: '▸',
  button: '●',
  group: '⛦',
}

export function domainEmoji(domain: string) {
  return domainIcon[domain] || '▦'
}

// composite device helpers
export function isComposite(d: DeviceView): boolean {
  return d.meta?.kind === 'composite' || (d.entity_ids?.length ?? 0) > 1
}

export function compositeMemberCount(d: DeviceView): number {
  return d.members?.length || d.entity_ids?.length || 1
}

// any member on → on
export function compositeIsOn(d: DeviceView): boolean {
  if (d.members?.length) {
    return d.members.some((m) => m.state === 'on' || m.state === 'open' || m.state === 'playing')
  }
  return d.state === 'on' || d.state === 'open' || d.state === 'playing'
}

export function canToggle(d: DeviceView) {
  if (isComposite(d)) return true
  return (
    d.control_level !== 'read_only' &&
    (d.capabilities?.includes('on_off') ||
      ['light', 'switch', 'input_boolean', 'fan'].includes(d.domain))
  )
}

export function canRun(d: DeviceView): boolean {
  return d.domain === 'scene' || d.domain === 'script' || d.domain === 'button' || d.domain === 'input_button'
}

export function isOn(d: DeviceView) {
  if (isComposite(d)) return compositeIsOn(d)
  return d.state === 'on' || d.state === 'open' || d.state === 'playing' || d.state === 'home'
}

export function toggleAction(d: DeviceView): 'turn_on' | 'turn_off' {
  return isOn(d) ? 'turn_off' : 'turn_on'
}

// ---- P1 control attribute helpers ----

function asNum(v: unknown): number | null {
  if (v === null || v === undefined) return null
  const n = typeof v === 'string' ? parseFloat(v) : (v as number)
  return Number.isFinite(n) ? n : null
}

export function brightnessPct(d: DeviceView): number | null {
  if (d.domain !== 'light') return null
  const b = asNum(d.attributes?.brightness)
  if (b === null) return null
  return Math.round((b / 255) * 100)
}

export function colorTempKelvin(d: DeviceView): number | null {
  return asNum(d.attributes?.color_temp_kelvin) ?? asNum(d.attributes?.color_temp)
}

export function coverPosition(d: DeviceView): number | null {
  if (d.domain !== 'cover') return null
  return asNum(d.attributes?.position)
}

export function currentTemp(d: DeviceView): number | null {
  return asNum(d.attributes?.current_temperature)
}

export function targetTemp(d: DeviceView): number | null {
  return asNum(d.attributes?.temperature)
}

export function hvacModes(d: DeviceView): string[] {
  const v = d.attributes?.hvac_modes
  if (!Array.isArray(v)) return []
  return v.filter((x): x is string => typeof x === 'string')
}

export function hvacMode(d: DeviceView): string {
  return typeof d.attributes?.hvac_mode === 'string' ? (d.attributes.hvac_mode as string) : ''
}

export function isButton(d: DeviceView): boolean {
  return d.domain === 'button' || d.domain === 'input_button'
}

export function isScene(d: DeviceView): boolean {
  return d.domain === 'scene' || d.domain === 'script'
}

export function isMediaPlayer(d: DeviceView): boolean {
  return d.domain === 'media_player'
}

export function isVacuum(d: DeviceView): boolean {
  return d.domain === 'vacuum'
}

export function volumeLevel(d: DeviceView): number | null {
  if (d.domain !== 'media_player') return null
  const v = asNum(d.attributes?.volume_level)
  if (v === null) return null
  return Math.round(v * 100)
}

export function hasCap(d: DeviceView, cap: string): boolean {
  return !!d.capabilities?.includes(cap)
}

// slider fill percentage for inline --pct style
export function pctStyle(value: number, min: number, max: number): Record<string, string> {
  const clamped = Math.min(100, Math.max(0, ((value - min) / (max - min)) * 100))
  return { '--pct': `${clamped}%` }
}
