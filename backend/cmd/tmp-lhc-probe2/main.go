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

type probeCase struct {
	rule, content string
	bets          int
	minBonus      float64
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	key, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	var memberID int64
	_ = pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, "vs8888").Scan(&memberID)
	var tokenEnc string
	_ = pool.QueryRow(ctx, `SELECT access_token_enc FROM member_guaji_accounts WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc)
	token, _ := guaji.DecryptSecret(key, tokenEnc)
	client := guaji.NewClient(cfg.Guaji)
	gameID := 79
	unit := 2.0

	// 65-slot bitmap / index hypotheses for tema (bets_all=65)
	bitmap49 := strings.Repeat("0", 48) + "1" // index 48 = number 49?
	bitmap01 := "1" + strings.Repeat("0", 48)
	slots65 := make([]string, 65)
	for i := range slots65 {
		slots65[i] = "0"
	}
	slots65[0] = "1"
	slots65[1] = "01"

	cases := []probeCase{
		// tema 385 — index / bitmap
		{"385", "0", 1, 47.179},
		{"385", "1", 1, 47.179},
		{"385", bitmap01, 1, 47.179},
		{"385", bitmap49, 1, 47.179},
		{"385", strings.Join(slots65, ","), 1, 47.179},
		{"385", strings.Join(slots65, ""), 1, 47.179},
		{"385", "01", 65, 47.179},
		{"385", "01", 10, 47.179},
		{"385", "01|47.179", 1, 47.179},
		{"385", "47.179", 1, 47.179},
		// zongxiao 301 — odds tiers hint 6 options?
		{"301", "2", 1, 15.3},
		{"301", "3", 1, 3.1},
		{"301", "4", 1, 1.98},
		{"301", "5", 1, 5.5},
		{"301", "6", 1, 15.3},
		{"301", "7", 1, 15.3},
		{"301", "234567", 1, 0},
		{"301", "二三四五六七", 1, 0},
		{"301", "234567肖", 1, 0},
		// tematouwei 307 — odds 4.4|9.4 (head vs tail?)
		{"307", "0", 1, 4.4},
		{"307", "1", 1, 9.4},
		{"307", "头", 1, 4.4},
		{"307", "尾", 1, 9.4},
		{"307", "0|1", 1, 0},
		{"307", "0,1", 2, 0},
		{"307", "01", 1, 0},
		{"307", "00", 1, 0},
		{"307", "10", 1, 0},
		// qima 313 — 32 odds tiers
		{"313", "01,02,03,04,05,06,07", 1, 242.564},
		{"313", "01,02,03,04,05,06,07", 7, 242.564},
		{"313", "1234567", 1, 0},
		{"313", "0", 1, 242.564},
		{"313", "1", 1, 25.3},
	}

	for _, c := range cases {
		amount := unit * float64(c.bets)
		item := guaji.LottBetContent{
			RuleID: c.rule, BetContent: c.content, AmountUnit: unit,
			BetsNums: c.bets, Multiple: 1, BetAmount: amount, Solo: false,
		}
		if c.minBonus > 0 {
			mb := c.minBonus
			item.MinSingleBetBonus = &mb
			s := unit
			item.SingleBetAmount = &s
		}
		_, err := client.PlaceLottBet(ctx, token, guaji.LottBetRequest{
			AutoType: "platform", BetContents: []guaji.LottBetContent{item},
			GameID: gameID, Currency: 3, BetMultiple: []guaji.LottBetMultipleOuter{},
		})
		preview := c.content
		if len(preview) > 40 {
			preview = preview[:40] + "..."
		}
		if err != nil {
			msg := err.Error()
			if i := strings.LastIndex(msg, ": "); i >= 0 {
				msg = msg[i+2:]
			}
			fmt.Printf("FAIL rule=%s bets=%d bonus=%.1f content=%q => %s\n", c.rule, c.bets, c.minBonus, preview, msg)
		} else {
			fmt.Printf("OK   rule=%s bets=%d bonus=%.1f content=%q\n", c.rule, c.bets, c.minBonus, preview)
		}
	}
}
