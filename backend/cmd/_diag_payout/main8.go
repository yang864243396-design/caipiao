package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  fmt.Println("cloud guaji_account_id=4 by status:")
  rows,_:=pool.Query(ctx,`SELECT status, COUNT(*) FROM cloud_bet_records WHERE guaji_account_id=4 GROUP BY status ORDER BY COUNT(*) DESC`)
  defer rows.Close(); for rows.Next(){ var s string; var n int; _=rows.Scan(&s,&n); fmt.Printf("  %s: %d\n", s, n) }

  fmt.Println("\ncloud guaji=4 linked to scheme_instances:")
  var linked int; _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records c WHERE c.guaji_account_id=4 AND EXISTS (SELECT 1 FROM scheme_instances si WHERE si.id=c.scheme_id)`).Scan(&linked)
  fmt.Println(" ", linked)

  fmt.Println("\n8 linked orphan pending cloud detail:")
  rows2,_:=pool.Query(ctx,`SELECT c.record_no, c.scheme_id, c.status, c.bet_order_no FROM cloud_bet_records c JOIN bet_orders b ON b.order_no=c.bet_order_no WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM member_guaji_accounts ga WHERE ga.id=b.guaji_account_id) LIMIT 10`)
  defer rows2.Close(); for rows2.Next(){ var rn,sid,st,bo string; _=rows2.Scan(&rn,&sid,&st,&bo); fmt.Printf("  %s scheme=%s status=%s order=%s\n", rn,sid,st,bo) }
}
