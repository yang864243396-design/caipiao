package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: e2e-check <orderNo>")
		os.Exit(1)
	}
	_ = godotenv.Load()
	cfg := config.Load()
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db err:", err)
		os.Exit(1)
	}
	defer pool.Close()
	var tpID, status, currency string
	var amount float64
	err = pool.QueryRow(context.Background(), `
		SELECT COALESCE(third_party_bet_id,''), status, COALESCE(currency,''), amount::float8
		FROM bet_orders WHERE order_no = $1`, os.Args[1]).Scan(&tpID, &status, &currency, &amount)
	if err != nil {
		fmt.Println("query err:", err)
		os.Exit(1)
	}
	fmt.Printf("order=%s third_party_bet_id=%s status=%s currency=%s amount=%.0f\n", os.Args[1], tpID, status, currency, amount)
}
