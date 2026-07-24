package hass

import (
	"strings"
)

// Domain denylist for discovery noise
var denylistDomains = map[string]bool{
	"zone": true, "person": true, "device_tracker": true,
	"persistent_notification": true, "update": true,
	"sun": true, "event": true, "conversation": true,
	"tts": true, "stt": true, "assist_satellite": true,
}

func DomainOf(entityID string) string {
	if i := strings.IndexByte(entityID, '.'); i > 0 {
		return entityID[:i]
	}
	return ""
}

func IsDenylisted(domain string) bool {
	return denylistDomains[domain]
}

func FriendlyName(st State) string {
	if st.Attributes != nil {
		if n, ok := st.Attributes["friendly_name"].(string); ok && n != "" {
			return n
		}
	}
	return st.EntityID
}

func Available(st State) bool {
	return st.State != "unavailable" && st.State != "unknown"
}

// InferCapabilities returns capabilities + control_level for a domain/state.
func InferCapabilities(domain string, attrs map[string]any) (caps []string, level string) {
	if attrs == nil {
		attrs = map[string]any{}
	}
	switch domain {
	case "light":
		caps = []string{"on_off"}
		if hasColorMode(attrs, "brightness") || attrs["brightness"] != nil {
			caps = append(caps, "brightness")
		}
		if hasColorMode(attrs, "color_temp") || attrs["color_temp"] != nil || attrs["color_temp_kelvin"] != nil {
			caps = append(caps, "color_temp")
		}
		if hasColorMode(attrs, "hs") || attrs["hs_color"] != nil {
			caps = append(caps, "color_hs")
		}
		if el, ok := attrs["effect_list"]; ok && el != nil {
			caps = append(caps, "effect")
		}
		return caps, "full"
	case "switch", "input_boolean", "fan", "remote", "siren":
		return []string{"on_off"}, "full"
	case "cover":
		return []string{"open_close", "position"}, "full"
	case "climate":
		return []string{"on_off", "hvac_mode", "temperature"}, "full"
	case "media_player":
		return []string{"on_off", "play_pause", "volume"}, "full"
	case "lock":
		return []string{"lock"}, "full"
	case "vacuum":
		return []string{"start_stop", "return_home"}, "full"
	case "scene", "script", "button", "input_button":
		return []string{"activate"}, "action"
	case "sensor", "binary_sensor", "weather", "sun", "person", "device_tracker":
		return []string{"read"}, "read_only"
	default:
		return []string{"read"}, "read_only"
	}
}

func hasColorMode(attrs map[string]any, mode string) bool {
	raw, ok := attrs["supported_color_modes"]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok && s == mode {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s == mode {
				return true
			}
		}
	}
	return false
}

func PrimaryDisplay(domain, state string, attrs map[string]any) string {
	if !Available(State{State: state}) {
		return "不可用"
	}
	switch domain {
	case "light", "switch", "input_boolean", "fan":
		if state == "on" {
			if domain == "light" && attrs != nil {
				if b, ok := asFloat(attrs["brightness"]); ok {
					pct := int(b / 255 * 100)
					return "开启 · " + itoa(pct) + "%"
				}
			}
			return "开启"
		}
		if state == "off" {
			return "关闭"
		}
	case "sensor", "binary_sensor":
		unit, _ := attrs["unit_of_measurement"].(string)
		if unit != "" {
			return state + " " + unit
		}
		return state
	case "climate":
		if t, ok := asFloat(attrs["temperature"]); ok {
			return state + " · " + ftoa(t) + "°"
		}
	}
	return state
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func ftoa(f float64) string {
	// simple one-decimal if needed
	n := int(f)
	frac := int((f - float64(n)) * 10)
	if frac < 0 {
		frac = -frac
	}
	if frac == 0 {
		return itoa(n)
	}
	return itoa(n) + "." + itoa(frac)
}
