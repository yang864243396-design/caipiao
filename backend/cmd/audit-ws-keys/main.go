// audit-ws-keys：对照 DB guaji_ws_key、REST 历史期号与 v6hs1 WS 广播 key。
//
// 用法：
//
//	go run ./cmd/audit-ws-keys              # 报告模式
//	go run ./cmd/audit-ws-keys -ci        # CI：DB 配置错误或 live 期号不对齐则 exit 1
//	make guaji-audit-ws-keys              # 同上（需 GUAJI_ENABLED + DB）
//
// 环境：GUAJI_ENABLED=true、DATABASE_URL、可选 GUAJI_WS_BASE。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/historysync"
)

type catRow struct {
	Code, WsKey string
	OnSale      bool
}

type auditResult struct {
	Code, WsKey, RestPath, RestPeriod, WsPeriod, Status, Note string
}

func main() {
	ciMode := flag.Bool("ci", false, "CI 模式：FAIL 时 exit 1")
	wsSec := flag.Int("ws-sec", 60, "WS 采样秒数")
	flag.Parse()

	_ = godotenv.Load()
	cfg := config.Load()
	if !cfg.Guaji.Enabled {
		fmt.Fprintln(os.Stderr, "GUAJI_ENABLED=false，跳过 audit")
		os.Exit(0)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fail(*ciMode, "db: %v", err)
	}
	defer pool.Close()

	cats, err := loadCatalog(ctx, pool)
	if err != nil {
		fail(*ciMode, "catalog: %v", err)
	}

	wsKeys, wsTypes, err := sampleWS(ctx, cfg.Guaji, time.Duration(*wsSec)*time.Second)
	if err != nil {
		fail(*ciMode, "ws: %v", err)
	}

	restPeriod, restErr := sampleREST(ctx, cfg.Guaji, cats)

	results := auditAll(cats, restPeriod, wsKeys)
	printReport(wsTypes, wsKeys, restPeriod, restErr, results)

	okN, badN, warnN, skipN, dbBad := summarize(results)
	fmt.Printf("\nSummary: OK=%d BAD=%d WARN=%d SKIP=%d DB_MISMATCH=%d\n", okN, badN, warnN, skipN, dbBad)

	if *ciMode && (badN > 0 || dbBad > 0) {
		os.Exit(1)
	}
}

func loadCatalog(ctx context.Context, pool *pgxpool.Pool) ([]catRow, error) {
	rows, err := pool.Query(ctx, `
SELECT code, COALESCE(guaji_ws_key,''), on_sale
FROM lottery_catalog WHERE sale_status='on_sale' ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []catRow
	for rows.Next() {
		var r catRow
		if err := rows.Scan(&r.Code, &r.WsKey, &r.OnSale); err != nil {
			return nil, err
		}
		cats = append(cats, r)
	}
	return cats, rows.Err()
}

func sampleWS(ctx context.Context, gcfg guaji.Config, dur time.Duration) (map[string]string, map[string]int, error) {
	u, _ := url.Parse(gcfg.WSBase + "/ws")
	q := u.Query()
	q.Set("token", "Anonymous")
	u.RawQuery = q.Encode()
	hdr := http.Header{}
	hdr.Set("Origin", gcfg.Origin)
	wctx, cancel := context.WithTimeout(ctx, dur)
	defer cancel()
	conn, _, err := websocket.DefaultDialer.DialContext(wctx, u.String(), hdr)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	wsKeys := map[string]string{}
	wsTypes := map[string]int{}
	for wctx.Err() == nil {
		_ = conn.SetReadDeadline(time.Now().Add(12 * time.Second))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		for _, ev := range guaji.ParseDrawEvents(raw) {
			wsKeys[ev.GameKey] = ev.Periods
		}
		body := raw
		var env struct{ Message json.RawMessage `json:"message"` }
		if json.Unmarshal(raw, &env) == nil && len(env.Message) > 0 {
			body = env.Message
		}
		var m map[string]json.RawMessage
		if json.Unmarshal(body, &m) == nil {
			if typ := strings.Trim(string(m["type"]), `"`); typ != "" {
				wsTypes[typ]++
			}
		}
	}
	return wsKeys, wsTypes, nil
}

func sampleREST(ctx context.Context, gcfg guaji.Config, cats []catRow) (map[string]string, map[string]string) {
	client := guaji.NewClient(gcfg)
	restPeriod := map[string]string{}
	restErr := map[string]string{}
	seen := map[string]bool{}
	for _, c := range cats {
		p := historysync.HistoryAPIPathForCode(c.Code)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		rctx, rc := context.WithTimeout(ctx, 12*time.Second)
		logs, err := client.FetchHistoryDrawLogs(rctx, p, 1, 1)
		rc()
		if err != nil || len(logs) == 0 {
			restErr[p] = fmt.Sprintf("%v", err)
			continue
		}
		restPeriod[p] = logs[0].Periods
	}
	return restPeriod, restErr
}

