// guaji-capture：用测试账号直连第三方，打印登录/余额/开奖等原始报文，
// 用于核对真实协议（成功码、字段名、WS 结构）。诊断工具，不参与生产。
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	httpBase := envOr("GUAJI_HTTP_BASE", "https://www.v6hs1.com")
	authBase := envOr("GUAJI_AUTH_BASE", "https://www.v6hs1.com")
	wsBase := envOr("GUAJI_WS_BASE", "wss://www.v6hs1.com")
	origin := envOr("GUAJI_ORIGIN", "https://www.v6hs1.com")
	user := envOr("GUAJI_TEST_USERNAME", "testcq01")
	pass := envOr("GUAJI_TEST_PASSWORD", "testcq01")

	hc := &http.Client{Timeout: 20 * time.Second}

	fmt.Println("=== POST /auth/login ===")
	loginBody := map[string]any{"username": user, "password": pass, "is_ai": true}
	status, raw := postJSON(hc, authBase+"/auth/login", origin, "", loginBody)
	fmt.Printf("HTTP %d\n%s\n\n", status, truncate(raw, 2000))

	token := extractToken(raw)
	if token == "" {
		fmt.Println("未取到 token（可能需 MFA：code=40045 带 login_key + google_code）。停在登录环节。")
	} else {
		fmt.Println("token 前缀:", safePrefix(token, 16))

		fmt.Println("\n=== GET /api/users/i/info ===")
		st2, raw2 := getJSON(hc, httpBase+"/api/users/i/info", origin, token)
		fmt.Printf("HTTP %d\n%s\n\n", st2, truncate(raw2, 2000))

		fmt.Println("=== GET /api/agents/i/real/rate ===")
		st3, raw3 := getJSON(hc, httpBase+"/api/agents/i/real/rate", origin, token)
		fmt.Printf("HTTP %d\n%s\n\n", st3, truncate(raw3, 1500))

		gameID := envOr("GUAJI_CAPTURE_GAME_ID", "29")
		fmt.Printf("=== GET /api/web_bets/lott/periods?game_id=%s ===\n", gameID)
		st4, raw4 := getJSON(hc, httpBase+"/api/web_bets/lott/periods?game_id="+gameID+"&num_periods=3", origin, token)
		fmt.Printf("HTTP %d\n%s\n\n", st4, truncate(raw4, 1500))

		fmt.Println("=== GET /api/web_bets/ （历史注单，确认返回结构）===")
		st5, raw5 := getJSON(hc, httpBase+"/api/web_bets/", origin, token)
		fmt.Printf("HTTP %d\n%s\n\n", st5, truncate(raw5, 2000))

		fmt.Println("=== GET /api/users/i/security_question （密保资讯）===")
		stsq, rawsq := getJSON(hc, httpBase+"/api/users/i/security_question", origin, token)
		fmt.Printf("HTTP %d\n%s\n\n", stsq, truncate(rawsq, 1500))

		if os.Getenv("GUAJI_CAPTURE_SETUP_SECURITY") == "1" {
			fmt.Println("=== POST /auth/login/security （设置资金密保，文档值 147258）===")
			secBody := map[string]any{
				"username":          user,
				"password":          pass,
				"new_password":      "147258",
				"wp_password":       "147258",
				"wp_password2":      "147258",
				"security_question": "1.您的学号(或工号)是?",
				"security_reminder": "147255",
				"security_code":     "147258",
			}
			sts, raws := postJSON(hc, authBase+"/auth/login/security", origin, token, secBody)
			fmt.Printf("HTTP %d\n%s\n\n", sts, truncate(raws, 1500))
		}

		if os.Getenv("GUAJI_CAPTURE_BET") == "1" {
			ruleID := envOr("GUAJI_CAPTURE_RULE_ID", "13")
			betContent := envOr("GUAJI_CAPTURE_BET_CONTENT", ",,,13579,")
			unit := 2.0
			betsNums := 5
			multiple := 1
			amount := float64(betsNums) * unit * float64(multiple) // 与 amount_unit*注数*倍数 一致
			gid := 29
			fmt.Sscanf(gameID, "%d", &gid)
			fmt.Printf("=== POST /api/web_bets/lott （真实下单 game_id=%d rule_id=%s amount=%.0f cny）===\n", gid, ruleID, amount)
			betBody := map[string]any{
				"auto_type": "platform",
				"bet_contents": []map[string]any{{
					"rule_id":     ruleID,
					"bet_content": betContent,
					"amount_unit": unit,
					"bets_nums":   betsNums,
					"multiple":    multiple,
					"bet_amount":  amount,
					"solo":        false,
				}},
				"game_id":      gid,
				"currency":     3, // cny
				"bet_multiple": []any{},
			}
			st6, raw6 := postJSON(hc, httpBase+"/api/web_bets/lott", origin, token, betBody)
			fmt.Printf("HTTP %d\n%s\n\n", st6, truncate(raw6, 2000))
		} else {
			fmt.Println("（下单已跳过：设 GUAJI_CAPTURE_BET=1 才执行真实下单）")
		}
	}

	if os.Getenv("GUAJI_CAPTURE_SKIP_WS") == "1" {
		return
	}
	fmt.Println("=== WS 开奖抓包 + 彩种线汇总（/ws，最多 60 条 / 40s）===")
	summarizeDraws(wsBase+"/ws", origin, "Anonymous", 60, 40*time.Second)
}

