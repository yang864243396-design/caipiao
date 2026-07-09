package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	id := "inst-1-1782023721006"
	ctx := context.Background()

	type inst struct {
		Status, Reason, DefID, Lottery string
		Turnover, Pnl, LookbackPnl, SessionPnl, Mult float64
		RoundIndex, RunTimeSec         int32
		RunningSince, UpdatedAt        *string
		SimBet                         bool
	}
	var i inst
	err = pool.QueryRow(ctx, `
SELECT status, COALESCE(status_reason,''), definition_id, lottery_code,
       turnover::float8, pnl::float8, lookback_pnl::float8, session_pnl::float8, multiplier::float8,
       round_index, run_time_sec, running_since::text, updated_at::text, sim_bet
FROM scheme_instances WHERE id=$1`, id).Scan(
		&i.Status, &i.Reason, &i.DefID, &i.Lottery,
		&i.Turnover, &i.Pnl, &i.LookbackPnl, &i.SessionPnl, &i.Mult,
		&i.RoundIndex, &i.RunTimeSec, &i.RunningSince, &i.UpdatedAt, &i.SimBet,
	)
	if err != nil {
		panic(err)
	}
	b, _ := json.MarshalIndent(i, "", "  ")
	fmt.Println("instance:", string(b))

	var defCfg []byte
	_ = pool.QueryRow(ctx, `SELECT config FROM scheme_definitions WHERE id=$1`, i.DefID).Scan(&defCfg)
	fmt.Println("definition_config:", string(defCfg))

	rows, _ := pool.Query(ctx, `
SELECT period_no, amount::float8, pnl::float8, status, multiplier, round_label,
       bet_content, placed_at::text, third_party_bet_id
FROM cloud_bet_records WHERE scheme_id=$1 ORDER BY placed_at ASC`, id)
	defer rows.Close()
	type rec struct {
		Period, Status, Mult, Round, Content, At, TpID string
		Amount, Pnl                                    float64
	}
	var recs []rec
	for rows.Next() {
		var r rec
		_ = rows.Scan(&r.Period, &r.Amount, &r.Pnl, &r.Status, &r.Mult, &r.Round, &r.Content, &r.At, &r.TpID)
		recs = append(recs, r)
	}
	b2, _ := json.MarshalIndent(recs, "", "  ")
	fmt.Println("cloud_bets:", string(b2))

	type lbSettings struct {
		Judgment                string  `json:"judgment"`
		ApplyFormal, ApplySim   bool    `json:"applyFormal"`
		SP, SL, OP, OL          float64 `json:"sp"`
		WMin, WMax              float64 `json:"wMin"`
	}
	var j lbSettings
	_ = pool.QueryRow(ctx, `
SELECT judgment, apply_formal, apply_sim,
       COALESCE(single_profit_threshold,0)::float8,
       COALESCE(single_loss_threshold,0)::float8,
       COALESCE(overall_profit_threshold,0)::float8,
       COALESCE(overall_loss_threshold,0)::float8,
       COALESCE(scheme_wins_min,0)::float8,
       COALESCE(scheme_wins_max,0)::float8
FROM member_lookback_settings m
JOIN scheme_instances si ON si.member_id = m.member_id
WHERE si.id=$1`, id).Scan(&j.Judgment, &j.ApplyFormal, &j.ApplySim, &j.SP, &j.SL, &j.OP, &j.OL, &j.WMin, &j.WMax)
	b3, _ := json.MarshalIndent(j, "", "  ")
	fmt.Println("lookback_settings:", string(b3))

	lrows, _ := pool.Query(ctx, `
SELECT m.sim_bet, m.session_pnl::float8, COALESCE(m.period_issue,''), m.period_pnl::float8,
       m.period_hit_count, m.total_hit_count, m.updated_at::text
FROM member_lookback_runtime m
JOIN scheme_instances si ON si.member_id = m.member_id AND m.sim_bet = si.sim_bet
WHERE si.id=$1`, id)
	if lrows != nil {
		defer lrows.Close()
		for lrows.Next() {
			var sim bool
			var sp, pp float64
			var issue string
			var ph, th int32
			var at string
			_ = lrows.Scan(&sim, &sp, &issue, &pp, &ph, &th, &at)
			fmt.Printf("lookback_runtime: simBet=%v sessionPnl=%.2f period=%s periodPnl=%.2f periodHits=%d totalHits=%d at=%s\n",
				sim, sp, issue, pp, ph, th, at)
		}
	}

	// global cloud settings
	var stopLoss, takeProfit *float64
	_ = pool.QueryRow(ctx, `
SELECT total_stop_loss::float8, total_take_profit::float8
FROM cloud_global_settings c
JOIN scheme_instances si ON si.member_id = c.member_id
WHERE si.id=$1`, id).Scan(&stopLoss, &takeProfit)
	fmt.Printf("cloud_global: stopLoss=%v takeProfit=%v\n", stopLoss, takeProfit)
}
