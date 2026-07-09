package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  fmt.Println("payout ledgers with/without scheme match (current fuzzy join):")
  rows,_:=pool.Query(ctx,`
SELECT l.ledger_no, l.order_ref, l.delta_amount, l.created_at,
  sch.scheme_name AS fuzzy_name,
  c2.scheme_name AS order_ref_name
FROM wallet_ledger l
LEFT JOIN LATERAL (
    SELECT c.scheme_name FROM cloud_bet_records c
    WHERE c.member_id = l.member_id
      AND ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at))) <= 5
      AND ABS(c.amount::float8 - ABS(l.delta_amount::float8)) < 0.001
      AND c.guaji_account_id IS NOT DISTINCT FROM l.guaji_account_id
    ORDER BY ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at))) LIMIT 1
) sch ON true
LEFT JOIN cloud_bet_records c2 ON c2.bet_order_no = l.order_ref AND c2.member_id = l.member_id
WHERE l.txn_type = 'payout' AND l.delta_amount > 0
ORDER BY l.created_at DESC LIMIT 10`)
  defer rows.Close()
  for rows.Next(){
    var ln, ref, fuzzy, byRef string; var delta float64; var t any
    _=rows.Scan(&ln,&ref,&delta,&t,&fuzzy,&byRef)
    fmt.Printf("  %s ref=%s delta=%.2f fuzzy=%q byRef=%q\n", ln, ref, delta, fuzzy, byRef)
  }
}
