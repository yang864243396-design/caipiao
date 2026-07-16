package guaji

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LottBetContent 是单个注单内容（接口文档 §11 bet_contents 元素）。
type LottBetContent struct {
	RuleID             string   `json:"rule_id"`                        // 规则ID（如 "13"）
	BetContent         string   `json:"bet_content"`                    // 投注内容（位段逗号分隔，如 ",,,13579,"）
	AmountUnit         float64  `json:"amount_unit"`                    // 单元金额
	BetsNums           int      `json:"bets_nums"`                      // 注数
	Multiple           int      `json:"multiple"`                       // 倍数
	BetAmount          float64  `json:"bet_amount"`                     // 金额
	Solo               bool     `json:"solo"`                           // 是否单挑
	MinSingleBetBonus  *float64 `json:"min_single_bet_bonus,omitempty"` // 每注中奖（单挑校验用，前端常传）
	SingleBetAmount    *float64 `json:"singleBetAmount,omitempty"`      // 单注金额（前端字段，部分规则校验用）
}

// LottBetMultipleOuter 外层倍投（§11 bet_multiple；不加倍时传 []）。
type LottBetMultipleOuter struct {
	BetAmount float64 `json:"bet_amount"`
	Multiple  int     `json:"multiple"`
}

// LottBetRequest 是 POST /api/web_bets/lott 的真实下单请求体（接口文档 §11）。
type LottBetRequest struct {
	AutoType    string                 `json:"auto_type"`    // 投注来源（≤10 字符）
	BetContents []LottBetContent       `json:"bet_contents"` // 注单内容数组
	GameID      int                    `json:"game_id"`      // 游戏ID（数字）
	Currency    int                    `json:"currency"`     // 0 usdt / 1 trx / 3 cny
	BetMultiple []LottBetMultipleOuter `json:"bet_multiple"` // 外层倍投，可 []
}

// LottBetResult 是 web_bets/lott 接单成功返回（实测 code=201 时 data 常为空，periods 在顶层）。
type LottBetResult struct {
	ThirdPartyBetID string  `json:"id"`
	Periods         string  `json:"periods"`
	Status          string  `json:"status"`
	Amount          float64 `json:"amount"`
	Currency        int     `json:"currency"`
}

