package main
import ("context";"encoding/json";"fmt";"time";"github.com/jackc/pgx/v5/pgtype";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db";"caipiao/backend/internal/db/sqlcdb";"caipiao/backend/internal/guaji")
func main(){
  _=godotenv.Load(); cfg:=config.Load(); ctx:=context.Background(); pool,_:=db.Connect(ctx,cfg.DatabaseURL,5,1); defer pool.Close()
  client:=guaji.NewClient(cfg.Guaji); q:=sqlcdb.New(pool)
  path:="lottery_logs"; code:="tron_ffc_1m"; tpl:="ssc_std"
  logs,_:=client.FetchHistoryDrawLogs(ctx,path,1,30)
  inserted:=0
  for _, row := range logs {
    balls:=row.Balls.BallsFor(tpl)
    if len(balls)==0 { continue }
    bj,_:=json.Marshal(balls)
    _, err := q.InsertLotteryDraw(ctx, sqlcdb.InsertLotteryDrawParams{LotteryCode:code, IssueNo:row.Periods, PeriodShort:row.Periods[len(row.Periods)-6:], Balls:bj, SumValue:int32(guaji.SumBalls(balls)), DrawnAt:pgtype.Timestamptz{Time:row.DrawnAt, Valid:true}})
    if err==nil { inserted++ }
  }
  var n int; _=pool.QueryRow(ctx,`SELECT count(*) FROM lottery_draws WHERE lottery_code='tron_ffc_1m' AND issue_no='1014016200388'`).Scan(&n)
  fmt.Printf("backfill inserted=%d tron_ffc_1m draw_1014016200388=%d\n", inserted, n)
  _=time.Now()
}
