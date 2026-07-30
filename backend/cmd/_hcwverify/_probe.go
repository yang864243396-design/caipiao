package main
import (
  "context"
  "fmt"
  "os"
  "github.com/jackc/pgx/v5/pgxpool"
  "github.com/joho/godotenv"
  "caipiao/backend/internal/config"
)
func main() {
  _ = godotenv.Load()
  cfg := config.Load()
  pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
  if err != nil { panic(err) }
  defer pool.Close()
  ctx := context.Background()
  var n int
  _ = pool.QueryRow(ctx, `SELECT count(*) FROM lottery_draws WHERE lottery_code=$1`, "tron_ffc_1m").Scan(&n)
  fmt.Println("draws count tron_ffc_1m:", n)
  rows, err := pool.Query(ctx, `SELECT issue_no, balls::text FROM lottery_draws WHERE lottery_code=$1 ORDER BY issue_no DESC LIMIT 5`, "tron_ffc_1m")
  if err != nil { panic(err) }
  defer rows.Close()
  for rows.Next() {
    var issue, balls string
    _ = rows.Scan(&issue, &balls)
    fmt.Println(issue, balls)
  }
  // compare period string
  cur := "1014146800040"
  var cnt int
  _ = pool.QueryRow(ctx, `SELECT count(*) FROM lottery_draws WHERE lottery_code=$1 AND issue_no < $2`, "tron_ffc_1m", cur).Scan(&cnt)
  fmt.Println("issue_no < period:", cnt)
  _ = pool.QueryRow(ctx, `SELECT count(*) FROM lottery_draws WHERE lottery_code=$1 AND issue_no::text < $2`, "tron_ffc_1m", cur).Scan(&cnt)
  fmt.Println("issue text < period:", cnt)
  // sample distinct lottery codes with recent draws
  rows2, _ := pool.Query(ctx, `SELECT lottery_code, count(*) FROM lottery_draws GROUP BY 1 ORDER BY 2 DESC LIMIT 15`)
  defer rows2.Close()
  for rows2.Next() {
    var code string
    var c int
    _ = rows2.Scan(&code, &c)
    fmt.Println("code", code, c)
  }
}
