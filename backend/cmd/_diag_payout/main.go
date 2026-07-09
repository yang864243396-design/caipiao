package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  queries := []struct{ label, sql string }{
    {"pending guaji total", `SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND third_party_bet_id IS NOT NULL`},
    {"pending no tpid", `SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND (third_party_bet_id IS NULL OR TRIM(third_party_bet_id)='')`},
    {"pending all", `SELECT COUNT(*) FROM bet_orders WHERE status='pending'`},
    {"linked cloud", `SELECT COUNT(*) FROM bet_orders b JOIN cloud_bet_records c ON c.bet_order_no=b.order_no WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL`},
    {"linked running scheme", `SELECT COUNT(*) FROM bet_orders b JOIN cloud_bet_records c ON c.bet_order_no=b.order_no JOIN scheme_instances si ON si.id=c.scheme_id AND si.status='running' WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL`},
    {"linked non-running scheme", `SELECT COUNT(*) FROM bet_orders b JOIN cloud_bet_records c ON c.bet_order_no=b.order_no JOIN scheme_instances si ON si.id=c.scheme_id AND si.status<>'running' WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL`},
    {"no cloud link", `SELECT COUNT(*) FROM bet_orders b WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM cloud_bet_records c WHERE c.bet_order_no=b.order_no)`},
    {"older than 7d", `SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND placed_at < now()-interval '7 days'`},
    {"older than 30d", `SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND placed_at < now()-interval '30 days'`},
  }
  for _, q := range queries { var n int; _=pool.QueryRow(ctx, q.sql).Scan(&n); fmt.Printf("%-28s %d\n", q.label+":", n) }
  fmt.Println("\n--- by lottery TOP10 ---")
  rows,_:=pool.Query(ctx,`SELECT lottery_code, COUNT(*) n FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL GROUP BY lottery_code ORDER BY n DESC LIMIT 10`)
  defer rows.Close(); for rows.Next(){ var c string; var n int; _=rows.Scan(&c,&n); fmt.Printf("  %s: %d\n", c, n) }
  fmt.Println("\n--- by scheme status ---")
  rows2,_:=pool.Query(ctx,`SELECT COALESCE(si.status,'(none)'), COUNT(*) FROM bet_orders b LEFT JOIN cloud_bet_records c ON c.bet_order_no=b.order_no LEFT JOIN scheme_instances si ON si.id=c.scheme_id WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL GROUP BY si.status ORDER BY COUNT(*) DESC`)
  defer rows2.Close(); for rows2.Next(){ var st string; var n int; _=rows.Scan(&st,&n); fmt.Printf("  %s: %d\n", st, n) }
  fmt.Println("\n--- oldest 5 pending ---")
  rows3,_:=pool.Query(ctx,`SELECT order_no, lottery_code, issue_no, placed_at::date, third_party_bet_id FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL ORDER BY placed_at ASC LIMIT 5`)
  defer rows3.Close(); for rows3.Next(){ var on,lc,iss,tp string; var d any; _=rows3.Scan(&on,&lc,&iss,&d,&tp); fmt.Printf("  %s %s %s %v tpid=%s\n", on, lc, iss, d, tp) }
  var cp,cpRun,all,pend int
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records WHERE status='pending'`).Scan(&cp)
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records c JOIN scheme_instances si ON si.id=c.scheme_id AND si.status='running' WHERE c.status='pending'`).Scan(&cpRun)
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders WHERE guaji_account_id IS NOT NULL`).Scan(&all)
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders WHERE guaji_account_id IS NOT NULL AND status='pending'`).Scan(&pend)
  fmt.Printf("\ncloud pending=%d running=%d\nall guaji bets=%d pending=%d (%.1f%%)\n", cp, cpRun, all, pend, float64(pend)*100/float64(all))
}
