// fetch-periods：拉取指定 game_id 的 /api/web_bets/lott/periods 原始响应。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	gameID := flag.Int("game", 77, "game_id")
	lottery := flag.String("lottery", "tron_ffc_15s", "lottery code for PickOpenLottPeriod")
	account := flag.String("account", "vs8888", "member account")
	num := flag.Int("num", 5, "num_periods")
	flag.Parse()

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	token, err := loadToken(ctx, pool, cfg, *account)
	if err != nil {
		panic(err)
	}

	client := guaji.NewClient(cfg.Guaji)
	periods, raw, err := client.FetchLottPeriods(ctx, token, *gameID, *num)
	fmt.Println("=== raw JSON ===")
	fmt.Println(string(raw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode err: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n=== decoded %d periods ===\n", len(periods))
	for i, p := range periods {
		fmt.Printf("[%d] period=%s start=%s end=%s\n", i, p.Period, p.StartTime, p.EndTime)
	}
	now := time.Now()
	open, closeAt, ok := guaji.PickOpenLottPeriod(periods, *lottery, now)
	fmt.Printf("\n=== PickOpenLottPeriod(lottery=%s now=%s) ===\n", *lottery, now.UTC().Format(time.RFC3339))
	if !ok {
		fmt.Println("no open period (would skip matrix: 当前无开盘期号)")
	} else {
		fmt.Printf("open period=%s closeAt=%s\n", open.Period, closeAt.UTC().Format(time.RFC3339))
	}
}

func loadToken(ctx context.Context, pool *db.Pool, cfg config.Config, account string) (string, error) {
	key, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	var memberID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, account).Scan(&memberID); err != nil {
		return "", err
	}
	var tokenEnc string
	if err := pool.QueryRow(ctx, `
SELECT access_token_enc FROM member_guaji_accounts
WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc); err != nil {
		return "", err
	}
	return guaji.DecryptSecret(key, tokenEnc)
}
