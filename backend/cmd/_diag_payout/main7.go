package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  var n1,n2,n3 int
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records WHERE guaji_account_id=4`).Scan(&n1)
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records c JOIN bet_orders b ON b.order_no=c.bet_order_no WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM member_guaji_accounts ga WHERE ga.id=b.guaji_account_id)`).Scan(&n2)
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM wallet_ledger wl JOIN bet_orders b ON wl.ref_no=b.order_no WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM member_guaji_accounts ga WHERE ga.id=b.guaji_account_id)`).Scan(&n3)
  fmt.Printf("cloud guaji_account_id=4: %d\ncloud linked orphan pending: %d\nwallet linked orphan pending: %d\n", n1,n2,n3)
}
