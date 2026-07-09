package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  defID:="def-1-1782963020232"
  fmt.Println("=== definition ===")
  var name, lc string; var cfg []byte
  _=pool.QueryRow(ctx,`SELECT COALESCE(name,''), lottery_code, config FROM scheme_definitions WHERE id=$1`, defID).Scan(&name,&lc,&cfg)
  fmt.Printf("name=%q lottery=%s config=%s\n", name, lc, string(cfg))
  fmt.Println("=== instances ===")
  rows,_:=pool.Query(ctx,`SELECT id, status, status_reason, lottery_code, COALESCE(last_settled_issue,''), sim_bet, updated_at::text FROM scheme_instances WHERE definition_id=$1 ORDER BY updated_at DESC`, defID)
  defer rows.Close()
  var instID string
  for rows.Next(){ var id,st,sr,lc2,lsi,up string; var sim bool; _=rows.Scan(&id,&st,&sr,&lc2,&lsi,&sim,&up); instID=id; fmt.Printf("inst=%s %s/%s lottery=%s lastSettled=%s sim=%v up=%s\n", id,st,sr,lc2,lsi,sim,up) }
  period:="1014016200388"
  fmt.Println("\n=== lottery_draws ===")
  rows2,_:=pool.Query(ctx,`SELECT lottery_code, issue_no, balls::text, drawn_at::text FROM lottery_draws WHERE issue_no=$1 OR issue_no LIKE $2 ORDER BY drawn_at DESC`, period, period+"%")
  defer rows2.Close()
  n:=0
  for rows2.Next(){ n++; var lc3,iss,balls,at string; _=rows2.Scan(&lc3,&iss,&balls,&at); fmt.Printf("%s %s %s %s\n", lc3,iss,balls,at) }
  if n==0 { fmt.Println("(no draw)") }
  if instID=="" { return }
  fmt.Println("\n=== cloud bets ===")
  rows3,_:=pool.Query(ctx,`SELECT period_no, third_party_period, status, third_party_bet_id, bet_order_no, placed_at::text FROM cloud_bet_records WHERE scheme_id=$1 ORDER BY placed_at DESC LIMIT 8`, instID)
  defer rows3.Close()
  for rows3.Next(){ var pn,tp,st,tpid,bon,at string; _=rows3.Scan(&pn,&tp,&st,&tpid,&bon,&at); fmt.Printf("pn=%s tp=%s %s tpid=%s order=%s %s\n", pn,tp,st,tpid,bon,at) }
  fmt.Println("\n=== catalog ===")
  var outbound, ws string
  _=pool.QueryRow(ctx,`SELECT COALESCE(outbound_lottery_code,''), COALESCE(guaji_ws_key,'') FROM lottery_catalog WHERE code=$1`, lc).Scan(&outbound,&ws)
  fmt.Printf("outbound=%s ws=%s\n", outbound, ws)
}
