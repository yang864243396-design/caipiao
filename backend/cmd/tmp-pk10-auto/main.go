package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()
	pool, _ := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	defer pool.Close()
	account := "vs8888"
	key, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	var memberID int64
	_ = pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, account).Scan(&memberID)
	var tokenEnc string
	_ = pool.QueryRow(ctx, `SELECT access_token_enc FROM member_guaji_accounts WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc)
	token, _ := guaji.DecryptSecret(key, tokenEnc)
	client := guaji.NewClient(cfg.Guaji)

	for _, auto := range []string{"platform", "web", "mobile", "hash", "probe", ""} {
		for _, cur := range []int{3, 0, 1} {
			_, err := client.PlaceLottBet(ctx, token, guaji.LottBetRequest{
				AutoType: auto,
				BetContents: []guaji.LottBetContent{{
					RuleID: "221", BetContent: "大", AmountUnit: 2, BetsNums: 1, Multiple: 1, BetAmount: 2,
				}},
				GameID: 55, Currency: cur, BetMultiple: []guaji.LottBetMultipleOuter{},
			})
			detail := "OK"
			if err != nil {
				detail = err.Error()
				if i := strings.Index(detail, "40000:"); i >= 0 {
					detail = strings.TrimSpace(detail[i+6:])
				} else if i := strings.Index(detail, "40055"); i >= 0 {
					detail = "封盘"
				}
			}
			fmt.Printf("auto=%-10s currency=%d %s\n", auto, cur, detail)
		}
	}
}
