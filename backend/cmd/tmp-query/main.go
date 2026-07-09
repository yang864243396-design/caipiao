package main
import ("context"; "fmt"; "time"; "github.com/joho/godotenv"; "caipiao/backend/internal/config"; "caipiao/backend/internal/db")
func main() {
  _ = godotenv.Load()
  pool, _ := db.Connect(context.Background(), config.Load().DatabaseURL, 5, 1)
  defer pool.Close()
  ctx := context.Background()
  rows, _ := pool.Query(ctx, `SELECT issue_no, drawn_at FROM lottery_draws WHERE lottery_code='tron_ffc_1m' ORDER BY drawn_at DESC LIMIT 5`)
  defer rows.Close()
  for rows.Next() {
    var issue string; var at time.Time
    _ = rows.Scan(&issue, &at)
    fmt.Printf("issue=%s at=%s ago=%s\n", issue, at.UTC().Format(time.RFC3339), time.Since(at.UTC()).Round(time.Second))
  }
}
