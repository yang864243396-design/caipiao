package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  fmt.Println("member_guaji_accounts:")
  rows,_:=pool.Query(ctx,`SELECT id, member_id, is_active, created_at::date FROM member_guaji_accounts ORDER BY id`)
  defer rows.Close(); for rows.Next(){ var id,mid int64; var act bool; var d any; _=rows.Scan(&id,&mid,&act,&d); fmt.Printf("  id=%d member=%d active=%v created=%v\n", id,mid,act,d) }

  fmt.Println("\nscheme-related pending:")
  var n int; _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders b JOIN cloud_bet_records c ON c.bet_order_no=b.order_no WHERE b.status='pending'`).Scan(&n)
  fmt.Println("  bet_orders pending with cloud:", n)

  fmt.Println("\nreal-bet-matrix fingerprint (Jun 25-29 bulk):")
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id=4 AND placed_at BETWEEN '2026-06-25' AND '2026-06-30'`).Scan(&n)
  fmt.Println("  account=4 Jun25-30:", n)
}
