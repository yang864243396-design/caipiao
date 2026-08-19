package config

import (
	"strconv"
	"strings"
)

func splitInt32CSV(raw string) []int32 {
	parts := strings.Split(raw, ",")
	out := make([]int32, 0, len(parts))
	seen := make(map[int32]struct{}, len(parts))
	for _, part := range parts {
		n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 32)
		if err != nil || n < 0 {
			continue
		}
		value := int32(n)
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
