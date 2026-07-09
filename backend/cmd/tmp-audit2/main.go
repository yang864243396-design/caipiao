package main

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		fmt.Println("db:", err)
		return
	}
	defer pool.Close()
	ctx := context.Background()

	var outbound, ws string
	_ = pool.QueryRow(ctx, `SELECT COALESCE(outbound_lottery_code,''), COALESCE(guaji_ws_key,'') FROM lottery_catalog WHERE code='tron_ffc_1m'`).Scan(&outbound, &ws)
	fmt.Println("tron_ffc_1m outbound=", outbound, "ws=", ws)

	var user string
	var te *string
	_ = pool.QueryRow(ctx, `SELECT guaji_username, token_error FROM member_guaji_accounts WHERE member_id=1 AND is_active=true LIMIT 1`).Scan(&user, &te)
	fmt.Println("guaji user=", user, "token_error=", te)

	var wlGuaji, wlLocal int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE member_id=1 AND guaji_account_id IS NOT NULL`).Scan(&wlGuaji)
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE member_id=1 AND guaji_account_id IS NULL`).Scan(&wlLocal)
	fmt.Println("wallet_ledger guaji=", wlGuaji, "local=", wlLocal)

	rows, _ := pool.Query(ctx, `SELECT order_no, COALESCE(third_party_bet_id,''), status, amount::float8, guaji_account_id IS NOT NULL FROM bet_orders WHERE member_id=1 ORDER BY placed_at DESC LIMIT 8`)
	for rows.Next() {
		var o, t, s string
		var a float64
		var hasG bool
		_ = rows.Scan(&o, &t, &s, &a, &hasG)
		fmt.Printf("order %s tp=[%s] st=%s amt=%.0f guaji=%v\n", o, t, s, a, hasG)
	}
	rows.Close()

	fmt.Println("GUAJI_ENABLED", cfg.Guaji.Enabled)
	fmt.Println("GUAJI_HTTP_BASE", cfg.Guaji.HTTPBase)
	fmt.Println("CREDENTIALS_KEY set", cfg.Guaji.CredentialsKey != "")
}
