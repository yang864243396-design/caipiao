// 压测第三方 Guaji HTTP 并发能力（periods / rate / 可选最小真下单）。
//
//	go run ./cmd/guaji-qps-bench -mode=periods -concurrency=256 -total=512 -account=v6ceshi01
//	go run ./cmd/guaji-qps-bench -mode=bet -concurrency=32 -total=32 -account=v6ceshi01
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	mode := flag.String("mode", "periods", "periods|rate|bet")
	account := flag.String("account", "v6ceshi01", "member account with guaji token")
	concurrency := flag.Int("concurrency", 256, "in-flight goroutines")
	total := flag.Int("total", 512, "total requests")
	gameID := flag.Int("game", 27, "game_id for periods/bet")
	timeoutSec := flag.Int("timeout", 45, "per-request timeout seconds")
	maxConns := flag.Int("max-conns", 0, "HTTP MaxConnsPerHost (0=concurrency)")
	flag.Parse()

	if *concurrency <= 0 || *total <= 0 {
		fmt.Fprintln(os.Stderr, "concurrency/total must be > 0")
		os.Exit(2)
	}
	if *maxConns <= 0 {
		*maxConns = *concurrency
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	client := guaji.NewClient(cfg.Guaji)
	if !client.Enabled() {
		fmt.Fprintln(os.Stderr, "GUAJI_ENABLED=false")
		os.Exit(1)
	}
	// 覆盖默认 MaxConnsPerHost=32，否则测不出 256 真并发。
	client.TuneHTTPConcurrency(*maxConns)

	key, err := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	if err != nil {
		fmt.Fprintln(os.Stderr, "credentials key:", err)
		os.Exit(1)
	}
	var memberID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, *account).Scan(&memberID); err != nil {
		fmt.Fprintln(os.Stderr, "member:", err)
		os.Exit(1)
	}
	var tokenEnc string
	err = pool.QueryRow(ctx, `
SELECT access_token_enc FROM member_guaji_accounts
WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "guaji token:", err)
		os.Exit(1)
	}
	token, err := guaji.DecryptSecret(key, tokenEnc)
	if err != nil {
		fmt.Fprintln(os.Stderr, "decrypt token:", err)
		os.Exit(1)
	}

	fmt.Printf("mode=%s account=%s concurrency=%d total=%d maxConnsPerHost=%d game=%d base=%s\n",
		*mode, *account, *concurrency, *total, *maxConns, *gameID, cfg.Guaji.HTTPBase)

	type result struct {
		err  error
		ms   float64
		kind string
	}
	results := make([]result, *total)
	var inflight atomic.Int32
	var peak atomic.Int32

	sem := make(chan struct{}, *concurrency)
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < *total; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			cur := inflight.Add(1)
			for {
				old := peak.Load()
				if cur <= old || peak.CompareAndSwap(old, cur) {
					break
				}
			}
			defer inflight.Add(-1)

			reqCtx, cancel := context.WithTimeout(ctx, time.Duration(*timeoutSec)*time.Second)
			t0 := time.Now()
			var callErr error
			switch strings.ToLower(*mode) {
			case "periods":
				_, _, callErr = client.FetchLottPeriods(reqCtx, token, *gameID, 2)
			case "rate":
				_, callErr = client.FetchRealRate(reqCtx, token)
			case "bet":
				callErr = placeMinBet(reqCtx, client, token, *gameID)
			default:
				callErr = fmt.Errorf("unknown mode %q", *mode)
			}
			cancel()
			results[idx] = result{err: callErr, ms: float64(time.Since(t0).Microseconds()) / 1000, kind: classifyErr(callErr)}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	ok := 0
	byKind := map[string]int{}
	lat := make([]float64, 0, *total)
	for _, r := range results {
		byKind[r.kind]++
		if r.err == nil {
			ok++
			lat = append(lat, r.ms)
		}
	}
	sort.Float64s(lat)

	fmt.Printf("elapsed=%s peakInflight=%d ok=%d/%d effective_qps=%.1f\n",
		elapsed.Round(time.Millisecond), peak.Load(), ok, *total, float64(*total)/elapsed.Seconds())
	fmt.Printf("latency_ms p50=%s p95=%s p99=%s max=%s\n",
		pct(lat, 50), pct(lat, 95), pct(lat, 99), pct(lat, 100))
	fmt.Println("errors_by_kind:")
	kinds := make([]string, 0, len(byKind))
	for k := range byKind {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Printf("  %-24s %d\n", k, byKind[k])
	}
	if ok < *total {
		n := 0
		for _, r := range results {
			if r.err == nil {
				continue
			}
			fmt.Printf("sample_err: %v\n", r.err)
			n++
			if n >= 8 {
				break
			}
		}
		os.Exit(2)
	}
}

func placeMinBet(ctx context.Context, c *guaji.Client, token string, gameID int) error {
	unit := 1.0
	mult := 1
	betsNums := 1
	amount := unit * float64(betsNums) * float64(mult)
	_, err := c.PlaceLottBet(ctx, token, guaji.LottBetRequest{
		AutoType: "platform",
		GameID:   gameID,
		Currency: 3,
		BetContents: []guaji.LottBetContent{{
			RuleID:     "3",
			BetContent: "0",
			AmountUnit: unit,
			BetsNums:   betsNums,
			Multiple:   mult,
			BetAmount:  amount,
			Solo:       true,
		}},
		BetMultiple: []guaji.LottBetMultipleOuter{},
	})
	return err
}

func classifyErr(err error) string {
	if err == nil {
		return "ok"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
		return "timeout"
	case strings.Contains(msg, "connection refused") || strings.Contains(msg, "connectex"):
		return "connect"
	case strings.Contains(msg, "429") || strings.Contains(msg, "too many") || strings.Contains(msg, "rate limit"):
		return "rate_limit"
	case strings.Contains(msg, "40055") || strings.Contains(msg, "封盘"):
		return "period_closed"
	case strings.Contains(msg, "empty data"):
		return "empty_data"
	case strings.Contains(msg, "40000"):
		return "game_off"
	case strings.Contains(msg, "401") || strings.Contains(msg, "unauthorized"):
		return "auth"
	case strings.Contains(msg, "guaji http"):
		return "http"
	default:
		return "other"
	}
}

func pct(sorted []float64, p int) string {
	if len(sorted) == 0 {
		return "-"
	}
	if p >= 100 {
		return fmt.Sprintf("%.1f", sorted[len(sorted)-1])
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return fmt.Sprintf("%.1f", sorted[idx])
}
