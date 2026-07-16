package guaji

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// DrawEvent 是一条彩种线开奖（T3，实测协议）。
//
// 真实开奖 WS（wss://…/ws）一条 lottery_v2_broadcast 消息含一个区块、多个彩种线
// （lottery_logXXX，各自 periods），以及该区块衍生的多玩法号码字段（共享）。
// ParseDrawEvents 把一条消息拆成多个 DrawEvent：
//   - lottery_v2_broadcast 内每个 lottery_logXXX 一条，另附 type 本身一条（波场 3 秒）
//   - lottery1/3/5_wsds 独立 type 各一条（波场 1/3/5 分 00 区块）
// 号码字段全部带上，由 drawsync 按彩种 play_template 选取对应玩法号码。
type DrawEvent struct {
	GameKey     string    // 彩种线键，如 "lottery_log033"（反查 outbound_lottery_code）
	Periods     string    // 该彩种线期号
	NextPeriods string    // 下一期期号（可投注期）
	DrawnAt     time.Time // 开奖时间
	Balls       DrawBalls // 该消息全玩法号码（共享，按 template 选）
}

// DrawBalls 一个区块衍生的各玩法开奖号（实测字段）。
type DrawBalls struct {
	SSC     string // last5_num（极速 5 位连写）
	SYXW    string // last11_5_num（11选5）
	PK10    string // last_pk10_num
	K3      string // last_k3_num
	LHC     string // lhc_num（六合彩 7 个）
	ETH10_6 string // last10_6_num（新以太坊）
	TW5     string // last_tw5_num
	TWPK10  string // last_tw_pk10_num
	TW28    string // last_tw28_num
}

// BallsFor 按 play_template 返回对应玩法的开奖号数组。
func (b DrawBalls) BallsFor(playTemplate string) []string {
	switch playTemplate {
	case "ssc_std", "fast_ssc_std":
		if c := splitDrawCodes(b.SSC); len(c) > 0 {
			return c
		}
		return splitDrawCodes(b.TW5)
	case "syxw_std":
		return splitDrawCodes(b.SYXW)
	case "pk10_std":
		if c := splitDrawCodes(b.PK10); len(c) > 0 {
			return c
		}
		return splitDrawCodes(b.TWPK10)
	case "k3_std":
		return splitDrawCodes(b.K3)
	case "lhc_std":
		return splitDrawCodes(b.LHC)
	case "pc28_std":
		return splitDrawCodes(b.TW28)
	default:
		return nil
	}
}

// 忽略彩种关键词（§1.2 不做：福彩 3D、福彩排列 3D、排列 2/3）+ 心跳类型。
var IgnoredDrawKeywords = []string{
	"fc3d", "fc_pl3d", "pl35", "pl3d", "福彩", "排列",
}

var ignoredMessageTypes = map[string]bool{
	"block":              true,
	"block-new":          true,
	"long_dragon_update": true,
	"fc3d_lottery_v2_broadcast":    true,
	"pl35_lottery_v2_broadcast":    true,
	"fc_pl3d_lottery_v2_broadcast": true,
}

