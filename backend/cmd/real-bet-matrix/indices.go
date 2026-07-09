package main

import (
	"fmt"
	"strconv"
	"strings"
)

func parseIndexRanges(spec string) (map[int]bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	out := map[int]bool{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid range %q", part)
			}
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q: %w", bounds[0], err)
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q: %w", bounds[1], err)
			}
			if end < start {
				return nil, fmt.Errorf("invalid range %q: end < start", part)
			}
			for i := start; i <= end; i++ {
				out[i] = true
			}
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid index %q: %w", part, err)
		}
		out[n] = true
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty indices")
	}
	return out, nil
}

func indexAllowed(indexSet map[int]bool, index int) bool {
	if len(indexSet) == 0 {
		return true
	}
	return indexSet[index]
}
