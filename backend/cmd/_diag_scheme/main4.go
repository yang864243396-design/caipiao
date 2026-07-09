package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  period:="1014016200388"
  rows,_:=pool.Query(ctx,`SELECT c.code, c.play_template, c.sale_status, c.guaji_ws_key,
    (SELECT count(*) FROM lottery_draws d WHERE d.lottery_code=c.code AND d.issue_no=$1) AS has_draw
    FROM lottery_catalog c WHERE c.guaji_ws_key='lottery_log101' ORDER BY c.code`, period)
  defer rows.Close()
  for rows.Next(){ var code,tpl,st,ws string; var n int; _=rows.Scan(&code,&tpl,&st,&ws,&n); fmt.Printf("%s tpl=%s sale=%s draw=%d\n", code,tpl,st,n) }
}