// LottPeriod 是 GET /api/web_bets/lott/periods 返回的未来期信息。
type LottPeriod struct {
	Period    string `json:"period"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// WebBetRecord 是 GET /api/web_bets/ 列表项（T5 派奖同步字段映射）。
type WebBetRecord struct {
	ID            int64   `json:"id"`
	GameID        int     `json:"game_id"`
	Periods       string  `json:"periods"`
	BetAmount     float64 `json:"bet_amount"`
	NetAmount     float64 `json:"net_amount"`
	PayoutAmount  float64 `json:"payout_amount"`
	Status        int     `json:"status"`
	Settled       bool    `json:"settled"`
	Confirmed     bool    `json:"confirmed"`
	Currency      int     `json:"currency"`
}

// PlaceLottBet 调用 /api/web_bets/lott 真实下单（接口文档 §11）。
func (c *Client) PlaceLottBet(ctx context.Context, accessToken string, req LottBetRequest) (*LottBetResult, error) {
	if !c.cfg.Enabled {
		return nil, ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if accessToken == "" {
		return nil, fmt.Errorf("guaji place bet: empty access token")
	}
	if req.GameID == 0 || len(req.BetContents) == 0 {
		return nil, fmt.Errorf("guaji place bet: game_id/bet_contents 不能为空")
	}
	if req.BetMultiple == nil {
		req.BetMultiple = []LottBetMultipleOuter{}
	}
	if req.AutoType == "" {
		req.AutoType = "platform"
	}
	env, raw, err := c.doJSONRaw(ctx, "POST", c.cfg.HTTPBase, "/api/web_bets/lott", accessToken, req)
	if err != nil {
		return nil, err
	}
	if err := c.parseEnvelope(env); err != nil {
		return nil, err
	}
	res := parseLottBetResult(env, raw)
	if res.ThirdPartyBetID == "" && res.Periods != "" {
		amount := req.BetContents[0].BetAmount
		if id, ferr := c.findBetIDByPeriod(ctx, accessToken, req.GameID, res.Periods, amount); ferr == nil {
			res.ThirdPartyBetID = id
		}
	}
	if strings.TrimSpace(res.ThirdPartyBetID) == "" {
		return nil, fmt.Errorf("guaji place bet: upstream did not return bet id (periods=%q rule_id=%s raw=%s)", res.Periods, req.BetContents[0].RuleID, truncate(string(raw), 256))
	}
	return &res, nil
}

// FetchLottPeriods 拉取未来开盘信息 POST /api/web_bets/lott/periods（需登录 token；GET 会 405）。
// 返回解码后的期列表与上游原始 JSON 响应体。
func (c *Client) FetchLottPeriods(ctx context.Context, accessToken string, gameID, numPeriods int) ([]LottPeriod, []byte, error) {
	if !c.cfg.Enabled {
		return nil, nil, ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if accessToken == "" {
		return nil, nil, fmt.Errorf("guaji lott periods: empty access token")
	}
	if gameID <= 0 {
		return nil, nil, fmt.Errorf("guaji lott periods: invalid game_id")
	}
	if numPeriods <= 0 {
		numPeriods = 2
	}
	if numPeriods > 10 {
		numPeriods = 10
	}
	body := map[string]int{"game_id": gameID, "num_periods": numPeriods}
	env, raw, err := c.doJSONRaw(ctx, "POST", c.cfg.HTTPBase, "/api/web_bets/lott/periods", accessToken, body)
	if err != nil {
		return nil, nil, err
	}
	if err := c.parseEnvelope(env); err != nil {
		return nil, raw, err
	}
	items, err := decodeLottPeriods(env.Data, raw)
	return items, raw, err
}

func decodeLottPeriods(data json.RawMessage, raw []byte) ([]LottPeriod, error) {
	if len(data) > 0 && string(data) != "null" {
		var items []LottPeriod
		if err := json.Unmarshal(data, &items); err == nil && len(items) > 0 {
			return items, nil
		}
	}
	var wrap struct {
		Data []LottPeriod `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Data) > 0 {
		return wrap.Data, nil
	}
	return nil, fmt.Errorf("guaji lott periods decode: empty data")
}

// EffectiveBetCloseAt 当前可投封盘时刻与展示用 end_time 字符串。
// 已开盘期用 end_time；列表中最近一期尚未开盘时（当前期未出现在 periods 列表）用 start_time，与第三方页面对齐。
func EffectiveBetCloseAt(lotteryCode string, p LottPeriod, now time.Time) (closeAt time.Time, closeEndTimeRaw string, ok bool) {
	lotteryCode = strings.TrimSpace(lotteryCode)
	endAt, err := ParseGuajiPeriodTimeForLottery(lotteryCode, p.EndTime)
	if err != nil || !endAt.After(now.UTC()) {
		return time.Time{}, "", false
	}
	startAt, startErr := ParseGuajiPeriodTimeForLottery(lotteryCode, p.StartTime)
	if startErr == nil && startAt.After(now.UTC()) {
		return startAt.UTC(), strings.TrimSpace(p.StartTime), true
	}
	return endAt.UTC(), strings.TrimSpace(p.EndTime), true
}

// PickOpenLottPeriod 选取当前可投注期：优先 start_time 已到达且 end_time 尚未到达；无则兜底 end>now 最小期号。
func PickOpenLottPeriod(periods []LottPeriod, lotteryCode string, now time.Time) (LottPeriod, time.Time, bool) {
	now = now.UTC()
	lotteryCode = strings.TrimSpace(lotteryCode)
	var p LottPeriod
	var ok bool
	if p, _, ok = pickOpenLottPeriodStarted(periods, lotteryCode, now); !ok {
		p, _, ok = pickOpenLottPeriodEndOnly(periods, lotteryCode, now)
	}
	if !ok {
		return LottPeriod{}, time.Time{}, false
	}
	closeAt, _, ok := EffectiveBetCloseAt(lotteryCode, p, now)
	return p, closeAt, ok
}

