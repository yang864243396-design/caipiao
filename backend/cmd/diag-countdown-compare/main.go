package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/guaji/periodsync"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/schemes"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	q := sqlcdb.New(pool)

	instID := "inst-1-1782023721006"
	row, err := q.GetSchemeInstanceFull(ctx, instID)
	if err != nil {
		panic(err)
	}

	guajiCfg := guaji.LoadConfigFromEnv()
	client := guaji.NewClient(guajiCfg)
	appCfg := config.Load()
	accounts := accountsvc.NewService(pool, client, appCfg.Guaji.CredentialsKey, appCfg.JWTSecret)
	syncer := periodsync.NewSyncer(pool, client, accounts)
	svc := schemes.NewService(pool, syncer)

	now := time.Now()
	if syncer != nil {
		_ = syncer.ForceRefresh(ctx, row.LotteryCode)
	}

	memberAccount, _ := q.GetMemberAccountByID(ctx, row.MemberID)
	list, err := svc.ListInstancesQuery(ctx, memberAccount, schemes.InstanceListQuery{IDs: []string{instID}})
	if err != nil {
		panic(err)
	}
	var apiItem schemes.Instance
	if len(list.Items) > 0 {
		apiItem = list.Items[0]
	}

	fmt.Println("=== 本平台 API 倒计时字段 ===")
	b, _ := json.MarshalIndent(map[string]any{
		"lotteryCode":        apiItem.LotteryCode,
		"status":             apiItem.Status,
		"statusReason":       apiItem.StatusReason,
		"countdownSec":       apiItem.CountdownSec,
		"countdownCloseAt":   apiItem.CountdownCloseAt,
		"countdownEndTime":   apiItem.CountdownEndTime,
		"countdownPeriod":    apiItem.CountdownPeriod,
		"countdownWindowSec": apiItem.CountdownWindowSec,
		"countdownLabel":     apiItem.CountdownLabel,
	}, "", "  ")
	fmt.Println(string(b))

	fmt.Println("\n=== 第三方 periods (tron_ffc_1m game_id=27) ===")
	if !guajiCfg.Enabled {
		fmt.Println("GUAJI_ENABLED=false, skip upstream")
		return
	}
	token, err := accounts.SyncAccessToken(ctx)
	if err != nil {
		panic(err)
	}
	periods, raw, err := client.FetchLottPeriods(ctx, token, 27, 5)
	if err != nil {
		panic(err)
	}
	fmt.Println("raw:", string(raw))
	open, endAt, ok := guaji.PickOpenLottPeriod(periods, row.LotteryCode, now)
	if !ok {
		fmt.Println("no open period")
		return
	}
	remUpstream := int(endAt.Sub(now.UTC()).Round(time.Second).Seconds())
	if remUpstream < 0 {
		remUpstream = 0
	}
	// 第三方展示封顶为单期窗口（1 分彩 = 60s）
	remDisplay := remUpstream
	if d := int(endAt.Sub(openStartAtForPeriod(open, row.LotteryCode)).Round(time.Second).Seconds()); d > 0 && remDisplay > d {
		remDisplay = d
	}
	fmt.Printf("open period=%s start=%s end=%s endAtUTC=%s\n", open.Period, open.StartTime, open.EndTime, endAt.UTC().Format(time.RFC3339))
	fmt.Printf("upstream raw remaining sec: %d\n", remUpstream)
	fmt.Printf("upstream display sec (capped): %d\n", remDisplay)

	if ps, ok := lottery.PeriodsScheduleFor(row.LotteryCode); ok {
		fmt.Printf("\ncache: period=%s closeAt=%s rawEnd=%s duration=%d updatedAt=%s\n",
			ps.CurrentPeriod, ps.CloseAt.UTC().Format(time.RFC3339), ps.CloseEndTimeRaw, ps.PeriodDurationSec, ps.UpdatedAt.UTC().Format(time.RFC3339))
		if sec, ok := lottery.PeriodsDisplayCountdownSec(row.LotteryCode, now); ok {
			fmt.Printf("PeriodsDisplayCountdownSec: %d\n", sec)
		}
	}

	fmt.Printf("\n=== 差值 ===\napi countdownSec=%d vs upstream display=%d diff=%d\n",
		apiItem.CountdownSec, remDisplay, apiItem.CountdownSec-remDisplay)
}

func openStartAtForPeriod(p guaji.LottPeriod, lotteryCode string) time.Time {
	t, err := guaji.ParseGuajiPeriodTimeForLottery(lotteryCode, p.StartTime)
	if err != nil {
		return time.Time{}
	}
	return t
}
