package lookback

import (
	"fmt"
	"sort"
	"strings"
)

type RunMode string

const (
	RunModeReal RunMode = "real"
	RunModeSim  RunMode = "sim"
)

// EncodeRunModes 将运行模式列表写入 DB/API 存储串（real,sim 或空）。
func EncodeRunModes(modes []RunMode) string {
	if len(modes) == 0 {
		return ""
	}
	seen := make(map[RunMode]struct{}, len(modes))
	var uniq []RunMode
	for _, m := range modes {
		if m != RunModeReal && m != RunModeSim {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		uniq = append(uniq, m)
	}
	if len(uniq) == 0 {
		return ""
	}
	sort.Slice(uniq, func(i, j int) bool { return uniq[i] < uniq[j] })
	parts := make([]string, len(uniq))
	for i, m := range uniq {
		parts[i] = string(m)
	}
	return strings.Join(parts, ",")
}

// DecodeRunModes 解析存储串；非法片段忽略。
func DecodeRunModes(raw string) []RunMode {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []RunMode
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		switch RunMode(p) {
		case RunModeReal, RunModeSim:
			out = append(out, RunMode(p))
		}
	}
	return NormalizeRunModes(out)
}

// NormalizeRunModes 去重并校验，允许空集。
func NormalizeRunModes(modes []RunMode) []RunMode {
	if len(modes) == 0 {
		return nil
	}
	seen := make(map[RunMode]struct{}, len(modes))
	var uniq []RunMode
	for _, m := range modes {
		if m != RunModeReal && m != RunModeSim {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		uniq = append(uniq, m)
	}
	if len(uniq) == 0 {
		return nil
	}
	sort.Slice(uniq, func(i, j int) bool { return uniq[i] < uniq[j] })
	return uniq
}

// ContainsRunMode 判断实例运行模式是否在已选集合内；空集不匹配任何实例。
func ContainsRunMode(modes []RunMode, instRunMode string) bool {
	if len(modes) == 0 {
		return false
	}
	for _, m := range modes {
		if string(m) == instRunMode {
			return true
		}
	}
	return false
}

func validateRunModes(modes []RunMode) error {
	for _, m := range modes {
		if m != RunModeReal && m != RunModeSim {
			return fmt.Errorf("%w: runModes 仅允许 real、sim", ErrInvalidSettings)
		}
	}
	return nil
}
