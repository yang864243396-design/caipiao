// 诊断：方案 pending 注单 vs 第三方结算状态
// go run ./cmd/diag-scheme-pending/ inst-1-1782018133383 [period]
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	schemeID := "inst-1-1782018133383"
	if len(os.Args) > 1 {
		schemeID = os.Args[1]
	}
	filterPeriod := ""
	if len(os.Args) > 2 {
		filterPeriod = os.Args[2]
	}

	ctx := context.Background()

	type cloudRow struct {
		PeriodNo         string  `json:"periodNo"`
		ThirdPartyPeriod string  `json:"thirdPartyPeriod"`
		Status           string  `json:"status"`
		ThirdPartyBetID  string  `json:"thirdPartyBetId"`
		BetOrderNo       string  `json:"betOrderNo"`
		Amount           float64 `json:"amount"`
		Pnl              float64 `json:"pnl"`
		PlacedAt         string  `json:"placedAt"`
	}
	rows, err := pool.Query(ctx, `
SELECT COALESCE(period_no,''), COALESCE(third_party_period,''), status,
       COALESCE(third_party_bet_id,''), COALESCE(bet_order_no,''),
       amount::float8, COALESCE(pnl,0)::float8, placed_at::text
FROM cloud_bet_records
WHERE scheme_id = $1
ORDER BY placed_at DESC
LIMIT 10`, schemeID)
	if err != nil {
		fmt.Println("cloud query:", err)
		os.Exit(1)
	}
	defer rows.Close()
	var cloud []cloudRow
	for rows.Next() {
		var r cloudRow
		if err := rows.Scan(&r.PeriodNo, &r.ThirdPartyPeriod, &r.Status, &r.ThirdPartyBetID, &r.BetOrderNo, &r.Amount, &r.Pnl, &r.PlacedAt); err != nil {
			fmt.Println("scan:", err)
			os.Exit(1)
		}
		if filterPeriod != "" && r.PeriodNo != filterPeriod && r.ThirdPartyPeriod != filterPeriod {
			continue
		}
		cloud = append(cloud, r)
	}

	type orderRow struct {
		OrderNo         string  `json:"orderNo"`
		IssueNo         string  `json:"issueNo"`
		Status          string  `json:"status"`
		ThirdPartyBetID string  `json:"thirdPartyBetId"`
		GuajiAccountID  int64   `json:"guajiAccountId"`
		Amount          float64 `json:"amount"`
		Pnl             float64 `json:"pnl"`
		PlacedAt        string  `json:"placedAt"`
	}
	orows, err := pool.Query(ctx, `
SELECT b.order_no, b.issue_no, b.status,
       COALESCE(b.third_party_bet_id,''), COALESCE(b.guaji_account_id,0),
       b.amount::float8, COALESCE(b.pnl,0)::float8, b.placed_at::text
FROM bet_orders b
JOIN cloud_bet_records c ON c.bet_order_no = b.order_no
WHERE c.scheme_id = $1
ORDER BY b.placed_at DESC
LIMIT 10`, schemeID)
	if err != nil {
		fmt.Println("orders query:", err)
		os.Exit(1)
	}
	defer orows.Close()
	var orders []orderRow
	for orows.Next() {
		var r orderRow
		if err := orows.Scan(&r.OrderNo, &r.IssueNo, &r.Status, &r.ThirdPartyBetID, &r.GuajiAccountID, &r.Amount, &r.Pnl, &r.PlacedAt); err != nil {
			fmt.Println("scan order:", err)
			os.Exit(1)
		}
		orders = append(orders, r)
	}

	out := map[string]any{"schemeId": schemeID, "cloudBetRecords": cloud, "betOrders": orders}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)

	// 对 pending 且有第三方注单号的，查第三方结算
	client := guaji.NewClient(cfg.Guaji)
	for _, c := range cloud {
		if c.Status != "pending" || c.ThirdPartyBetID == "" {
			continue
		}
		if filterPeriod != "" && c.PeriodNo != filterPeriod && c.ThirdPartyPeriod != filterPeriod {
			continue
		}
		credKey, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
		var tokenEnc string
		err := pool.QueryRow(ctx, `
SELECT ga.access_token_enc
FROM cloud_bet_records c
JOIN member_guaji_accounts ga ON ga.id = c.guaji_account_id
WHERE c.scheme_id = $1 AND c.third_party_bet_id = $2
LIMIT 1`, schemeID, c.ThirdPartyBetID).Scan(&tokenEnc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "token lookup %s: %v\n", c.ThirdPartyBetID, err)
			continue
		}
		token, err := guaji.DecryptSecret(credKey, tokenEnc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "decrypt: %v\n", err)
			continue
		}
		res, err := client.QuerySettlement(ctx, token, c.ThirdPartyBetID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "QuerySettlement %s: %v\n", c.ThirdPartyBetID, err)
			continue
		}
		b, _ := json.MarshalIndent(map[string]any{
			"thirdPartyBetId": c.ThirdPartyBetID,
			"periodNo":        c.PeriodNo,
			"thirdPartyPeriod": c.ThirdPartyPeriod,
			"upstream":        res,
		}, "", "  ")
		fmt.Fprintln(os.Stderr, string(b))
	}
	_ = strings.TrimSpace
}
