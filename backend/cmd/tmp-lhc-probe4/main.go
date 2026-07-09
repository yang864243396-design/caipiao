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
	key, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	var memberID int64
	_ = pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, "vs8888").Scan(&memberID)
	var tokenEnc string
	_ = pool.QueryRow(ctx, `SELECT access_token_enc FROM member_guaji_accounts WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc)
	token, _ := guaji.DecryptSecret(key, tokenEnc)
	client := guaji.NewClient(cfg.Guaji)
	gameID := 79
	unit := 2.0

	type c struct{ rule, content string; bets int; bonus float64 }
	var cases []c
	// 301 总肖 — 6 档 odds
	for _, tc := range []struct{ content string; bonus float64; bets int }{
		{"2", 15.3, 1}, {"3", 15.3, 1}, {"4", 3.1, 1}, {"5", 1.98, 1}, {"6", 5.5, 1}, {"7", 15.3, 1},
		{"二", 15.3, 1}, {"三", 15.3, 1}, {"四", 3.1, 1}, {"五", 1.98, 1}, {"六", 5.5, 1}, {"七", 15.3, 1},
		{"2|3", 15.3, 2}, {"2|3|4", 15.3, 3},
		{"234567", 15.3, 1}, {"234567", 15.3, 6},
		{"0|1|2", 15.3, 3},
	} {
		cases = append(cases, c{"301", tc.content, tc.bets, tc.bonus})
	}
	// 313 七码 — 32 档 odds，试命中个数索引
	for i := 0; i <= 7; i++ {
		cases = append(cases, c{"313", fmt.Sprintf("%d", i), 1, 242.564})
		cases = append(cases, c{"313", fmt.Sprintf("%d", i), 1, 25.3})
	}
	for _, content := range []string{"0123456", "1234567", "0,1,2,3,4,5,6", "七", "7中7"} {
		cases = append(cases, c{"313", content, 1, 242.564})
	}

	for _, tc := range cases {
		amount := unit * float64(tc.bets)
		item := guaji.LottBetContent{
			RuleID: tc.rule, BetContent: tc.content, AmountUnit: unit,
			BetsNums: tc.bets, Multiple: 1, BetAmount: amount, Solo: false,
		}
		if tc.bonus > 0 {
			mb := tc.bonus
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
			fmt.Printf("FAIL rule=%s bets=%d bonus=%.1f content=%q => %s\n", tc.rule, tc.bets, tc.bonus, tc.content, msg)
		} else {
			fmt.Printf("OK   rule=%s bets=%d bonus=%.1f content=%q\n", tc.rule, tc.bets, tc.bonus, tc.content)
		}
	}
}
