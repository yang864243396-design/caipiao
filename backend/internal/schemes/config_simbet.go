package schemes

import (
	"encoding/json"
	"strings"
)

// configSimBet 从 definition config 读取 simBet；兼容旧 runMode prod/sim。
func configSimBet(cfg []byte) bool {
	if len(cfg) == 0 {
		return false
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(cfg, &m); err != nil {
		return false
	}
	if v, ok := m["simBet"]; ok {
		switch t := v.(type) {
		case bool:
			return t
		case string:
			return strings.EqualFold(strings.TrimSpace(t), "true")
		}
	}
	if v, ok := m["runMode"]; ok {
		return strings.EqualFold(strings.TrimSpace(stringFromConfigValue(v)), "sim")
	}
	return false
}

func setConfigSimBet(cfg map[string]interface{}, simBet bool) {
	if cfg == nil {
		return
	}
	cfg["simBet"] = simBet
	delete(cfg, "runMode")
}

func simBetFromClientRunMode(raw string) bool {
	return strings.EqualFold(strings.TrimSpace(raw), "sim")
}

func runModeFromSimBet(simBet bool) string {
	if simBet {
		return "sim"
	}
	return "real"
}

func stringFromConfigValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return strings.TrimSpace(strings.Trim(string(mustJSON(v)), `"`))
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
