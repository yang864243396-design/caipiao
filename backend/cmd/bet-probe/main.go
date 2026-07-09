// 探测 guaji rule 参数；直接调 web_bets/lott，不经过 PlaceRealBet 重算。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ruleID := flag.String("rule", "3", "rule_id")
	content := flag.String("content", "6", "bet_content")
	betsNums := flag.Int("bets", 28, "bets_nums")
	solo := flag.Bool("solo", false, "solo")
	gameID := flag.Int("game", 27, "game_id")
	account := flag.String("account", "", "member account")
	autoType := flag.String("auto", "hash", "auto_type (hash|platform|probe)")
	amountUnit := flag.Float64("unit", 1, "amount_unit")
	multiple := flag.Int("mult", 2, "multiple")
	minBonus := flag.Float64("minbonus", 0, "min_single_bet_bonus (0=omit)")
	flag.Parse()

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	if *account == "" {
		*account = cfg.ClientDemoAccount
	}
	if *account == "" {
		*account = "vs8888"
	}

	guajiClient := guaji.NewClient(cfg.Guaji)
	key, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)

	var memberID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, *account).Scan(&memberID); err != nil {
		panic(err)
	}
	var tokenEnc string
	err = pool.QueryRow(ctx, `
SELECT access_token_enc FROM member_guaji_accounts
WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc)
	if err != nil {
		panic(err)
	}
	token, err := guaji.DecryptSecret(key, tokenEnc)
	if err != nil {
		panic(err)
	}

	unit := *amountUnit
	mult := *multiple
	amount := unit * float64(*betsNums) * float64(mult)
	contentItem := guaji.LottBetContent{
		RuleID:     strings.TrimSpace(*ruleID),
		BetContent: *content,
		AmountUnit: unit,
		BetsNums:   *betsNums,
		Multiple:   mult,
		BetAmount:  amount,
		Solo:       *solo,
	}
	if *minBonus > 0 {
		contentItem.MinSingleBetBonus = minBonus
		single := unit * float64(mult)
		contentItem.SingleBetAmount = &single
	}
	res, err := guajiClient.PlaceLottBet(ctx, token, guaji.LottBetRequest{
		AutoType:    *autoType,
		BetContents: []guaji.LottBetContent{contentItem},
		GameID:      *gameID,
		Currency:    3,
		BetMultiple: []guaji.LottBetMultipleOuter{},
	})
	if err != nil {
		fmt.Printf("FAIL rule=%s content=%q bets=%d solo=%v amount=%.0f err=%v\n",
			*ruleID, *content, *betsNums, *solo, amount, err)
		os.Exit(2)
	}
	fmt.Printf("OK id=%s periods=%s\n", res.ThirdPartyBetID, res.Periods)
	_ = pgx.ErrNoRows
}
