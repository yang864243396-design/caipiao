package guaji

import (
	"context"
	"encoding/json"
	"fmt"
)

// UserInfo fetches GET /api/users/i/info.
// 实测该接口返回裸对象（无 data 包裹）：优先取 envelope.data，缺失则解析整个响应体。
func (c *Client) UserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	env, raw, err := c.doJSONRaw(ctx, "GET", c.cfg.HTTPBase, "/api/users/i/info", accessToken, nil)
	if err != nil {
		return nil, err
	}
	if err := c.parseEnvelope(env); err != nil {
		return nil, err
	}
	var info UserInfo
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &info); err != nil {
			return nil, fmt.Errorf("guaji user info decode: %w", err)
		}
	} else if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("guaji user info decode: %w", err)
	}
	return &info, nil
}

// BalanceCNY returns CNY available balance from users/i/info.
func (c *Client) BalanceCNY(ctx context.Context, accessToken string) (float64, error) {
	info, err := c.UserInfo(ctx, accessToken)
	if err != nil {
		return 0, err
	}
	return info.CNYBalance(), nil
}
