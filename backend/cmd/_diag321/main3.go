package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); cfg:=config.Load(); fmt.Println("DB_HOST", cfg.DatabaseURL[:min(40,len(cfg.DatabaseURL))]+"...")
  pool,err:=db.Connect(context.Background(), cfg.DatabaseURL,5,1); if err!=nil{panic(err)}; defer pool.Close(); ctx:=context.Background()
  var n int; _=pool.QueryRow(ctx,`SELECT count(*) FROM scheme_instances`).Scan(&n); fmt.Println("scheme_instances count", n)
  _=pool.QueryRow(ctx,`SELECT count(*) FROM scheme_definitions`).Scan(&n); fmt.Println("scheme_definitions count", n)
  _=pool.QueryRow(ctx,`SELECT count(*) FROM cloud_bet_records`).Scan(&n); fmt.Println("cloud_bet_records count", n)
  rows,_:=pool.Query(ctx,`SELECT id, name FROM scheme_definitions ORDER BY updated_at DESC LIMIT 15`)
  defer rows.Close(); fmt.Println("recent definitions:")
  for rows.Next(){ var id,name string; _=rows.Scan(&id,&name); fmt.Printf("  %s | %s\n", id, name) }
}
func min(a,b int)int{if a<b{return a};return b}
