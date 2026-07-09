package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  ref:="BO11782964637252"
  var st, lc, iss string; var amt float64
  err:=pool.QueryRow(ctx,`SELECT status, amount::float8, lottery_code, issue_no FROM bet_orders WHERE order_no=$1`, ref).Scan(&st,&amt,&lc,&iss)
  fmt.Printf("bet %s err=%v status=%s amt=%.2f %s %s\n", ref, err, st, amt, lc, iss)
  rows2,_:=pool.Query(ctx,`SELECT record_no, scheme_name, status, amount::float8, COALESCE(bet_order_no,'') FROM cloud_bet_records WHERE bet_order_no=$1`, ref)
  defer rows2.Close()
  for rows2.Next(){ var rn,sn,st,bo string; var a float64; _=rows2.Scan(&rn,&sn,&st,&a,&bo); fmt.Printf("  cloud %s scheme=%s status=%s amt=%.2f\n", rn,sn,st,a) }
  rows3,_:=pool.Query(ctx,`
SELECT l.ledger_no, l.order_ref, sch.scheme_name fuzzy, c2.scheme_name by_ref
FROM wallet_ledger l
LEFT JOIN LATERAL (
    SELECT c.scheme_name FROM cloud_bet_records c
    WHERE c.member_id = l.member_id
      AND ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at))) <= 5
      AND ABS(c.amount::float8 - ABS(l.delta_amount::float8)) < 0.001
      AND c.guaji_account_id IS NOT DISTINCT FROM l.guaji_account_id
    ORDER BY ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at))) LIMIT 1
) sch ON true
LEFT JOIN cloud_bet_records c2 ON c2.bet_order_no = l.order_ref
WHERE l.txn_type='bet_debit' ORDER BY l.created_at DESC LIMIT 5`)
  defer rows3.Close()
  fmt.Println("bet_debit samples:")
  for rows3.Next(){ var ln,ref,fuzzy,byRef string; _=rows3.Scan(&ln,&ref,&fuzzy,&byRef); fmt.Printf("  %s fuzzy=%q byRef=%q\n", ref, fuzzy, byRef) }
}
