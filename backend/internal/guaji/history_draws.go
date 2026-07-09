package guaji

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HistoryDrawLog 第三方彩种开奖历史单条（文档 §5；GET /api/lottery_logs 等）。
type HistoryDrawLog struct {
	ID        int64
	Periods   string
	DrawnAt   time.Time
	Balls     DrawBalls
	PeriodRaw string
}

type historyDrawLogsEnvelope struct {
	Code    flexCode          `json:"code"`
	Data    []historyDrawItem `json:"data"`
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
	Count   int               `json:"count"`
}

type historyDrawItem struct {
	ID           int64  `json:"id"`
	Created      string `json:"created"`
	BlockTime    string `json:"block_time"`
	Periods      string `json:"periods"`
	Last5Num     string `json:"last5_num"`
	Last106Num   string `json:"last10_6_num"`
	Last115Num   string `json:"last11_5_num"`
	LastPK10Num  string `json:"last_pk10_num"`
	LastK3Num    string `json:"last_k3_num"`
	LHCNum       string `json:"lhc_num"`
	LastTW5Num   string `json:"last_tw5_num"`
	LastTWPK10   string `json:"last_tw_pk10_num"`
	LastTW28Num  string `json:"last_tw28_num"`
}

// FetchHistoryDrawLogs GET /api/{apiPath}?limit=&page=（匿名，无需 token）。
// apiPath 如 lottery_logs、lottery_log103s、eth_lottery_logs（不含 /api/ 前缀）。
func (c *Client) FetchHistoryDrawLogs(ctx context.Context, apiPath string, page, limit int) ([]HistoryDrawLog, error) {
	if !c.cfg.Enabled {
		return nil, ErrMisconfigured("GUAJI_ENABLED=false")
	}
	apiPath = strings.TrimPrefix(strings.TrimSpace(apiPath), "/api/")
	if apiPath == "" {
		return nil, fmt.Errorf("guaji history draws: empty api path")
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("page", strconv.Itoa(page))
	path := "/api/" + apiPath + "?" + q.Encode()

	env, raw, err := c.doJSONRaw(ctx, "GET", c.cfg.HTTPBase, path, "", nil)
	if err != nil {
		return nil, err
	}
	if err := c.parseEnvelope(env); err != nil {
		return nil, err
	}

	var body historyDrawLogsEnvelope
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &body.Data); err != nil {
			return nil, fmt.Errorf("guaji history draws decode data: %w", err)
		}
	} else if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("guaji history draws decode: %w", err)
		}
	}
	if body.Code != 0 && body.Code != 200 && body.Code != 201 {
		return nil, fmt.Errorf("guaji history draws: unexpected code %v", body.Code)
	}

	out := make([]HistoryDrawLog, 0, len(body.Data))
	for _, row := range body.Data {
		if log, ok := mapHistoryDrawItem(row); ok {
			out = append(out, log)
		}
	}
	return out, nil
}

func mapHistoryDrawItem(row historyDrawItem) (HistoryDrawLog, bool) {
	periods := strings.TrimSpace(row.Periods)
	if periods == "" {
		return HistoryDrawLog{}, false
	}
	drawnAt := parseDrawTime(firstNonEmpty(row.BlockTime, row.Created))
	return HistoryDrawLog{
		ID:        row.ID,
		Periods:   periods,
		PeriodRaw: periods,
		DrawnAt:   drawnAt,
		Balls: DrawBalls{
			SSC:     row.Last5Num,
			SYXW:    row.Last115Num,
			PK10:    row.LastPK10Num,
			K3:      row.LastK3Num,
			LHC:     row.LHCNum,
			ETH10_6: row.Last106Num,
			TW5:     row.LastTW5Num,
			TWPK10:  row.LastTWPK10,
			TW28:    row.LastTW28Num,
		},
	}, true
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
