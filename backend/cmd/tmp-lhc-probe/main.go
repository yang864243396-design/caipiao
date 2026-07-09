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

type caseItem struct {
	rule, content string
	bets          int
	solo          bool
	auto          string
	minBonus      float64
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()
	pool, _ := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	defer pool.Close()
	key, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	var memberID int64
	_ = pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, "vs8888").Scan(&memberID)
	var tokenEnc string
	_ = pool.QueryRow(ctx, `SELECT access_token_enc FROM member_guaji_accounts WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc)
	token, _ := guaji.DecryptSecret(key, tokenEnc)
	client := guaji.NewClient(cfg.Guaji)
	gameID := 79

	cases := []caseItem{
		// tema 385
		{"385", "01", 1, false, "platform", 0},
		{"385", "01", 1, false, "hash", 0},
		{"385", "01", 1, false, "probe", 0},
		{"385", "特码A", 1, false, "platform", 0},
		{"385", "A", 1, false, "platform", 0},
		{"385", "07", 1, false, "platform", 0},
		{"385", "7", 1, false, "platform", 0},
		{"385", "007", 1, false, "platform", 0},
		{"385", "01,02,03", 3, false, "platform", 0},
		{"385", "01,02,03", 1, false, "platform", 0},
		{"385", "01", 1, true, "platform", 42.5},
		{"385", "01", 1, false, "platform", 42.5},
		{"385", "49", 1, false, "platform", 42.5},
		{"385", "25", 1, false, "platform", 0},
		{"385", "马", 1, false, "platform", 0},
		{"385", "鼠", 1, false, "platform", 0},
		{"385", "红", 1, false, "platform", 0},
		{"385", "大", 1, false, "platform", 0},
		{"385", "单", 1, false, "platform", 0},
		// zhengte 271
		{"271", "01", 1, false, "platform", 0},
		{"271", "07", 1, false, "platform", 42.5},
		// zongxiao 301
		{"301", "0", 1, false, "platform", 0},
		{"301", "1", 1, false, "platform", 0},
		{"301", "5", 1, false, "platform", 0},
		{"301", "12", 1, false, "platform", 0},
		{"301", "二", 1, false, "platform", 0},
		{"301", "五", 1, false, "platform", 0},
		{"301", "0肖", 1, false, "platform", 0},
		{"301", "5肖", 1, false, "platform", 0},
		{"301", "总肖5", 1, false, "platform", 0},
		{"301", "012", 1, false, "platform", 0},
		// tematouwei 307
		{"307", "头0", 1, false, "platform", 0},
		{"307", "尾0", 1, false, "platform", 0},
		{"307", "0头", 1, false, "platform", 0},
		{"307", "0尾", 1, false, "platform", 0},
		{"307", "头0,尾1", 2, false, "platform", 0},
		{"307", "头0,尾1", 1, false, "platform", 0},
		{"307", "0头1尾", 1, false, "platform", 0},
		{"307", "头0尾0", 1, false, "platform", 0},
		{"307", "0", 1, false, "platform", 0},
		{"307", "1", 1, false, "platform", 0},
		// qima 313
		{"313", "01,02,03,04,05,06,07", 7, false, "platform", 0},
		{"313", "01,02,03,04,05,06,07", 1, false, "platform", 0},
		{"313", "01", 1, false, "platform", 0},
		{"313", "01020304050607", 1, false, "platform", 0},
		{"313", "1,2,3,4,5,6,7", 7, false, "platform", 0},
		{"313", "七码", 1, false, "platform", 0},
	}

	for _, c := range cases {
		unit := 2.0
		amount := unit * float64(c.bets)
		item := guaji.LottBetContent{
			RuleID: c.rule, BetContent: c.content, AmountUnit: unit,
			BetsNums: c.bets, Multiple: 1, BetAmount: amount, Solo: c.solo,
		}
		if c.minBonus > 0 {
			mb := c.minBonus
			item.MinSingleBetBonus = &mb
			s := unit
			item.SingleBetAmount = &s
		}
		auto := c.auto
		if auto == "" {
			auto = "platform"
		}
		_, err := client.PlaceLottBet(ctx, token, guaji.LottBetRequest{
			AutoType: auto, BetContents: []guaji.LottBetContent{item},
			GameID: gameID, Currency: 3, BetMultiple: []guaji.LottBetMultipleOuter{},
		})
		if err != nil {
			msg := err.Error()
			if i := strings.LastIndex(msg, ": "); i >= 0 {
				msg = msg[i+2:]
			}
			fmt.Printf("FAIL rule=%s auto=%s bets=%d solo=%v bonus=%.1f content=%q => %s\n",
				c.rule, auto, c.bets, c.solo, c.minBonus, c.content, msg)
		} else {
			fmt.Printf("OK   rule=%s auto=%s bets=%d content=%q\n", c.rule, auto, c.bets, c.content)
		}
	}
}
