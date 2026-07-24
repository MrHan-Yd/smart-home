package hass

import (
	"fmt"
	"strings"
)

// ResolveAction maps logical action + domain + params → HA domain/service/data.
// P0: turn_on / turn_off / toggle for switchable domains; light extras; scene/script/button.
func ResolveAction(domain, action string, params map[string]any, entityID string) (svcDomain, service string, data map[string]any, err error) {
	action = strings.TrimSpace(strings.ToLower(action))
	domain = strings.TrimSpace(strings.ToLower(domain))
	if params == nil {
		params = map[string]any{}
	}
	data = map[string]any{"entity_id": entityID}
	for k, v := range params {
		if k == "entity_id" {
			continue
		}
		data[k] = v
	}

	switch action {
	case "turn_on", "turn_off", "toggle":
		switch domain {
		case "light", "switch", "input_boolean", "fan", "remote", "siren", "media_player", "climate":
			return domain, action, data, nil
		default:
			return "", "", nil, fmt.Errorf("domain %s does not support %s", domain, action)
		}
	case "set_brightness":
		if domain != "light" {
			return "", "", nil, fmt.Errorf("set_brightness only for light")
		}
		return "light", "turn_on", data, nil
	case "set_color_temp":
		if domain != "light" {
			return "", "", nil, fmt.Errorf("set_color_temp only for light")
		}
		return "light", "turn_on", data, nil
	case "set_hs_color":
		if domain != "light" {
			return "", "", nil, fmt.Errorf("set_hs_color only for light")
		}
		return "light", "turn_on", data, nil
	case "set_effect":
		if domain != "light" {
			return "", "", nil, fmt.Errorf("set_effect only for light")
		}
		return "light", "turn_on", data, nil
	case "set_volume":
		if domain != "media_player" {
			return "", "", nil, fmt.Errorf("set_volume only for media_player")
		}
		return "media_player", "volume_set", data, nil
	case "play_pause":
		if domain != "media_player" {
			return "", "", nil, fmt.Errorf("play_pause only for media_player")
		}
		return "media_player", "media_play_pause", data, nil
	case "play":
		if domain != "media_player" {
			return "", "", nil, fmt.Errorf("play only for media_player")
		}
		return "media_player", "media_play", data, nil
	case "pause":
		if domain == "media_player" {
			return "media_player", "media_pause", data, nil
		}
		if domain != "vacuum" {
			return "", "", nil, fmt.Errorf("pause only for media_player or vacuum")
		}
		return "vacuum", "pause", data, nil
	case "open":
		if domain != "cover" {
			return "", "", nil, fmt.Errorf("open only for cover")
		}
		return "cover", "open_cover", data, nil
	case "close":
		if domain != "cover" {
			return "", "", nil, fmt.Errorf("close only for cover")
		}
		return "cover", "close_cover", data, nil
	case "stop":
		switch domain {
		case "cover":
			return "cover", "stop_cover", data, nil
		case "vacuum":
			return "vacuum", "stop", data, nil
		default:
			return "", "", nil, fmt.Errorf("stop only for cover or vacuum")
		}
	case "set_position":
		if domain != "cover" {
			return "", "", nil, fmt.Errorf("set_position only for cover")
		}
		return "cover", "set_cover_position", data, nil
	case "set_hvac_mode":
		if domain != "climate" {
			return "", "", nil, fmt.Errorf("set_hvac_mode only for climate")
		}
		return "climate", "set_hvac_mode", data, nil
	case "set_temperature":
		if domain != "climate" {
			return "", "", nil, fmt.Errorf("set_temperature only for climate")
		}
		return "climate", "set_temperature", data, nil
	case "lock":
		if domain != "lock" {
			return "", "", nil, fmt.Errorf("lock only for lock")
		}
		return "lock", "lock", data, nil
	case "unlock":
		if domain != "lock" {
			return "", "", nil, fmt.Errorf("unlock only for lock")
		}
		return "lock", "unlock", data, nil
	case "activate":
		if domain != "scene" {
			return "", "", nil, fmt.Errorf("activate only for scene")
		}
		return "scene", "turn_on", data, nil
	case "run":
		if domain != "script" {
			return "", "", nil, fmt.Errorf("run only for script")
		}
		return "script", "turn_on", data, nil
	case "press":
		if domain != "button" && domain != "input_button" {
			return "", "", nil, fmt.Errorf("press only for button")
		}
		return domain, "press", data, nil
	case "start":
		if domain != "vacuum" {
			return "", "", nil, fmt.Errorf("start only for vacuum")
		}
		return "vacuum", "start", data, nil
	case "return_to_base":
		if domain != "vacuum" {
			return "", "", nil, fmt.Errorf("return_to_base only for vacuum")
		}
		return "vacuum", "return_to_base", data, nil
	default:
		return "", "", nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// ActionAllowed checks control_level and basic capability for action.
func ActionAllowed(domain, action, controlLevel string) bool {
	if controlLevel == "read_only" {
		return false
	}
	action = strings.ToLower(action)
	switch action {
	case "turn_on", "turn_off", "toggle":
		switch domain {
		case "light", "switch", "input_boolean", "fan", "remote", "siren", "media_player", "climate":
			return true
		}
		return false
	case "set_volume", "play_pause", "play", "pause":
		return domain == "media_player" || (action == "pause" && domain == "vacuum")
	default:
		return controlLevel == "full" || controlLevel == "action"
	}
}
