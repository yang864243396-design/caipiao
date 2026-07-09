package main
import ("context";"encoding/json";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  inst:="inst-1-1782904423963"
  var st,sr,lc,ll,lsi,defID string; var sim bool; var mid int64
  err:=pool.QueryRow(ctx,`SELECT member_id, definition_id, status, COALESCE(status_reason,''), lottery_code, lottery_label, COALESCE(last_settled_issue,''), sim_bet FROM scheme_instances WHERE id=$1`, inst).Scan(&mid,&defID,&st,&sr,&lc,&ll,&lsi,&sim)
  fmt.Println("instance", inst, "err", err, "member", mid, "def", defID, st, sr, lc, ll, "lastSettled", lsi, "sim", sim)
  var dname, dlc string
  _=pool.QueryRow(ctx,`SELECT name, lottery_code FROM scheme_definitions WHERE id=$1`, defID).Scan(&dname,&dlc)
  fmt.Println("definition name", dname, "lottery", dlc)
  var outbound, ws string
  _=pool.QueryRow(ctx,`SELECT COALESCE(outbound_lottery_code,''), COALESCE(guaji_ws_key,'') FROM lottery_catalog WHERE code=$1`, lc).Scan(&outbound,&ws)
  fmt.Println("catalog outbound", outbound, "ws", ws)
  fmt.Println("draws for pending periods:")
  rows,_:=pool.Query(ctx,`SELECT issue_no, balls::text, drawn_at::text FROM lottery_draws WHERE lottery_code=$1 AND issue_no LIKE '1011120260701117%' ORDER BY drawn_at DESC`, lc)
  defer rows.Close()
  for rows.Next(){ var iss,balls,at string; _=rows.Scan(&iss,&balls,&at); fmt.Println(iss,balls,at) }
  fmt.Println("bet_orders pending:")
  rows2,_:=pool.Query(ctx,`SELECT b.order_no,b.issue_no,b.status,b.third_party_bet_id,b.guaji_account_id,b.placed_at::text FROM bet_orders b JOIN cloud_bet_records c ON c.bet_order_no=b.order_no WHERE c.scheme_id=$1 AND b.status='pending' ORDER BY b.placed_at DESC`, inst)
  defer rows2.Close()
  for rows2.Next(){ var on,iss,st,tpid,at string; var gid int64; _=rows2.Scan(&on,&iss,&st,&tpid,&gid,&at); fmt.Printf("%s issue=%s %s tpid=%s guaji=%d %s\n", on,iss,st,tpid,gid,at) }
  fmt.Println("cloud pending all:")
  rows3,_:=pool.Query(ctx,`SELECT period_no, third_party_period, status, third_party_bet_id, bet_order_no, pnl, placed_at::text FROM cloud_bet_records WHERE scheme_id=$1 AND status='pending'`, inst)
  defer rows3.Close()
  for rows3.Next(){ var pn,tp,st,tpid,bon,at string; var pnl float64; _=rows3.Scan(&pn,&tp,&st,&tpid,&bon,&pnl,&at); fmt.Printf("pn=%s tp=%s %s tpid=%s order=%s pnl=%.2f %s\n", pn,tp,st,tpid,bon,pnl,at) }
  fmt.Println("worker state:")
  var wsjson []byte
  _=pool.QueryRow(ctx,`SELECT worker_state FROM scheme_instances WHERE id=$1`, inst).Scan(&wsjson)
  var pretty json.RawMessage = wsjson
  b,_:=json.MarshalIndent(pretty,"","  "); fmt.Println(string(b))
}