// ListOpenLottPeriodCandidates 返回当前可投注期候选，按期号升序（用于锚点已封盘时尝试下一期）。
func ListOpenLottPeriodCandidates(periods []LottPeriod, lotteryCode string, now time.Time) []LottPeriod {
	now = now.UTC()
	lotteryCode = strings.TrimSpace(lotteryCode)
	var out []LottPeriod
	seen := map[string]bool{}
	for _, p := range periods {
		period := strings.TrimSpace(p.Period)
		if period == "" || seen[period] {
			continue
		}
		if _, _, ok := EffectiveBetCloseAt(lotteryCode, p, now); !ok {
			continue
		}
		seen[period] = true
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return comparePeriodNumber(strings.TrimSpace(out[i].Period), strings.TrimSpace(out[j].Period)) < 0
	})
	return out
}

func pickOpenLottPeriodStarted(periods []LottPeriod, lotteryCode string, now time.Time) (LottPeriod, time.Time, bool) {
	var best LottPeriod
	var bestEnd time.Time
	found := false
	for _, p := range periods {
		period := strings.TrimSpace(p.Period)
		if period == "" {
			continue
		}
		endAt, err := ParseGuajiPeriodTimeForLottery(lotteryCode, p.EndTime)
		if err != nil || !endAt.After(now) {
			continue
		}
		if startAt, err := ParseGuajiPeriodTimeForLottery(lotteryCode, p.StartTime); err == nil && startAt.After(now) {
			continue
		}
		if !found || comparePeriodNumber(period, strings.TrimSpace(best.Period)) < 0 {
			best, bestEnd, found = p, endAt, true
		}
	}
	if !found {
		return LottPeriod{}, time.Time{}, false
	}
	return best, bestEnd, true
}

func pickOpenLottPeriodEndOnly(periods []LottPeriod, lotteryCode string, now time.Time) (LottPeriod, time.Time, bool) {
	var best LottPeriod
	var bestEnd time.Time
	found := false
	for _, p := range periods {
		period := strings.TrimSpace(p.Period)
		if period == "" {
			continue
		}
		endAt, err := ParseGuajiPeriodTimeForLottery(lotteryCode, p.EndTime)
		if err != nil || !endAt.After(now) {
			continue
		}
		if !found || comparePeriodNumber(period, strings.TrimSpace(best.Period)) < 0 {
			best, bestEnd, found = p, endAt, true
		}
	}
	if !found {
		return LottPeriod{}, time.Time{}, false
	}
	return best, bestEnd, true
}

// comparePeriodNumber 期号比较：纯数字按期号大小，否则字典序。
func comparePeriodNumber(a, b string) int {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == b {
		return 0
	}
	na, ea := parsePeriodNumber(a)
	nb, eb := parsePeriodNumber(b)
	if ea == nil && eb == nil {
		switch {
		case na < nb:
			return -1
		case na > nb:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(a, b)
}

func parsePeriodNumber(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("not numeric")
		}
	}
	return strconv.ParseInt(s, 10, 64)
}

