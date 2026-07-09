package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db";"caipiao/backend/internal/guaji")
func main(){
  _=godotenv.Load(); cfg:=config.Load(); pool,_:=db.Connect(context.Background(), cfg.DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  // sample recent pending with valid token
  var on,tpid string; var gid int64
  _=pool.QueryRow(ctx,`SELECT order_no, third_party_bet_id, guaji_account_id FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL ORDER BY placed_at DESC LIMIT 1`).Scan(&on,&tpid,&gid)
  fmt.Println("newest pending", on, tpid, gid)
  client:=guaji.NewClient(cfg.Guaji); credKey,_:=guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
  var tokenEnc string; _=pool.QueryRow(ctx,`SELECT access_token_enc FROM member_guaji_accounts WHERE id=$1`, gid).Scan(&tokenEnc)
  token, err := guaji.DecryptSecret(credKey, tokenEnc)
  fmt.Println("token len", len(token), "decrypt err", err)
  res, qerr := client.QuerySettlement(ctx, token, tpid)
  fmt.Printf("QuerySettlement err=%v res=%+v\n", qerr, res)

  // count distinct guaji accounts in pending
  var accts int; _=pool.QueryRow(ctx,`SELECT COUNT(DISTINCT guaji_account_id) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL`).Scan(&accts)
  fmt.Println("distinct guaji accounts in pending:", accts)

  // guaji account health
  rows,_:=pool.Query(ctx,`SELECT ga.id, ga.is_active, ga.token_expires_at, COUNT(b.id) n FROM member_guaji_accounts ga JOIN bet_orders b ON b.guaji_account_id=ga.id AND b.status='pending' GROUP BY ga.id, ga.is_active, ga.token_expires_at ORDER BY n DESC`)
  defer rows.Close(); fmt.Println("\nguaji accounts with pending:")
  for rows.Next(){ var id int64; var active bool; var exp any; var n int; _=rows.Scan(&id,&active,&exp,&n); fmt.Printf("  id=%d active=%v expires=%v pending=%d\n", id, active, exp, n) }
}
