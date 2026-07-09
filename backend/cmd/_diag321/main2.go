package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  fmt.Println("=== search 321 in definitions/instances ===")
  rows,_:=pool.Query(ctx,`SELECT si.id, COALESCE(sd.name,''), si.status, si.lottery_code, COALESCE(si.current_period,''), si.updated_at::text FROM scheme_instances si LEFT JOIN scheme_definitions sd ON sd.id=si.definition_id WHERE sd.name LIKE '%321%' OR si.id LIKE '%321%' OR si.lottery_label LIKE '%321%' ORDER BY si.updated_at DESC LIMIT 20`)
  defer rows.Close()
  for rows.Next(){ var id,n,st,lc,cp,up string; _=rows.Scan(&id,&n,&st,&lc,&cp,&up); fmt.Println(id,n,st,lc,cp,up) }
  fmt.Println("=== cloud pending period 1011120260701117 ===")
  rows2,_:=pool.Query(ctx,`SELECT scheme_id, period_no, third_party_period, status, third_party_bet_id, placed_at::text FROM cloud_bet_records WHERE period_no='1011120260701117' OR third_party_period='1011120260701117' ORDER BY placed_at DESC LIMIT 10`)
  defer rows2.Close()
  for rows2.Next(){ var sid,pn,tp,st,tpid,at string; _=rows2.Scan(&sid,&pn,&tp,&st,&tpid,&at); fmt.Println(sid,pn,tp,st,tpid,at) }
  fmt.Println("=== running schemes ===")
  rows3,_:=pool.Query(ctx,`SELECT si.id, COALESCE(sd.name,''), si.status, si.lottery_code, COALESCE(si.current_period,'') FROM scheme_instances si LEFT JOIN scheme_definitions sd ON sd.id=si.definition_id WHERE si.status IN ('running','pending') ORDER BY si.updated_at DESC LIMIT 30`)
  defer rows3.Close()
  for rows3.Next(){ var id,n,st,lc,cp string; _=rows3.Scan(&id,&n,&st,&lc,&cp); fmt.Println(id,n,st,lc,cp) }
}