// summarizeDraws 连 /ws 抓多条，按 lottery_logXXX 键聚合，输出运营配置参考表。
func summarizeDraws(wsURL, origin, token string, maxMsgs int, dur time.Duration) {
	u, _ := url.Parse(wsURL)
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	hdr := http.Header{}
	if origin != "" {
		hdr.Set("Origin", origin)
	}
	hdr.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, u.String(), hdr)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		fmt.Println("WS dial 失败:", err)
		return
	}
	defer conn.Close()

	type agg struct {
		periods string
		plays   map[string]bool
	}
	keys := map[string]*agg{}
	msgTypes := map[string]int{}
	got := 0
	for got < maxMsgs {
		if ctx.Err() != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(dur))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		got++
		var envl struct {
			Message json.RawMessage `json:"message"`
		}
		body := raw
		if json.Unmarshal(raw, &envl) == nil && len(envl.Message) > 0 {
			body = envl.Message
		}
		var m map[string]json.RawMessage
		if json.Unmarshal(body, &m) != nil {
			continue
		}
		if t := strField(m["type"]); t != "" {
			msgTypes[t]++
		}
		for k, v := range m {
			if !strings.Contains(k, "lottery_log") {
				continue
			}
			var blk struct {
				Periods string `json:"periods"`
			}
			if json.Unmarshal(v, &blk) != nil || blk.Periods == "" {
				continue
			}
			a := keys[k]
			if a == nil {
				a = &agg{plays: map[string]bool{}}
				keys[k] = a
			}
			a.periods = blk.Periods
			for play, field := range map[string]string{"ssc": "last5_num", "syxw": "last11_5_num", "pk10": "last_pk10_num", "k3": "last_k3_num", "lhc": "lhc_num", "tw28": "last_tw28_num"} {
				if strField(m[field]) != "" {
					a.plays[play] = true
				}
			}
		}
	}
	fmt.Printf("\n抓到 %d 条消息。消息类型计数：%v\n", got, msgTypes)
	fmt.Println("\n彩种线键（lottery_logXXX）→ 最近期号 / 含玩法（供 outbound_lottery_code 配置参考）：")
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		a := keys[k]
		plays := make([]string, 0, len(a.plays))
		for p := range a.plays {
			plays = append(plays, p)
		}
		sort.Strings(plays)
		fmt.Printf("  %-22s periods=%-18s plays=%v\n", k, a.periods, plays)
	}
}

func strField(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return strings.Trim(string(raw), `"`)
}

func postJSON(hc *http.Client, u, origin, bearer string, body any) (int, string) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
		req.Header.Set("Referer", origin+"/")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return 0, "ERR: " + err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, string(raw)
}

func getJSON(hc *http.Client, u, origin, bearer string) (int, string) {
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Accept", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
		req.Header.Set("Referer", origin+"/")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return 0, "ERR: " + err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, string(raw)
}

func captureWS(wsURL, origin, token string, maxMsgs int, dur time.Duration) bool {
	u, err := url.Parse(wsURL)
	if err != nil {
		fmt.Println("ws url err:", err)
		return false
	}
	q := u.Query()
	if q.Get("token") == "" {
		q.Set("token", token)
	}
	u.RawQuery = q.Encode()

	hdr := http.Header{}
	if origin != "" {
		hdr.Set("Origin", origin)
	}
	// 模拟浏览器，规避 CDN/UA 拦截
	hdr.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	conn, resp, err := dialer.DialContext(ctx, u.String(), hdr)
	if err != nil {
		code := -1
		if resp != nil {
			code = resp.StatusCode
		}
		fmt.Printf("WS dial 失败: %v (http %d)\n", err, code)
		return false
	}
	defer conn.Close()
	got := 0
	for got < maxMsgs {
		if ctx.Err() != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(dur))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read err:", err)
			break
		}
		got++
		fmt.Printf("--- msg #%d ---\n%s\n", got, truncate(string(msg), 1500))
	}
	if got == 0 {
		fmt.Println("WS 已连接但无消息。")
	}
	return got > 0
}

func extractToken(raw string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	if t, ok := m["token"].(string); ok && t != "" {
		return t
	}
	if data, ok := m["data"].(map[string]any); ok {
		if t, ok := data["token"].(string); ok {
			return t
		}
	}
	return ""
}

func envOr(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + " …(truncated)"
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