// PickStartSkipLottPeriod 开启方案时跳过的最近一期：与当前可投期一致（PickOpenLottPeriod）。
func PickStartSkipLottPeriod(periods []LottPeriod, lotteryCode string, now time.Time) (LottPeriod, time.Time, bool) {
	if p, endAt, ok := PickOpenLottPeriod(periods, lotteryCode, now); ok {
		return p, endAt, true
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	for _, p := range periods {
		period := strings.TrimSpace(p.Period)
		if period == "" {
			continue
		}
		endAt, err := ParseGuajiPeriodTimeForLottery(lotteryCode, p.EndTime)
		if err != nil {
			continue
		}
		return p, endAt, true
	}
	return LottPeriod{}, time.Time{}, false
}

// LottPeriodDurationSec 从第三方 periods 推算单期时长（秒）；优先 start/end，否则相邻 end_time 之差。
func LottPeriodDurationSec(periods []LottPeriod, lotteryCode string, open LottPeriod, openClose time.Time) int {
	lotteryCode = strings.TrimSpace(lotteryCode)
	if startAt, err := ParseGuajiPeriodTimeForLottery(lotteryCode, open.StartTime); err == nil && !openClose.IsZero() {
		if d := int(openClose.Sub(startAt.UTC()).Round(time.Second).Seconds()); d > 0 {
			return d
		}
	}
	var ends []time.Time
	for _, p := range periods {
		endAt, err := ParseGuajiPeriodTimeForLottery(lotteryCode, p.EndTime)
		if err != nil {
			continue
		}
		ends = append(ends, endAt.UTC())
	}
	if len(ends) >= 2 {
		d := int(ends[1].Sub(ends[0]).Round(time.Second).Seconds())
		if d > 0 {
			return d
		}
	}
	return 0
}

// ParseGuajiPeriodTime 解析第三方 periods 时间（UTC 墙钟，兼容 hash 等；波场/以太请用 ParseGuajiPeriodTimeForLottery）。
func ParseGuajiPeriodTime(raw string) (time.Time, error) {
	return ParseGuajiPeriodTimeForLottery("", raw)
}

const wallClockLayout = "2006-01-02 15:04:05"

func parseLottBetResult(env envelope, raw []byte) LottBetResult {
	var res LottBetResult
	_ = env.dataInto(&res)
	var top struct {
		ID      json.RawMessage `json:"id"`
		Periods string          `json:"periods"`
	}
	if err := json.Unmarshal(raw, &top); err == nil {
		if res.Periods == "" {
			res.Periods = strings.TrimSpace(top.Periods)
		}
		if res.ThirdPartyBetID == "" {
			res.ThirdPartyBetID = rawJSONID(top.ID)
		}
	}
	return res
}

func rawJSONID(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return s
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}

func (c *Client) findBetIDByPeriod(ctx context.Context, accessToken string, gameID int, periods string, amount float64) (string, error) {
	periods = strings.TrimSpace(periods)
	if periods == "" {
		return "", fmt.Errorf("guaji find bet: empty periods")
	}
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(400 * time.Millisecond):
			}
		}
		items, err := c.ListWebBets(ctx, accessToken, 15, 1)
		if err != nil {
			lastErr = err
			continue
		}
		for _, it := range items {
			if strings.TrimSpace(it.Periods) != periods {
				continue
			}
			if gameID > 0 && it.GameID > 0 && it.GameID != gameID {
				continue
			}
			if amount > 0 && it.BetAmount > 0 && !floatNear(it.BetAmount, amount) {
				continue
			}
			return strconv.FormatInt(it.ID, 10), nil
		}
		lastErr = fmt.Errorf("guaji find bet: not found periods=%s", periods)
	}
	return "", lastErr
}

func floatNear(a, b float64) bool {
	const eps = 0.001
	if a > b {
		return a-b <= eps
	}
	return b-a <= eps
}