func auditAll(cats []catRow, restPeriod, wsKeys map[string]string) []auditResult {
	var out []auditResult
	for _, c := range cats {
		if !c.OnSale {
			continue
		}
		rp := historysync.HistoryAPIPathForCode(c.Code)
		st, note := classify(c.Code, c.WsKey, rp, restPeriod[rp], wsKeys)
		if exp, ok := expectedWSKey[c.Code]; ok && strings.TrimSpace(c.WsKey) != exp {
			st = "BAD"
			note = fmt.Sprintf("DB key=%q 期望 %q; %s", c.WsKey, exp, note)
		}
		out = append(out, auditResult{
			Code: c.Code, WsKey: c.WsKey, RestPath: rp,
			RestPeriod: restPeriod[rp], Status: st, Note: note,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

func classify(code, wsKey, restPath, restPeriod string, wsKeys map[string]string) (status, note string) {
	if reason, pending := knownPending[code]; pending {
		return "SKIP", reason
	}
	if wsKey == "" {
		return "BAD", "guaji_ws_key 为空"
	}
	if restPath == "" {
		return "WARN", "无 REST history 映射"
	}
	for _, cand := range wsKeyCandidates(wsKey, restPath) {
		wsPeriod, ok := wsKeys[cand]
		if !ok {
			continue
		}
		st, n := matchPeriods(restPeriod, wsPeriod, "OK", "WS="+cand)
		if st == "OK" {
			return st, n
		}
		if status != "OK" {
			status, note = st, n
		}
	}
	if status == "" {
		return "WARN", "WS 采样窗口内未出现 key（可能低频广播）"
	}
	return status, note
}

func matchPeriods(rest, ws, okStatus, prefix string) (string, string) {
	note := strings.TrimSpace(prefix)
	if rest == "" {
		return "WARN", note + "; REST 无数据"
	}
	if ws == "" {
		return "WARN", note + "; WS 无期号"
	}
	if periodAligned(rest, ws) {
		if note != "" {
			return okStatus, note + "; 期号对齐"
		}
		return okStatus, "期号对齐"
	}
	return "BAD", fmt.Sprintf("期号不对齐 REST=%s WS=%s", rest, ws)
}

func periodAligned(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == b {
		return true
	}
	if len(a) >= 4 && len(b) >= 4 && a[:4] == b[:4] {
		n := 6
		if len(a) < n {
			n = len(a)
		}
		if len(b) < n {
			n = len(b)
		}
		if a[len(a)-n:] == b[len(b)-n:] {
			return true
		}
		if abs(len(a)-len(b)) <= 1 {
			return true
		}
	}
	return false
}

func summarize(results []auditResult) (okN, badN, warnN, skipN, dbBad int) {
	for _, r := range results {
		switch r.Status {
		case "OK":
			okN++
		case "BAD":
			badN++
			if strings.Contains(r.Note, "期望") {
				dbBad++
			}
		case "WARN":
			warnN++
		default:
			skipN++
		}
	}
	return
}

func printReport(wsTypes map[string]int, wsKeys, restPeriod, restErr map[string]string, results []auditResult) {
	fmt.Printf("=== WS types ===\n")
	for _, t := range sortedKeys(wsTypes) {
		fmt.Printf("  %s x%d\n", t, wsTypes[t])
	}
	fmt.Println("\n=== WS keys (latest periods) ===")
	for _, k := range sortedKeysStr(wsKeys) {
		fmt.Printf("  %-22s %s\n", k, wsKeys[k])
	}
	fmt.Println("\n=== REST paths ===")
	for _, p := range sortedKeysStr(restPeriod) {
		fmt.Printf("  %-22s %s\n", p, restPeriod[p])
	}
	for p, e := range restErr {
		fmt.Printf("  %-22s ERR=%s\n", p, e)
	}
	fmt.Println("\n=== Audit (on_sale) ===")
	fmt.Printf("%-18s %-20s %-18s %-6s %s\n", "code", "guaji_ws_key", "REST", "status", "note")
	for _, r := range results {
		wsP := ""
		for _, cand := range wsKeyCandidates(r.WsKey, r.RestPath) {
			if wsKeys[cand] != "" {
				wsP = wsKeys[cand]
				break
			}
		}
		_ = wsP
		fmt.Printf("%-18s %-20s %-18s %-6s %s\n", r.Code, r.WsKey, r.RestPath, r.Status, r.Note)
	}
}

func fail(ci bool, format string, args ...any) {
	fmt.Fprintf(os.Stderr, "audit-ws-keys: "+format+"\n", args...)
	if ci {
		os.Exit(1)
	}
	os.Exit(0)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeysStr(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
