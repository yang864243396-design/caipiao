package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  rows,_:=pool.Query(ctx,`SELECT si.id, si.status, si.status_reason, si.lottery_code, si.lottery_label, COALESCE(si.last_settled_issue,''), si.sim_bet, si.updated_at::text, COALESCE(sd.name,'') FROM scheme_instances si LEFT JOIN scheme_definitions sd ON sd.id=si.definition_id ORDER BY si.updated_at DESC`)
  defer rows.Close()
  for rows.Next(){ var id,st,sr,lc,ll,lsi,def string; var sim bool; var up string; _=rows.Scan(&id,&st,&sr,&lc,&ll,&lsi,&sim,&up,&def); fmt.Printf("id=%s def=%q status=%s/%s lottery=%s lastSettled=%s sim=%v up=%s\n", id,def,st,sr,lc,lsi,sim,up) }
  fmt.Println("--- definitions ---")
  rows2,_:=pool.Query(ctx,`SELECT id, name, lottery_code FROM scheme_definitions`)
  defer rows2.Close()
  for rows2.Next(){ var id,name,lc string; _=rows2.Scan(&id,&name,&lc); fmt.Printf("%s name=%q lottery=%s\n", id,name,lc) }
  fmt.Println("--- cloud pending for instances ---")
  for _, inst := range []string{"inst-1-1782963020261","inst-1-1782904423963"} {
    fmt.Println("scheme", inst)
    rows3,_:=pool.Query(ctx,`SELECT period_no, third_party_period, status, placed_at::text FROM cloud_bet_records WHERE scheme_id=$1 ORDER BY placed_at DESC LIMIT 5`, inst)
    for rows3.Next(){ var pn,tp,st,at string; _=rows3.Scan(&pn,&tp,&st,&at); fmt.Printf("  %s / %s %s %s\n", pn,tp,st,at) }
    rows3.Close()
  }
  fmt.Println("--- search period ---")
  rows4,_:=pool.Query(ctx,`SELECT scheme_id, lottery_code, period_no, third_party_period, status FROM cloud_bet_records WHERE period_no LIKE '%0701117%' OR third_party_period LIKE '%0701117%' LIMIT 20`)
  defer rows4.Close()
  for rows4.Next(){ var sid,lc,pn,tp,st string; _=rows4.Scan(&sid,&lc,&pn,&tp,&st); fmt.Println(sid,lc,pn,tp,st) }
}
