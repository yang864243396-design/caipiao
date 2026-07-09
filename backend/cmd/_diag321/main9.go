package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  var pos int
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND placed_at < (SELECT placed_at FROM bet_orders WHERE order_no='BO11782905494687')`).Scan(&pos)
  fmt.Println("orders ahead in payout queue", pos)
  var total int; _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL`).Scan(&total)
  fmt.Println("total pending guaji orders", total)
  // draw exists check
  var balls string
  err:=pool.QueryRow(ctx,`SELECT balls::text FROM lottery_draws WHERE lottery_code='bnb_ffc_1m' AND issue_no='10111202607011172'`).Scan(&balls)
  fmt.Println("local draw", balls, "err", err)
}
