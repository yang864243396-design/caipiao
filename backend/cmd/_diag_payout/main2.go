package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()

  fmt.Println("--- by member TOP5 ---")
  rows,_:=pool.Query(ctx,`SELECT m.account, COUNT(*) n FROM bet_orders b JOIN members m ON m.id=b.member_id WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL GROUP BY m.account ORDER BY n DESC LIMIT 5`)
  defer rows.Close(); for rows.Next(){ var a string; var n int; _=rows.Scan(&a,&n); fmt.Printf("  %s: %d\n", a, n) }

  fmt.Println("\n--- by placed_at date (last 14d) ---")
  rows2,_:=pool.Query(ctx,`SELECT placed_at::date d, COUNT(*) n FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL GROUP BY placed_at::date ORDER BY d DESC LIMIT 14`)
  defer rows2.Close(); for rows2.Next(){ var d any; var n int; _=rows.Scan(&d,&n); fmt.Printf("  %v: %d\n", d, n) }

  fmt.Println("\n--- has third_party_bet_id ---")
  var withTP, withoutTP int
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND third_party_bet_id IS NOT NULL AND TRIM(third_party_bet_id)<>''`).Scan(&withTP)
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND (third_party_bet_id IS NULL OR TRIM(third_party_bet_id)='')`).Scan(&withoutTP)
  fmt.Printf("  with tpid: %d\n  without tpid: %d\n", withTP, withoutTP)

  fmt.Println("\n--- sample with tpid (5 oldest) ---")
  rows3,_:=pool.Query(ctx,`SELECT order_no, lottery_code, issue_no, placed_at::date, third_party_bet_id FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND third_party_bet_id IS NOT NULL AND TRIM(third_party_bet_id)<>'' ORDER BY placed_at ASC LIMIT 5`)
  defer rows3.Close(); for rows3.Next(){ var on,lc,iss,tp string; var d any; _=rows.Scan(&on,&lc,&iss,&d,&tp); fmt.Printf("  %s %s %s %v tpid=%s\n", on, lc, iss, d, tp) }

  fmt.Println("\n--- scheme cloud bets total ---")
  var cloudTotal, cloudGuaji int
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records`).Scan(&cloudTotal)
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records WHERE bet_order_no IS NOT NULL AND TRIM(bet_order_no)<>''`).Scan(&cloudGuaji)
  fmt.Printf("  cloud records: %d, with bet_order_no: %d\n", cloudTotal, cloudGuaji)

  fmt.Println("\n--- running schemes bet count ---")
  rows4,_:=pool.Query(ctx,`SELECT si.id, COUNT(c.id) n FROM scheme_instances si LEFT JOIN cloud_bet_records c ON c.scheme_id=si.id WHERE si.status='running' GROUP BY si.id ORDER BY n DESC LIMIT 10`)
  defer rows4.Close(); for rows4.Next(){ var id string; var n int; _=rows.Scan(&id,&n); fmt.Printf("  %s: %d cloud bets\n", id, n) }
}
