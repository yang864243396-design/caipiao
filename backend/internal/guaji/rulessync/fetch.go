package rulessync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RulesTemplate struct {
	Name   string       `json:"name"`
	Groups []RulesGroup `json:"groups"`
}

type RulesGroup struct {
	Name string      `json:"name"`
	Team []RulesTeam `json:"team"`
}

type RulesTeam struct {
	Name string      `json:"name"`
	Rule []RulesRule `json:"rule"`
}

type RulesRule struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Active   bool   `json:"active"`
}

type fetchEnvelope struct {
	Code flexCode        `json:"code"`
	Data json.RawMessage `json:"data"`
}

type flexCode int

func (c *flexCode) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*c = 0
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		n, err := strconv.Atoi(strings.TrimSpace(str))
		if err != nil {
			return err
		}
		*c = flexCode(n)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*c = flexCode(n)
	return nil
}

func (c flexCode) int() int { return int(c) }

func FetchRulesV2(ctx context.Context, httpBase string) (map[string]RulesTemplate, error) {
	base := strings.TrimRight(strings.TrimSpace(httpBase), "/")
	if base == "" {
		return nil, fmt.Errorf("httpBase 不能为空")
	}
	url := base + "/api/games/rules/v2"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch rules/v2: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("fetch rules/v2: status %d", resp.StatusCode)
	}

	var env fetchEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode rules/v2: %w", err)
	}
	code := env.Code.int()
	if code != 0 && code != 200 && code != 201 {
		return nil, fmt.Errorf("rules/v2 business code=%d", code)
	}

	var data map[string]RulesTemplate
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("decode rules/v2 data: %w", err)
	}
	return data, nil
}
