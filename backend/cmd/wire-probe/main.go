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

type probe struct {
	game    int
	rule    string
	content string
	bets    int
	solo    bool
	tag     string
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

	account := "vs8888"
	key, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	var memberID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, account).Scan(&memberID); err != nil {
		panic(err)
	}
	var tokenEnc string
	if err := pool.QueryRow(ctx, `
SELECT access_token_enc FROM member_guaji_accounts
WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc); err != nil {
		panic(err)
	}
	token, err := guaji.DecryptSecret(key, tokenEnc)
	if err != nil {
		panic(err)
	}
	client := guaji.NewClient(cfg.Guaji)

	g41 := 41
	g50 := 50
	g54 := 54
	g65 := 65

	probes := []probe{
		// g011 任二直选单式 75
		{g41, "75", "1,2,,,", 1, false, "ren2-ds-wire"},
		{g41, "75", "1,,,,2", 1, false, "ren2-ds-pos2"},
		{g41, "75", "12,34", 2, false, "ren2-ds-pairs"},
		{g41, "75", "12,34,56,78,90", 5, false, "ren2-ds-5pairs"},
		{g41, "75", "12,34,56,78,90", 10, false, "ren2-ds-10bets"},
		{g41, "75", "1,2,3,4,5,6,7,8,9,0", 10, false, "ren2-ds-pool10"},
		{g41, "75", "12", 1, true, "ren2-ds-solo"},
		// g011 任二直选和值 76
		{g41, "76", "3", 1, false, "ren2-hz-3"},
		{g41, "76", "6", 1, false, "ren2-hz-6"},
		{g41, "76", "3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18", 16, false, "ren2-hz-all"},
		{g41, "76", "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18", 18, false, "ren2-hz-1-18"},
		{g41, "76", "6", 1, true, "ren2-hz-solo"},
		// g011 任二组选 77-79
		{g41, "77", "1,2", 1, false, "ren2-zx-fs"},
		{g41, "77", "1,2,3", 3, false, "ren2-zx-fs-pool3"},
		{g41, "77", "1,2,,,", 1, false, "ren2-zx-wire"},
		{g41, "78", "12", 1, false, "ren2-zx-ds"},
		{g41, "78", "12,34", 2, false, "ren2-zx-ds-pairs"},
		{g41, "79", "3", 1, false, "ren2-zx-hz-3"},
		{g41, "79", "6", 1, false, "ren2-zx-hz-6"},
		// g011 任三 83-88
		{g41, "83", "1,1,2,,,", 1, true, "ren3-zu3-fs-solo"},
		{g41, "83", "11,22", 2, false, "ren3-zu3-fs-pairs"},
		{g41, "83", "1,1,2", 1, false, "ren3-zu3-flat"},
		{g41, "84", "112", 1, true, "ren3-zu3-ds-solo"},
		{g41, "84", "121", 1, true, "ren3-zu3-ds-121"},
		{g41, "84", "11,2", 1, false, "ren3-zu3-ds-11-2"},
		{g41, "85", "1,2,3,,", 1, true, "ren3-zu6-fs-solo"},
		{g41, "85", "1,2,3", 1, false, "ren3-zu6-flat"},
		{g41, "86", "123", 1, true, "ren3-zu6-ds-solo"},
		{g41, "87", "112", 1, true, "ren3-hunhe-solo"},
		{g41, "87", "112,121", 2, false, "ren3-hunhe-2"},
		{g41, "88", "6", 1, true, "ren3-zx-hz-solo"},
		{g41, "88", "6", 1, false, "ren3-zx-hz"},
		// g011 任四 142-145
		{g41, "142", "1,2,3,4", 1, true, "ren4-ds-solo"},
		{g41, "142", "1234", 1, true, "ren4-ds-1234-solo"},
		{g41, "142", "1,2,3,4,", 1, true, "ren4-ds-wire-solo"},
		{g41, "143", "1,2,3,4", 1, false, "ren4-zu24"},
		{g41, "144", "12,34", 2, false, "ren4-zu12"},
		{g41, "145", "1,2,3", 1, false, "ren4-zu6"},
		// g015 组选60/30/20/10/5
		{g41, "157", "1,2,3,4,5", 1, false, "zu60-12345"},
		{g41, "157", "1,2,3,4,5,6", 6, false, "zu60-pool6"},
		{g41, "158", "1,2,3,4,5", 1, false, "zu30-12345"},
		{g41, "159", "1,2,3,4,5", 1, false, "zu20-12345"},
		{g41, "160", "1,2,3,4,5", 1, false, "zu10-12345"},
		{g41, "161", "1,2,3,4,5", 1, false, "zu5-12345"},
		// g016 位置大小单双
		{g41, "261", ",,,大,大", 1, false, "hou2-dxds-cn"},
		{g41, "261", ",,,小,小", 1, false, "hou2-dxds-xiao"},
		{g41, "261", ",,,单,单", 1, false, "hou2-dxds-dan"},
		{g41, "261", "大,大,,,", 1, false, "hou2-dxds-front"},
		{g41, "261", "3,3,,,", 1, false, "hou2-dxds-33"},
		{g41, "261", "2,3,,,", 1, false, "hou2-dxds-23"},
		{g41, "262", ",,,,大", 1, false, "hou2-daxiao"},
		{g41, "265", ",,大,大,大", 1, false, "qian3-dxds"},
		{g41, "266", ",,,,小", 1, false, "hou2-danshuang"},
		// SYXW 50
		{g50, "166", "01,02,03", 1, true, "syxw-q3-fs-solo"},
		{g50, "166", "01,02,03", 1, false, "syxw-q3-fs"},
		{g50, "168", "01,02", 1, true, "syxw-zx-fs-solo"},
		{g50, "170", "01,02", 1, true, "syxw-q2-fs-solo"},
		// PK10 54
		{g54, "193", "01,02,,,,,,,,", 1, true, "pk10-q2-fs-solo"},
		{g54, "193", "0102,,,,,,,,,", 1, true, "pk10-q2-fs-compact"},
		{g54, "193", "01,02", 1, false, "pk10-q2-fs-flat"},
		// K3 65 bnb_k3_1m
		{g65, "224", "6", 1, false, "k3-hezhi-6"},
		{g65, "225", "11", 1, false, "k3-ertong-11"},
		{g65, "225", "22", 1, false, "k3-ertong-22"},
		{g65, "225", "1,1", 1, false, "k3-ertong-1-1"},
		{g65, "225", "112", 1, false, "k3-ertong-112"},
		{g65, "225", "11,22", 2, false, "k3-ertong-2pair"},
		{g65, "226", "11", 1, false, "k3-ertong-fu-11"},
		{g65, "226", "11,22,33", 3, false, "k3-ertong-fu-pool"},
		{g65, "229", "1,2", 1, false, "k3-biaozhun-12"},
		{g65, "229", "1,2,3", 3, false, "k3-biaozhun-pool3"},
		{g65, "229", "12", 1, false, "k3-biaozhun-12compact"},
	}

	unit := 2.0
	var okN, failN int
	for _, p := range probes {
		amount := unit * float64(p.bets)
		_, err := client.PlaceLottBet(ctx, token, guaji.LottBetRequest{
			AutoType: "probe",
			BetContents: []guaji.LottBetContent{{
				RuleID:     p.rule,
				BetContent: p.content,
				AmountUnit: unit,
				BetsNums:   p.bets,
				Multiple:   1,
				BetAmount:  amount,
				Solo:       p.solo,
			}},
			GameID:      p.game,
			Currency:    3,
			BetMultiple: []guaji.LottBetMultipleOuter{},
		})
		status := "OK"
		detail := ""
		if err != nil {
			status = "FAIL"
			failN++
			detail = err.Error()
			if i := strings.Index(detail, "40000:"); i >= 0 {
				detail = strings.TrimSpace(detail[i+6:])
			} else if i := strings.Index(detail, "40055:"); i >= 0 {
				detail = "封盘"
			}
		} else {
			okN++
		}
		fmt.Printf("%s game=%d rule=%s tag=%s content=%q bets=%d solo=%v %s\n",
			status, p.game, p.rule, p.tag, p.content, p.bets, p.solo, detail)
	}
	fmt.Printf("\n--- summary ok=%d fail=%d ---\n", okN, failN)
}
