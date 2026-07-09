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

func pairsBig() []string {
	out := []string{}
	for a := 1; a <= 10; a++ {
		for b := 1; b <= 10; b++ {
			if a != b && a+b >= 12 {
				out = append(out, fmt.Sprintf("%02d%02d", a, b))
			}
		}
	}
	return out
}

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

	bigPairs := pairsBig()
	bigWire := strings.Join(bigPairs, ",")
	bigSeg0 := strings.Join(bigPairs, "")

	type probe struct {
		rule, content, tag string
		bets               int
		bonus              float64
	}
	cases := []probe{
		// baseline: user capture pattern (rule 201)
		{"201", "0607080910,0607080910,,,,,,,,", "201-ref", 10, 19.4},
		// 221 with hash + 10-position rank pools (大=6-10)
		{"221", "0607080910,0607080910,,,,,,,,", "221-da-rank10", 25, 4.5},
		{"221", "0607080910,0607080910,,,,,,,,", "221-da-rank40", 40, 4.5},
		{"221", "0102030405,0102030405,,,,,,,,", "221-xiao-rank", 25, 3.6},
		{"221", "0102030405,0102030405,,,,,,,,", "221-xiao-50", 50, 3.6},
		// index in first segment only
		{"221", "0,,,,,,,,,", "221-idx0", 1, 4.5},
		{"221", "2,,,,,,,,,", "221-idx2", 1, 4.5},
		{"221", "02,,,,,,,,,", "221-idx02", 1, 4.5},
		{"221", "00,,,,,,,,,", "221-idx00", 1, 4.5},
		// pair list comma
		{"221", bigWire, "221-big-pairs", 40, 4.5},
		// all pairs concatenated in seg0
		{"221", bigSeg0 + ",,,,,,,,,", "221-big-flat0", 40, 4.5},
		{"221", bigSeg0, "221-big-flat-only", 40, 4.5},
		// chinese 10-pos
		{"221", "\u5927,\u5927,,,,,,,,,", "221-da-cn10", 1, 4.5},
		{"221", "\u5927,,,,,,,,,", "221-da-cn1", 1, 4.5},
		// hezhi-style sums for 大
		{"221", "1213141516171819,,,,,,,,,", "221-sums-big", 8, 4.5},
		{"221", "1213141516171819", "221-sums-flat", 8, 4.5},
	}

	for _, p := range cases {
		single := 2.0
		item := guaji.LottBetContent{
			RuleID: p.rule, BetContent: p.content, AmountUnit: 2, BetsNums: p.bets, Multiple: 1,
			BetAmount: 2 * float64(p.bets), Solo: false,
			SingleBetAmount: &single,
		}
		if p.bonus > 0 {
			item.MinSingleBetBonus = &p.bonus
		}
		_, err := client.PlaceLottBet(ctx, token, guaji.LottBetRequest{
			AutoType: "hash", BetContents: []guaji.LottBetContent{item},
			GameID: 53, Currency: 3,
			BetMultiple: []guaji.LottBetMultipleOuter{{BetAmount: 2 * float64(p.bets), Multiple: 1}},
		})
		if err != nil {
			msg := err.Error()
			for _, code := range []string{"40000:", "40055:", "40052:"} {
				if i := strings.Index(msg, code); i >= 0 {
					msg = strings.TrimSpace(msg[i+len(code):])
					break
				}
			}
			fmt.Printf("FAIL [%s] rule=%s bets=%d %s\n", p.tag, p.rule, p.bets, msg)
		} else {
			fmt.Printf("OK   [%s] rule=%s bets=%d content=%q\n", p.tag, p.rule, p.bets, trunc(p.content, 60))
		}
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
