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
	unit := 2.0

	var cases []probeCase
	// 307 pipe combos (head|tail index?)
	for _, content := range []string{"0|0", "0|1", "1|0", "2|3", "4|9", "0|", "|0", "0|0|0"} {
		for _, bets := range []int{1, 2} {
			cases = append(cases, probeCase{"307", content, bets, 0})
		}
	}
	// 307 with min_bonus matching odds tiers 4.4 / 9.4
	for _, tc := range []struct{ c string; b float64 }{
		{"0|0", 4.4}, {"0|1", 4.4}, {"1|0", 9.4}, {"2|5", 4.4},
	} {
		cases = append(cases, probeCase{"307", tc.c, 1, tc.b})
		cases = append(cases, probeCase{"307", tc.c, 2, tc.b})
	}
	// 301 — try pipe / multi
	for _, content := range []string{"5", "5|", "|5", "2|3|4", "0123456", "0123456789ABC"} {
		cases = append(cases, probeCase{"301", content, 1, 15.3})
	}
	// 313 — pipe / count index
	for _, content := range []string{"0", "1", "7", "0|1|2|3|4|5|6", "01|02|03|04|05|06|07"} {
		for _, bets := range []int{1, 7} {
			cases = append(cases, probeCase{"313", content, bets, 242.564})
		}
	}
	// 385 — try pipe sides like duipeng / tuotou patterns
	for _, content := range []string{
		"01|", "|01", "01|01", "马|01", "01|47.179",
		"01,02|03,04", "01#02", "01;02",
	} {
		cases = append(cases, probeCase{"385", content, 1, 47.179})
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
		if err != nil {
			msg := err.Error()
			if i := strings.LastIndex(msg, ": "); i >= 0 {
				msg = msg[i+2:]
			}
			fmt.Printf("FAIL rule=%s bets=%d bonus=%.1f content=%q => %s\n", c.rule, c.bets, c.minBonus, c.content, msg)
		} else {
			fmt.Printf("OK   rule=%s bets=%d bonus=%.1f content=%q\n", c.rule, c.bets, c.minBonus, c.content)
		}
	}
}
