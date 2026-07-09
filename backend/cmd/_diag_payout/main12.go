package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  ref:="BO11782964637252"
  rows,_:=pool.Query(ctx,`SELECT ledger_no, txn_type, delta_amount::float8, order_ref, created_at FROM wallet_ledger WHERE order_ref=$1 ORDER BY created_at`, ref)
  defer rows.Close()
  for rows.Next(){ var ln,tt,oref string; var d float64; var t any; _=rows.Scan(&ln,&tt,&d,&oref,&t); fmt.Printf("%s %s delta=%.2f ref=%s at=%v\n", ln, tt, d, oref, t) }
}