// FetchWebBetsRaw 拉取 GET /api/web_bets/ 原始 JSON（含 bet_contents）。
func (c *Client) FetchWebBetsRaw(ctx context.Context, accessToken string, limit, page int) (envelope, []byte, error) {
	if !c.cfg.Enabled {
		return envelope{}, nil, ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if accessToken == "" {
		return envelope{}, nil, fmt.Errorf("guaji list bets: empty access token")
	}
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	path := fmt.Sprintf("/api/web_bets/?limit=%d&page=%d", limit, page)
	return c.doJSONRaw(ctx, "GET", c.cfg.HTTPBase, path, accessToken, nil)
}

// ListWebBets 拉取 GET /api/web_bets/ 最近注单。
func (c *Client) ListWebBets(ctx context.Context, accessToken string, limit, page int) ([]WebBetRecord, error) {
	if !c.cfg.Enabled {
		return nil, ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if accessToken == "" {
		return nil, fmt.Errorf("guaji list bets: empty access token")
	}
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	path := fmt.Sprintf("/api/web_bets/?limit=%d&page=%d", limit, page)
	env, raw, err := c.doJSONRaw(ctx, "GET", c.cfg.HTTPBase, path, accessToken, nil)
	if err != nil {
		return nil, err
	}
	if err := c.parseEnvelope(env); err != nil {
		return nil, err
	}
	items, err := decodeWebBetList(env.Data, raw)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func decodeWebBetList(data json.RawMessage, raw []byte) ([]WebBetRecord, error) {
	if len(data) > 0 && string(data) != "null" {
		var items []WebBetRecord
		if err := json.Unmarshal(data, &items); err == nil {
			return items, nil
		}
	}
	var wrap struct {
		Data []WebBetRecord `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Data) > 0 {
		return wrap.Data, nil
	}
	return nil, fmt.Errorf("guaji list bets decode: empty data")
}

// GetWebBetRaw 按注单 id 拉取原始 map（含 bet_content 嵌套结构）。
func (c *Client) GetWebBetRaw(ctx context.Context, accessToken, betID string) (map[string]any, error) {
	betID = strings.TrimSpace(betID)
	if betID == "" {
		return nil, fmt.Errorf("guaji get bet raw: empty id")
	}
	for page := 1; page <= 3; page++ {
		_, raw, err := c.FetchWebBetsRaw(ctx, accessToken, 50, page)
		if err != nil {
			return nil, err
		}
		var wrap struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(raw, &wrap); err != nil {
			return nil, err
		}
		for _, row := range wrap.Data {
			if strconv.FormatInt(int64(intNumAny(row["id"])), 10) == betID {
				return row, nil
			}
		}
		if len(wrap.Data) < 50 {
			break
		}
	}
	return nil, fmt.Errorf("guaji get bet raw: id %s not found", betID)
}

func intNumAny(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	default:
		return 0
	}
}

// GetWebBet 按注单 id 查询（实测 GET /api/web_bets/{id} 404，回退列表扫描）。
func (c *Client) GetWebBet(ctx context.Context, accessToken, betID string) (*WebBetRecord, error) {
	betID = strings.TrimSpace(betID)
	if betID == "" {
		return nil, fmt.Errorf("guaji get bet: empty id")
	}
	var env envelope
	if err := c.doJSON(ctx, "GET", c.cfg.HTTPBase, "/api/web_bets/"+betID, accessToken, nil, &env); err == nil {
		if err := c.parseEnvelope(env); err == nil {
			var item WebBetRecord
			if err := env.dataInto(&item); err == nil && item.ID != 0 {
				return &item, nil
			}
			var direct WebBetRecord
			if len(env.Data) > 0 && json.Unmarshal(env.Data, &direct) == nil && direct.ID != 0 {
				return &direct, nil
			}
		}
	}
	items, err := c.ListWebBets(ctx, accessToken, 30, 1)
	if err != nil {
		return nil, err
	}
	want := betID
	for _, it := range items {
		if strconv.FormatInt(it.ID, 10) == want {
			copy := it
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("guaji get bet: id %s not found", betID)
}

// BetSettlement 第三方注单结算结果（T5 派奖同步）。
type BetSettlement struct {
	ThirdPartyBetID string  `json:"id"`
	Status          string  `json:"status"`
	Payout          float64 `json:"payout"`
	Pnl             float64 `json:"pnl"`
	Settled         bool    `json:"settled"`
}

// QuerySettlement 查询第三方注单结算结果（T5；优先列表项字段 net_amount/payout_amount/settled）。
func (c *Client) QuerySettlement(ctx context.Context, accessToken, thirdPartyBetID string) (*BetSettlement, error) {
	if !c.cfg.Enabled {
		return nil, ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if accessToken == "" || thirdPartyBetID == "" {
		return nil, fmt.Errorf("guaji query settlement: token/betId 不能为空")
	}
	item, err := c.GetWebBet(ctx, accessToken, thirdPartyBetID)
	if err != nil {
		return nil, err
	}
	return webBetToSettlement(item), nil
}

func webBetToSettlement(item *WebBetRecord) *BetSettlement {
	if item == nil {
		return nil
	}
	pnl := item.NetAmount
	// net≈0 但毛派奖明显高于本金时，用毛−本金作净额（避免漏记赢）
	if absFloat(pnl) < 0.01 && item.BetAmount > 0 && item.PayoutAmount > item.BetAmount+0.01 {
		pnl = item.PayoutAmount - item.BetAmount
	}
	status := "lose"
	switch {
	case pnl > 1e-6:
		status = "win"
	case item.PayoutAmount > item.BetAmount+1e-6:
		status = "win"
	// 直选组合嵌套小奖：派奖>0 但小于本金、净额为负 —— 仍记 win（勿与和局退本混淆：和局 payout≈本金）
	case item.PayoutAmount > 1e-6 && absFloat(item.PayoutAmount-item.BetAmount) > 0.01:
		status = "win"
		if absFloat(pnl) < 0.01 {
			pnl = item.PayoutAmount - item.BetAmount
		}
	}
	return &BetSettlement{
		ThirdPartyBetID: strconv.FormatInt(item.ID, 10),
		Status:          status,
		Payout:          item.PayoutAmount,
		Pnl:             pnl,
		Settled:         item.Settled,
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
