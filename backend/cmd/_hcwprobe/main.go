package main
import (
  "context"
  "fmt"
  "github.com/jackc/pgx/v5/pgxpool"
  "github.com/joho/godotenv"
  "caipiao/backend/internal/config"
  "caipiao/backend/internal/db/sqlcdb"
  "caipiao/backend/internal/schemes"
)
func main() {
  _ = godotenv.Load()
  cfg := config.Load()
  pool, _ := pgxpool.New(context.Background(), cfg.DatabaseURL)
  defer pool.Close()
  ctx := context.Background()
  defID := "def-1-1785294381718"
  var raw []byte
  var lottery, kind string
  _ = pool.QueryRow(ctx, `SELECT lottery_code, COALESCE(kind,'custom'), config FROM scheme_definitions WHERE id=$1`, defID).Scan(&lottery, &kind, &raw)
  rows, _ := pool.Query(ctx, `
SELECT c.period_no, c.bet_content FROM cloud_bet_records c
JOIN scheme_instances si ON si.id=c.scheme_id
WHERE si.definition_id=$1 ORDER BY c.placed_at DESC LIMIT 8`, defID)
  defer rows.Close()
  type bet struct{ p, c string }
  var bets []bet
  for rows.Next() {
    var b bet
    _ = rows.Scan(&b.p, &b.c)
    bets = append(bets, b)
  }
  load := func(period string, n int) [][]string {
    r, _ := pool.Query(ctx, `SELECT issue_no, balls FROM lottery_draws WHERE lottery_code=$1 ORDER BY drawn_at DESC, id DESC LIMIT $2`, lottery, n+8)
    defer r.Close()
    out := [][]string{}
    for r.Next() {
      var issue string
      var ballsRaw []byte
      _ = r.Scan(&issue, &ballsRaw)
      if issue == period || issue >= period { continue }
      balls := sqlcdb.ParseDrawBalls(ballsRaw)
      if len(balls)==0 { continue }
      out = append(out, balls)
      if len(out)>=n { break }
    }
    return out
  }
  norm := func(s string) string {
    // light normalize newlines
    return s
  }
  fmt.Println("lag check: 实下(N) vs 应下(N-1)")
  for i := 0; i+1 < len(bets); i++ {
    cur, prev := bets[i], bets[i+1]
    // expected for prev period
    expPrev, _, _ := schemes.RebuildHotColdPickContent(kind, raw, load(prev.p, 20))
    expCur, _, _ := schemes.RebuildHotColdPickContent(kind, raw, load(cur.p, 20))
    matchCur := norm(cur.c) == norm(expCur)
    matchLag := norm(cur.c) == norm(expPrev)
    fmt.Printf("N=%s matchCur=%v matchLag(应下N-1)=%v\n", cur.p, matchCur, matchLag)
    _ = prev
  }
}
