package schemes

import (
	"strconv"
	"strings"
)

// isBetUnitArtifact 识别误写入 betMode 的投注单位（如 "1"、"0.01"），与玩法 betMode（danshi/dingwei 等）区分。
func isBetUnitArtifact(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
			return false
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	return err == nil && f > 0
}

func schemeBetUnitFromConfig(cfg map[string]interface{}) float64 {
	if cfg == nil {
		return baseBetUnitYuan
	}
	if v := strings.TrimSpace(stringVal(cfg, "betUnit")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	if v := strings.TrimSpace(stringVal(cfg, "betMode")); isBetUnitArtifact(v) {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return baseBetUnitYuan
}

func playBetModeFromConfig(cfg map[string]interface{}) string {
	betMode := strings.TrimSpace(stringVal(cfg, "betMode"))
	if isBetUnitArtifact(betMode) {
		return ""
	}
	return betMode
}

// normalizeSchemeConfigBetFields 将误存的投注单位从 betMode 迁到 betUnit，避免玩法解析污染。
func normalizeSchemeConfigBetFields(cfg map[string]interface{}) {
	if cfg == nil {
		return
	}
	betUnit := strings.TrimSpace(stringVal(cfg, "betUnit"))
	betMode := strings.TrimSpace(stringVal(cfg, "betMode"))

	if betUnit == "" && isBetUnitArtifact(betMode) {
		cfg["betUnit"] = betMode
		delete(cfg, "betMode")
		return
	}
	if betUnit != "" && isBetUnitArtifact(betMode) {
		delete(cfg, "betMode")
	}
}
