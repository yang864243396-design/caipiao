package main
import (
  "context"; "encoding/json"; "fmt"
  "github.com/joho/godotenv"
  "caipiao/backend/internal/config"
  "caipiao/backend/internal/db"
)
func main() {
  _ = godotenv.Load(".env")
  cfg := config.Load()
  pool,_ := db.Connect(context.Background(), cfg.DatabaseURL, 5, 1)
  defer pool.Close()
  ctx := context.Background()
  rows,_ := pool.Query(ctx, `
SELECT order_no, play_method, amount::float8, bet_payload::text, issue_no
FROM bet_orders WHERE member_id=1 AND created_at > now() - interval '2 hours'
AND (bet_payload::text LIKE '%1,2,3%' OR bet_payload::text LIKE '%123%')
ORDER BY created_at DESC LIMIT 5`)
  defer rows.Close()
  for rows.Next() {
    var ono, pm, iss, payload string; var amt float64
    _ = rows.Scan(&ono, &pm, &amt, &payload, &iss)
    fmt.Printf("order=%s issue=%s amt=%.2f method=%s\npayload=%s\n---\n", ono, iss, amt, pm, payload)
  }
}