// IsIgnoredDrawGame 判断是否为忽略彩种（按名称/码关键词）。
func IsIgnoredDrawGame(gameID, gameName string) bool {
	hay := strings.ToLower(gameID + " " + gameName)
	for _, kw := range IgnoredDrawKeywords {
		if strings.Contains(hay, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// flatDrawMessageTypes：独立消息类型（期号在根上，非 lottery_v2_broadcast 内嵌键）。
// 文档 §7.3 / 前端 00 区块：波场 1/3/5 分彩 → lottery1_wsds / lottery3_wsds / lottery5_wsds。
var flatDrawMessageTypes = map[string]bool{
	"lottery1_wsds": true,
	"lottery3_wsds": true,
	"lottery5_wsds": true,
}

// ParseDrawEvents 解析一条 WS 文本消息，返回 0..N 条彩种线开奖。
// 非开奖/心跳/忽略彩种返回空切片。
func ParseDrawEvents(raw []byte) []DrawEvent {
	body := raw
	var envlp struct {
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(raw, &envlp); err == nil && len(envlp.Message) > 0 {
		body = envlp.Message
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return nil
	}
	typ := looseStr(m["type"])
	if typ == "" || ignoredMessageTypes[typ] {
		return nil
	}

	balls := DrawBalls{
		SSC:     looseStr(m["last5_num"]),
		SYXW:    looseStr(m["last11_5_num"]),
		PK10:    looseStr(m["last_pk10_num"]),
		K3:      looseStr(m["last_k3_num"]),
		LHC:     looseStr(m["lhc_num"]),
		ETH10_6: looseStr(m["last10_6_num"]),
		TW5:     looseStr(m["last_tw5_num"]),
		TWPK10:  looseStr(m["last_tw_pk10_num"]),
		TW28:    looseStr(m["last_tw28_num"]),
	}
	drawnAt := parseDrawTime(looseStr(m["created"]))

	// 波场 1/3/5 分彩（00 区块）：独立 type，期号在根字段。
	if flatDrawMessageTypes[typ] {
		periods := looseStr(m["periods"])
		if periods == "" {
			return nil
		}
		return []DrawEvent{{
			GameKey:     typ,
			Periods:     periods,
			NextPeriods: looseStr(m["next_periods"]),
			DrawnAt:     drawnAt,
			Balls:       balls,
		}}
	}

	var out []DrawEvent
	for key, rawVal := range m {
		if !isLotteryLogKey(key) {
			continue
		}
		var blk struct {
			Periods     string `json:"periods"`
			NextPeriods string `json:"next_periods"`
		}
		if err := json.Unmarshal(rawVal, &blk); err != nil || blk.Periods == "" {
			continue
		}
		out = append(out, DrawEvent{
			GameKey:     key,
			Periods:     blk.Periods,
			NextPeriods: blk.NextPeriods,
			DrawnAt:     drawnAt,
			Balls:       balls,
		})
	}

	// 波场 3 秒彩：每条 lottery_v2_broadcast ≈ 一个区块；GameKey 用 type 本身。
	// 期号优先根 periods；否则用 block_num（前端 bcsmc / blockSpace=1）。
	if typ == "lottery_v2_broadcast" {
		periods := looseStr(m["periods"])
		next := looseStr(m["next_periods"])
		if periods == "" {
			if bn := looseStr(m["block_num"]); bn != "" {
				periods = bn
				if n, err := strconv.ParseInt(bn, 10, 64); err == nil {
					next = strconv.FormatInt(n+1, 10)
				}
			}
		}
		if periods != "" {
			out = append(out, DrawEvent{
				GameKey:     typ,
				Periods:     periods,
				NextPeriods: next,
				DrawnAt:     drawnAt,
				Balls:       balls,
			})
		}
	}
	return out
}

// isLotteryLogKey 识别彩种线期号块键：lottery_logXXX / eth_lottery_logXX / bsc_lottery_logXX / tw_lottery_log。
func isLotteryLogKey(k string) bool {
	if strings.Contains(k, "fc3d") || strings.Contains(k, "pl35") || strings.Contains(k, "pl3d") {
		return false
	}
	return strings.Contains(k, "lottery_log")
}

// SumBalls 求数字球之和（非数字球忽略）。
func SumBalls(balls []string) int {
	sum := 0
	for _, b := range balls {
		if n, err := strconv.Atoi(strings.TrimSpace(b)); err == nil {
			sum += n
		}
	}
	return sum
}

// splitDrawCodes 把 "04,10,11" 或 "44881"（连写数字）拆为球数组。
func splitDrawCodes(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.ContainsAny(s, ", +|") {
		sep := ","
		if !strings.Contains(s, ",") {
			switch {
			case strings.Contains(s, " "):
				sep = " "
			case strings.Contains(s, "+"):
				sep = "+"
			case strings.Contains(s, "|"):
				sep = "|"
			}
		}
		parts := strings.Split(s, sep)
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	// 连写数字（如极速 5 位 "44881"）逐位拆。
	out := make([]string, 0, len(s))
	for _, r := range s {
		out = append(out, string(r))
	}
	return out
}

func looseStr(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.String()
	}
	return strings.Trim(string(raw), `"`)
}

func parseDrawTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now().UTC()
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func wsPathOrDefault(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/ws"
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

// SubscribeDraws 连接开奖 WS（/ws）并持续读取，逐条消息回调 handler（T3）。
func (c *Client) SubscribeDraws(ctx context.Context, handler func([]DrawEvent)) error {
	if !c.cfg.Enabled {
		return ErrMisconfigured("GUAJI_ENABLED=false")
	}
	if err := c.cfg.Valid(); err != nil {
		return err
	}
	u, err := url.Parse(c.cfg.WSBase + wsPathOrDefault(c.cfg.WSPath))
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("token", "Anonymous")
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		NetDialContext:   dialContextPreferHealthy,
		Proxy:            httpProxyFunc(),
	}
	hdr := http.Header{}
	if c.cfg.Origin != "" {
		hdr.Set("Origin", c.cfg.Origin)
	}
	hdr.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	conn, resp, err := dialer.DialContext(ctx, u.String(), hdr)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		events := ParseDrawEvents(raw)
		if len(events) > 0 {
			handler(events)
		}
	}
}
