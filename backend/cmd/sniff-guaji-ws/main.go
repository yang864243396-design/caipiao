// 一次性诊断：抓取第三方开奖 WS 原始消息，列出 lottery_logXXX 键与 tron_ffc_1m 期号对照。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/guaji"
)

var lotteryLogKeyRe = regexp.MustCompile(`"(lottery_log[^"]+|eth_lottery_log[^"]*|bsc_lottery_log[^"]*|tw_lottery_log[^"]*)"\s*:`)

func main() {
	_ = godotenv.Load()
	cfg := config.Load().Guaji
	if !cfg.Enabled {
		fmt.Fprintln(os.Stderr, "GUAJI_ENABLED=false")
		os.Exit(1)
	}
	if err := cfg.Valid(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 拉一条 tron_ffc_1m 历史 REST 作期号参照
	refPeriod := fetchTronFfc1mReferencePeriod(cfg)

	u, _ := url.Parse(cfg.WSBase + wsPath(cfg.WSPath))
	q := u.Query()
	q.Set("token", "Anonymous")
	u.RawQuery = q.Encode()

	fmt.Printf("WS: %s\n", u.String())
	if refPeriod != "" {
		fmt.Printf("tron_ffc_1m REST 最近期号参照: %s\n\n", refPeriod)
	}

	hdr := http.Header{}
	hdr.Set("Origin", cfg.Origin)
	hdr.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0 Safari/537.36")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	conn, resp, err := dialer.DialContext(ctx, u.String(), hdr)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial:", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("已连接，等待 lottery_v2_broadcast ...")

	for {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, "timeout: 120s 内未收到含 lottery_log 的开奖广播")
			os.Exit(1)
		}
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			fmt.Fprintln(os.Stderr, "read:", err)
			os.Exit(1)
		}

		body := unwrapMessage(raw)
		typ := jsonGetString(body, "type")
		if typ != "" && typ != "lottery_v2_broadcast" {
			continue
		}

		keys := lotteryLogKeyRe.FindAllStringSubmatch(string(body), -1)
		if len(keys) == 0 {
			// 也尝试外层 type
			if typ == "" {
				continue
			}
		}

		events := guaji.ParseDrawEvents(raw)
		if len(events) == 0 && len(keys) == 0 {
			continue
		}

		fmt.Println("========== 原始消息（截断 4000 字符）==========")
		s := string(body)
		if len(s) > 4000 {
			s = s[:4000] + "..."
		}
		fmt.Println(s)

		fmt.Println("\n========== lottery_log* 键 ==========")
		seen := map[string]bool{}
		for _, m := range keys {
			if m[1] != "" && !seen[m[1]] {
				seen[m[1]] = true
				period := extractPeriod(body, m[1])
				mark := ""
				if refPeriod != "" && period != "" && strings.HasSuffix(refPeriod, period[len(period)-min(6, len(period)):]) {
					mark = "  ← 期号后缀与 tron_ffc_1m REST 接近"
				}
				if period != "" && strings.HasPrefix(period, "1014") {
					mark += "  ← 1014* 波场秒级/1分区块线"
				}
				fmt.Printf("  %s  periods=%s%s\n", m[1], period, mark)
			}
		}

		fmt.Println("\n========== ParseDrawEvents 解析 ==========")
		for _, ev := range events {
			balls := ev.Balls.BallsFor("ssc_std")
			mark := ""
			if refPeriod != "" && ev.Periods == refPeriod {
				mark = "  ★ 与 tron_ffc_1m REST 期号完全一致"
			} else if refPeriod != "" && strings.HasPrefix(ev.Periods, "1014") {
				mark = "  ? 1014* 候选"
			}
			fmt.Printf("  gameKey=%-20s periods=%-16s next=%-16s ssc=%v%s\n",
				ev.GameKey, ev.Periods, ev.NextPeriods, balls, mark)
		}

		fmt.Println("\n========== 结论 ==========")
		fmt.Println("  DB guaji_ws_key: tron_ffc_1m = lottery_logs (migration 00116)")
		fmt.Println("  若上方无 lottery_logs 而有其它键 periods 对齐 1014*，则需改 guaji_ws_key。")
		return
	}
}

func fetchTronFfc1mReferencePeriod(cfg guaji.Config) string {
	c := guaji.NewClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	logs, err := c.FetchHistoryDrawLogs(ctx, "lottery_logs", 1, 3)
	if err != nil || len(logs) == 0 {
		return ""
	}
	return strings.TrimSpace(logs[0].Periods)
}

func unwrapMessage(raw []byte) []byte {
	var envlp struct {
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(raw, &envlp); err == nil && len(envlp.Message) > 0 {
		return envlp.Message
	}
	return raw
}

func jsonGetString(body []byte, key string) string {
	var m map[string]json.RawMessage
	if json.Unmarshal(body, &m) != nil {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return ""
}

func extractPeriod(body []byte, key string) string {
	var m map[string]json.RawMessage
	if json.Unmarshal(body, &m) != nil {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var blk struct {
		Periods string `json:"periods"`
	}
	if json.Unmarshal(raw, &blk) == nil {
		return strings.TrimSpace(blk.Periods)
	}
	return ""
}

func wsPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/ws"
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
