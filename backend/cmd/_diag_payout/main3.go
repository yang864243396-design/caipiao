package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db";"caipiao/backend/internal/guaji")
func main(){
  _=godotenv.Load(); cfg:=config.Load(); pool,_:=db.Connect(context.Background(), cfg.DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()

  var nullPlaced int; _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND placed_at IS NULL`).Scan(&nullPlaced)
  fmt.Println("placed_at IS NULL:", nullPlaced)

  fmt.Println("\n--- placed_at min/max ---")
  var minD,maxD any; _=pool.QueryRow(ctx,`SELECT MIN(placed_at), MAX(placed_at) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL`).Scan(&minD,&maxD)
  fmt.Printf("  min=%v max=%v\n", minD, maxD)

  fmt.Println("\n--- daily count ---")
  rows,_:=pool.Query(ctx,`SELECT date_trunc('day', placed_at AT TIME ZONE 'UTC')::date d, COUNT(*) FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND placed_at IS NOT NULL GROUP BY 1 ORDER BY 1 DESC LIMIT 10`)
  defer rows.Close(); for rows.Next(){ var d any; var n int; _=rows.Scan(&d,&n); fmt.Printf("  %v: %d\n", d, n) }

  fmt.Println("\n--- probe 3 oldest with tpid ---")
  type item struct{ on,tpid string; gid int64 }
  var items []item
  r2,_:=pool.Query(ctx,`SELECT order_no, third_party_bet_id, guaji_account_id FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL AND third_party_bet_id IS NOT NULL AND TRIM(third_party_bet_id)<>'' ORDER BY id ASC LIMIT 3`)
  defer r2.Close(); for r2.Next(){ var it item; _=r2.Scan(&it.on,&it.tpid,&it.gid); items=append(items,it) }

  client:=guaji.NewClient(cfg.Guaji); credKey,_:=guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
  for _, it := range items {
    var tokenEnc string
    _=pool.QueryRow(ctx,`SELECT access_token_enc FROM member_guaji_accounts WHERE id=$1`, it.gid).Scan(&tokenEnc)
    token,_:=guaji.DecryptSecret(credKey, tokenEnc)
    res, err := client.QuerySettlement(ctx, token, it.tpid)
    fmt.Printf("  %s tpid=%s err=%v settled=%v status=%s\n", it.on, it.tpid, err, res!=nil && res.Settled, func() string { if res!=nil { return res.Status }; return "" }())
  }

  var running int; _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM scheme_instances WHERE status='running'`).Scan(&running)
  fmt.Println("\nrunning schemes:", running)
}
