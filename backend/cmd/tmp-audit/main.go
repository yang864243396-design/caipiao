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
  pool, _ := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
  defer pool.Close()
  ctx := context.Background()
  var memberID int64
  pool.QueryRow(ctx, `SELECT id FROM members WHERE account='vs8888'`).Scan(&memberID)
  fmt.Println("memberID", memberID)
  rows, _ := pool.Query(ctx, `SELECT run_mode, COUNT(*), COALESCE(SUM(amount::float8),0), COUNT(*) FILTER (WHERE guaji_account_id IS NOT NULL) FROM cloud_bet_records WHERE member_id=$1 GROUP BY run_mode`, memberID)
  for rows.Next() {
    var mode string; var cnt, withG int; var sum float64
    rows.Scan(&mode, &cnt, &sum, &withG)
    fmt.Printf("cloud_bet run_mode=%s count=%d sum=%.0f with_guaji=%d\n", mode, cnt, sum, withG)
  }
  rows.Close()
  var boTotal, boTP, boGuaji int
  pool.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE COALESCE(third_party_bet_id,'') != ''), COUNT(*) FILTER (WHERE guaji_account_id IS NOT NULL) FROM bet_orders WHERE member_id=$1`, memberID).Scan(&boTotal, &boTP, &boGuaji)
  fmt.Printf("bet_orders total=%d with_third_party_id=%d with_guaji=%d\n", boTotal, boTP, boGuaji)
  var wl, fr int
  pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallet_ledger WHERE member_id=$1`, memberID).Scan(&wl)
  pool.QueryRow(ctx, `SELECT COUNT(*) FROM fund_records WHERE member_id=$1`, memberID).Scan(&fr)
  fmt.Printf("wallet_ledger=%d fund_records=%d\n", wl, fr)
  rows2, _ := pool.Query(ctx, `SELECT id, run_mode, status, turnover::float8 FROM scheme_instances WHERE member_id=$1 ORDER BY updated_at DESC LIMIT 5`, memberID)
  for rows2.Next() {
    var id, mode, st string; var to float64
    rows2.Scan(&id, &mode, &st, &to)
    fmt.Printf("instance %s run_mode=%s status=%s turnover=%.0f\n", id, mode, st, to)
  }
  rows2, _ = pool.Query(ctx, `SELECT order_no, status, COALESCE(third_party_bet_id,''), amount::float8 FROM bet_orders WHERE member_id=$1 ORDER BY placed_at DESC LIMIT 5`, memberID)
  for rows2.Next() {
    var ono, st, tp string; var amt float64
    rows2.Scan(&ono, &st, &tp, &amt)
    fmt.Printf("bet_order %s status=%s tp=%s amt=%.0f\n", ono, st, tp, amt)
  }
}
