package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  rows,_:=pool.Query(ctx,`SELECT code, outbound_lottery_code, guaji_ws_key FROM lottery_catalog WHERE code IN ('tron_ffc_1m','hash_ffc_1m','tron_ffc_3m','hash_ffc_3m') ORDER BY code`)
  defer rows.Close()
  for rows.Next(){ var c,o,w string; _=rows.Scan(&c,&o,&w); fmt.Println(c,o,w) }
  var instSt string; var pending int
  _=pool.QueryRow(ctx,`SELECT status FROM scheme_instances WHERE id='inst-1-1782963020261'`).Scan(&instSt)
  _=pool.QueryRow(ctx,`SELECT count(*) FROM cloud_bet_records WHERE scheme_id='inst-1-1782963020261' AND status='pending' AND NULLIF(TRIM(third_party_bet_id),'') IS NOT NULL`).Scan(&pending)
  fmt.Printf("\ninst status=%s unsettled=%d\n", instSt, pending)
}
