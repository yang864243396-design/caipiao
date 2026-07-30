package main
import (
  "context"; "fmt"
  "caipiao/backend/internal/config"
  "github.com/jackc/pgx/v5/pgxpool"
)
func main() {
  cfg := config.Load()
  pool,_ := pgxpool.New(context.Background(), cfg.DatabaseURL)
  defer pool.Close()
  // draw for period
  var balls string
  err := pool.QueryRow(context.Background(), `
SELECT balls FROM lottery_draws WHERE lottery_code='tron_ffc_1m' AND issue_no='1014150300106'`).Scan(&balls)
  fmt.Println("draw", balls, err)
  // also check ledger for this order
  rows,_ := pool.Query(context.Background(), `
SELECT kind, amount::text, balance_after::text, created_at
FROM member_ledger_entries
WHERE ref_no LIKE '%11785377023282266000369839%' OR ref_no LIKE '%CB17853770221284370000015%'
ORDER BY created_at`)
  for rows.Next() {
    var k,a,b string; var t interface{}
    rows.Scan(&k,&a,&b,&t)
    fmt.Println("ledger", k, a, b, t)
  }
}
