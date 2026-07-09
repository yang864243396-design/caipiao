package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db";"caipiao/backend/internal/db/sqlcdb";"caipiao/backend/internal/schemes")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background(); q:=sqlcdb.New(pool)
  var lottery, issue, play string; var payload []byte; var amount float64
  _=pool.QueryRow(ctx,`SELECT lottery_code, issue_no, play_method, bet_payload, amount::float8 FROM bet_orders WHERE order_no='BO11782905494687'`).Scan(&lottery,&issue,&play,&payload,&amount)
  draw, err := q.GetLotteryDrawByIssue(ctx, sqlcdb.GetLotteryDrawByIssueParams{LotteryCode: lottery, IssueNo: issue})
  if err != nil { panic(err) }
  balls := sqlcdb.ParseDrawBalls(draw.Balls)
  p := schemes.EnsureBetPayload(payload, play, "BO11782905494687")
  hit, odds := schemes.EvaluateBetPayload(p, balls)
  pnl := schemes.CalcOrderPnL(amount, hit, odds)
  fmt.Printf("lottery=%s issue=%s balls=%v hit=%v odds=%v pnl=%.2f\n", lottery, issue, balls, hit, odds, pnl)
}
