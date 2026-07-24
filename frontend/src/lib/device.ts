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
}

export function domainEmoji(domain: string) {
  return domainIcon[domain] || '▦'
}

export function canToggle(d: DeviceView) {
  return (
    d.control_level !== 'read_only' &&
    (d.capabilities?.includes('on_off') ||
      ['light', 'switch', 'input_boolean', 'fan'].includes(d.domain))
  )
}

export function isOn(d: DeviceView) {
  return d.state === 'on' || d.state === 'open' || d.state === 'playing' || d.state === 'home'
}

export function toggleAction(d: DeviceView): 'turn_on' | 'turn_off' {
  return isOn(d) ? 'turn_off' : 'turn_on'
}
