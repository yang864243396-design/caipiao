package schemes

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultPoolMin = 0
const defaultPoolMax = 9

func ruleNumberPool(rule playRule) (min, max int) {
	if rule.NumberPoolMax > 0 {
		min = rule.NumberPoolMin
		max = rule.NumberPoolMax
		return min, max
	}
	return defaultPoolMin, defaultPoolMax
}

func normalizePoolToken(raw string, minVal, maxVal int) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < minVal || n > maxVal {
		return "", false
	}
	// 11选5 等号池 ≥11 才补零；PK10(1–10) 与开奖球保持 unpadded
	if maxVal >= 11 {
		return fmt.Sprintf("%02d", n), true
	}
	return strconv.Itoa(n), true
}

func parsePoolTokens(raw string, minVal, maxVal int) []string {
	raw = strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		tok, ok := normalizePoolToken(p, minVal, maxVal)
		if !ok {
			continue
		}
		if _, dup := seen[tok]; dup {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	return out
}

func parsePickTokensForRule(rule playRule, raw string) []string {
	min, max := ruleNumberPool(rule)
	if max <= defaultPoolMax && min == defaultPoolMin {
		return parseDigitTokens(raw)
	}
	tokens := parsePoolTokens(raw, min, max)
	if len(tokens) == 0 && max <= defaultPoolMax {
		return parseDigitTokens(raw)
	}
	return tokens
}

func parseSegmentTokensForRule(rule playRule, raw string, segmentLen int) []string {
	raw = strings.NewReplacer("\n", ",", "，", ",", " ", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	min, max := ruleNumberPool(rule)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if max > defaultPoolMax {
			segs := strings.FieldsFunc(p, func(r rune) bool { return r == '-' || r == '+' })
			if len(segs) == segmentLen {
				ok := true
				norm := make([]string, 0, segmentLen)
				for _, s := range segs {
					tok, valid := normalizePoolToken(s, min, max)
					if !valid {
						ok = false
						break
					}
					norm = append(norm, tok)
				}
				if ok {
					out = append(out, strings.Join(norm, ""))
					continue
				}
			}
		}
		if segmentLen > 0 && len(p) == segmentLen && max <= defaultPoolMax {
			out = append(out, p)
			continue
		}
		if max > defaultPoolMax && strings.Count(p, "-") == segmentLen-1 {
			segs := strings.Split(p, "-")
			ok := true
			norm := make([]string, 0, segmentLen)
			for _, s := range segs {
				tok, valid := normalizePoolToken(s, min, max)
				if !valid {
					ok = false
					break
				}
				norm = append(norm, tok)
			}
			if ok {
				out = append(out, strings.Join(norm, ""))
			}
		}
	}
	return out
}

func ballsMatchToken(drawn []string, token string) bool {
	if len(drawn) == 0 || token == "" {
		return false
	}
	min, max := defaultPoolMin, defaultPoolMax
	if len(drawn) > 0 {
		if n := atoiBall(drawn[0]); n > 9 {
			min, max = 1, 11
		}
	}
	if max > defaultPoolMax && len(token)%2 == 0 {
		want := len(token) / 2
		if want != len(drawn) {
			return false
		}
		for i := 0; i < want; i++ {
			part := token[i*2 : i*2+2]
			tok, ok := normalizePoolToken(part, min, max)
			if !ok || !containsDigit(drawn, tok) {
				return false
			}
		}
		return true
	}
	return token == strings.Join(drawn, "")
}
