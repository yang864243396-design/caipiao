package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  fmt.Println("=== all scheme_instances ===")
  rows,_:=pool.Query(ctx,`SELECT si.id, si.member_id, si.definition_id, si.status, si.status_reason, si.lottery_code, COALESCE(si.current_period,''), COALESCE(si.last_settled_issue,''), si.sim_bet, si.updated_at::text FROM scheme_instances si ORDER BY si.updated_at DESC`)
  defer rows.Close()
  for rows.Next(){ var id,def,st,sr,lc,cp,lsi,up string; var mid int64; var sim bool; _=rows.Scan(&id,&mid,&def,&st,&sr,&lc,&cp,&lsi,&sim,&up); fmt.Printf("%s member=%d def=%s %s/%s %s period=%s lastSettled=%s sim=%v %s\n", id,mid,def,st,sr,lc,cp,lsi,sim,up) }
  fmt.Println("\n=== all scheme_definitions ===")
  rows2,_:=pool.Query(ctx,`SELECT id, member_id, name, lottery_code FROM scheme_definitions`)
  defer rows2.Close()
  for rows2.Next(){ var id,name,lc string; var mid int64; _=rows2.Scan(&id,&mid,&name,&lc); fmt.Printf("%s member=%d name=%q lottery=%s\n", id,mid,name,lc) }
  fmt.Println("\n=== cloud_bet period 1011120260701117 ===")
  rows3,_:=pool.Query(ctx,`SELECT scheme_id, lottery_code, period_no, third_party_period, status, third_party_bet_id, placed_at::text FROM cloud_bet_records WHERE period_no LIKE '%1011120260701117%' OR third_party_period LIKE '%1011120260701117%'`)
  defer rows3.Close()
  for rows3.Next(){ var sid,lc,pn,tp,st,tpid,at string; _=rows3.Scan(&sid,&lc,&pn,&tp,&st,&tpid,&at); fmt.Println(sid,lc,pn,tp,st,tpid,at) }
  fmt.Println("\n=== cloud_bet pending recent ===")
  rows4,_:=pool.Query(ctx,`SELECT scheme_id, lottery_code, period_no, third_party_period, status, placed_at::text FROM cloud_bet_records WHERE status='pending' ORDER BY placed_at DESC LIMIT 15`)
  defer rows4.Close()
  for rows4.Next(){ var sid,lc,pn,tp,st,at string; _=rows4.Scan(&sid,&lc,&pn,&tp,&st,&at); fmt.Println(sid,lc,pn,tp,st,at) }
}
