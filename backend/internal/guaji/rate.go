package guaji

import (
	"context"
	"encoding/json"
	"fmt"
)

// RealRate 是 GET /api/agents/i/real/rate 的账号级赔率/返水信息。
// 实测（2026-07）：{"real_rate":0.015,"lott_odds":1940,"hash_odds":1950,...}
//   - lott_odds：普通彩（波场/以太/币安）2 元「三星直选」基准派彩（÷2=1 元单注可中 970）
//   - hash_odds：哈希彩对应基准
type RealRate struct {
	RealRate     float64 `json:"real_rate"`
	LottOdds     float64 `json:"lott_odds"`
	HashOdds     float64 `json:"hash_odds"`
	LottRealRate float64 `json:"lott_real_rate"`
	UserType     string  `json:"user_type"`
}

// FetchRealRate 拉取 GET /api/agents/i/real/rate（需登录 token；返回 {"code":0,"data":{...}}）。
func (c *Client) FetchRealRate(ctx context.Context, accessToken string) (*RealRate, error) {
	if !c.cfg.Enabled {
		return nil, ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if accessToken == "" {
		return nil, fmt.Errorf("guaji real rate: empty access token")
	}
	env, raw, err := c.doJSONRaw(ctx, "GET", c.cfg.HTTPBase, "/api/agents/i/real/rate", accessToken, nil)
	if err != nil {
		return nil, err
	}
	if err := c.parseEnvelope(env); err != nil {
		return nil, err
	}
	var rate RealRate
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &rate); err != nil {
			return nil, fmt.Errorf("guaji real rate decode: %w", err)
		}
	} else if err := json.Unmarshal(raw, &rate); err != nil {
		return nil, fmt.Errorf("guaji real rate decode: %w", err)
	}
	return &rate, nil
}
