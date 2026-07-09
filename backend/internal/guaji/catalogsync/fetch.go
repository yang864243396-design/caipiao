package catalogsync

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

// RemoteLottery 第三方 new_lott 彩种条目。
type RemoteLottery struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
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

// FetchNewLott 拉取第三方彩种列表（公开接口，无需登录）。
func FetchNewLott(ctx context.Context, httpBase string, limit, page int) ([]RemoteLottery, error) {
	base := strings.TrimRight(strings.TrimSpace(httpBase), "/")
	if base == "" {
		return nil, fmt.Errorf("httpBase 不能为空")
	}
	if limit <= 0 {
		limit = 299
	}
	if page <= 0 {
		page = 1
	}
	url := fmt.Sprintf("%s/api/games/new_lott?limit=%d&page=%d", base, limit, page)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch new_lott: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("fetch new_lott: status %d body=%s", resp.StatusCode, truncate(string(raw), 256))
	}

	var env fetchEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode new_lott: %w", err)
	}
	code := env.Code.int()
	if code != 0 && code != 200 && code != 201 {
		return nil, fmt.Errorf("new_lott business code=%d", code)
	}
	var items []RemoteLottery
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return items, nil
	}
	if err := json.Unmarshal(env.Data, &items); err != nil {
		return nil, fmt.Errorf("decode new_lott data: %w", err)
	}
	return items, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
