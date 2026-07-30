package main

import (
	"sort"
	"time"
)

// 严重度。P0/P1 在 -ci 模式下 exit 1。
const (
	P0 = "P0" // 资金/判定错误：钱或输赢已经算错
	P1 = "P1" // 下注非法：内容越界、注数不可下，会被第三方拒单或静默错投
	P2 = "P2" // 逻辑缺陷：结果还对，但选号/状态推导已偏
	P3 = "P3" // 展示瑕疵
)

type Finding struct {
	Severity string `json:"severity"`
	Check    string `json:"check"`
	Scope    string `json:"scope"` // bet | scheme | lottery
	Key      string `json:"key"`
	Lottery  string `json:"lottery,omitempty"`
	Play     string `json:"play,omitempty"`
	SimBet   bool   `json:"simBet,omitempty"`
	Detail   string `json:"detail"`
	PlacedAt string `json:"placedAt,omitempty"`
}

type collector struct {
	items []Finding
	// scanned 记录各作用域实际检查过的对象数，用于报告里给出分母
	scanned map[string]int
	// skipped 记录因缺输入而跳过的检查，避免把"没查"读成"没问题"
	skipped map[string]int
}

func newCollector() *collector {
	return &collector{scanned: map[string]int{}, skipped: map[string]int{}}
}

func (c *collector) add(f Finding) { c.items = append(c.items, f) }

func (c *collector) skip(check string) { c.skipped[check]++ }

func (c *collector) countBySeverity() map[string]int {
	out := map[string]int{}
	for _, f := range c.items {
		out[f.Severity]++
	}
	return out
}

type checkSummary struct {
	Check    string
	Severity string
	Count    int
	Recent   int // 近 24 小时内的命中数：区分"历史遗留"与"现在还在发生"
	Sample   string
}

// summarize 汇总；recentSince 之后的命中单独计数。
func (c *collector) summarize(recentSince time.Time) []checkSummary {
	agg := map[string]*checkSummary{}
	for _, f := range c.items {
		s, ok := agg[f.Check]
		if !ok {
			s = &checkSummary{Check: f.Check, Severity: f.Severity, Sample: f.Key + "：" + f.Detail}
			agg[f.Check] = s
		}
		s.Count++
		if f.PlacedAt == "" {
			s.Recent++ // 方案 / 彩种级检查无时间维度，视为当前状态
			continue
		}
		if t, err := time.Parse(time.RFC3339, f.PlacedAt); err == nil && t.After(recentSince) {
			s.Recent++
		}
	}
	out := make([]checkSummary, 0, len(agg))
	for _, s := range agg {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity < out[j].Severity
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func (c *collector) hasBlocking() bool {
	for _, f := range c.items {
		if f.Severity == P0 || f.Severity == P1 {
			return true
		}
	}
	return false
}
