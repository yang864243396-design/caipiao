package schemerounds

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Round 倍投轮次（0-based 跳转目标）。
type Round struct {
	Mult      float64
	AfterHit  int
	AfterMiss int
}

// ParseFromDefinitionConfig 从 scheme_definitions.config JSON 解析 rounds。
func ParseFromDefinitionConfig(config []byte) []Round {
	if len(config) == 0 {
		return nil
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(config, &raw); err != nil {
		return nil
	}
	return normalize(parseFromRaw(raw["rounds"]))
}

func parseFromRaw(raw interface{}) []Round {
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]Round, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, Round{
			Mult:      toFloat(m["mult"], 1),
			AfterHit:  toInt(m["afterHit"], 0),
			AfterMiss: toInt(m["afterMiss"], 0),
		})
	}
	return out
}

// useOneBasedTargets 高级倍投轮次页以「第 N 局」存储跳转目标（≥1）。
func useOneBasedTargets(rounds []Round) bool {
	if len(rounds) == 0 {
		return false
	}
	for _, r := range rounds {
		if r.AfterHit == 0 || r.AfterMiss == 0 {
			return false
		}
	}
	return true
}

func normalize(rounds []Round) []Round {
	if len(rounds) == 0 || !useOneBasedTargets(rounds) {
		return rounds
	}
	out := make([]Round, len(rounds))
	for i, r := range rounds {
		out[i] = Round{
			Mult:      r.Mult,
			AfterHit:  r.AfterHit - 1,
			AfterMiss: r.AfterMiss - 1,
		}
	}
	return out
}

// NextIndex 按实际中/未中计算下一期轮次索引。
func NextIndex(rounds []Round, cur int, hit bool) int {
	if len(rounds) == 0 {
		return 0
	}
	if cur < 0 || cur >= len(rounds) {
		cur = 0
	}
	r := rounds[cur]
	if hit {
		return clampIndex(r.AfterHit, len(rounds))
	}
	return clampIndex(r.AfterMiss, len(rounds))
}

func clampIndex(v, n int) int {
	if n <= 0 {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v >= n {
		return v % n
	}
	return v
}

func toFloat(v interface{}, fallback float64) float64 {
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return n
		}
	case int:
		if n > 0 {
			return float64(n)
		}
	case json.Number:
		f, err := n.Float64()
		if err == nil && f > 0 {
			return f
		}
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			break
		}
		f, err := strconv.ParseFloat(s, 64)
		if err == nil && f > 0 {
			return f
		}
	}
	return fallback
}

func toInt(v interface{}, fallback int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return int(i)
		}
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			break
		}
		i, err := strconv.Atoi(s)
		if err == nil {
			return i
		}
	}
	return fallback
}
