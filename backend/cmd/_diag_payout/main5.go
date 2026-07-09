package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  rows,_:=pool.Query(ctx,`SELECT guaji_account_id, COUNT(*) n FROM bet_orders WHERE status='pending' AND guaji_account_id IS NOT NULL GROUP BY guaji_account_id ORDER BY n DESC`)
  defer rows.Close(); fmt.Println("pending by guaji_account_id:")
  for rows.Next(){ var id int64; var n int; _=rows.Scan(&id,&n); fmt.Printf("  %d: %d\n", id, n) }

  fmt.Println("\norphan guaji_account_id (no member_guaji_accounts row):")
  var orphan int; _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM bet_orders b WHERE b.status='pending' AND b.guaji_account_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM member_guaji_accounts ga WHERE ga.id=b.guaji_account_id)`).Scan(&orphan)
  fmt.Println(" ", orphan)

  fmt.Println("\n--- why stuck: sample account id=1 ---")
  var exists bool; _=pool.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM member_guaji_accounts WHERE id=1)`).Scan(&exists)
  fmt.Println("account 1 exists:", exists)
  if exists {
    var active bool; var exp any; _=pool.QueryRow(ctx,`SELECT is_active, token_expires_at FROM member_guaji_accounts WHERE id=1`).Scan(&active,&exp)
    fmt.Printf("  active=%v expires=%v\n", active, exp)
  }
}
