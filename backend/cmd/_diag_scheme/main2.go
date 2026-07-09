package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db";"caipiao/backend/internal/guaji";"caipiao/backend/internal/guaji/accountsvc")
func main(){
  _=godotenv.Load(); cfg:=config.Load(); pool,_:=db.Connect(context.Background(), cfg.DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  for _, code := range []string{"tron_ffc_1m","hash_ffc_1m","bnb_ffc_1m"} {
    var outbound, ws, name string
    _=pool.QueryRow(ctx,`SELECT display_name, COALESCE(outbound_lottery_code,''), COALESCE(guaji_ws_key,'') FROM lottery_catalog WHERE code=$1`, code).Scan(&name,&outbound,&ws)
    var cnt int
    _=pool.QueryRow(ctx,`SELECT count(*) FROM lottery_draws WHERE lottery_code=$1 AND issue_no='1014016200388'`, code).Scan(&cnt)
    fmt.Printf("%s (%s) outbound=%s ws=%s draws_for_period=%d\n", code, name, outbound, ws, cnt)
  }
  // bet order + third party
  var on, iss, st, tpid string; var gid int64
  _=pool.QueryRow(ctx,`SELECT order_no, issue_no, status, COALESCE(third_party_bet_id,''), COALESCE(guaji_account_id,0) FROM bet_orders WHERE order_no='BO11782963024341'`).Scan(&on,&iss,&st,&tpid,&gid)
  fmt.Printf("\norder %s issue=%s status=%s tpid=%s guaji=%d\n", on, iss, st, tpid, gid)
  client:=guaji.NewClient(cfg.Guaji)
  credKey,_:=guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
  var tokenEnc string
  _=pool.QueryRow(ctx,`SELECT access_token_enc FROM member_guaji_accounts WHERE id=$1`, gid).Scan(&tokenEnc)
  token,_:=guaji.DecryptSecret(credKey, tokenEnc)
  res, err := client.QuerySettlement(ctx, token, tpid)
  fmt.Printf("QuerySettlement err=%v res=%+v\n", err, res)
  // try sync
  svc:=accountsvc.NewService(pool, client, cfg.Guaji.CredentialsKey, cfg.JWTSecret)
  row, err := svc.LoadPendingGuajiBetOrder(ctx, on)
  if err != nil { fmt.Println("load", err); return }
  w := svc.NewPayoutSyncWorker(nil, nil)
  if err := w.SyncOne(ctx, row); err != nil { fmt.Println("sync err", err) } else { fmt.Println("sync done") }
  _=pool.QueryRow(ctx,`SELECT status FROM bet_orders WHERE order_no=$1`, on).Scan(&st)
  fmt.Println("order after sync", st)
}
