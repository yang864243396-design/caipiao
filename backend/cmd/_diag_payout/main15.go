package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  fmt.Println("=== running scheme instances ===")
  rows,_:=pool.Query(ctx,`SELECT si.id, si.scheme_name, si.lottery_code, si.updated_at FROM scheme_instances si WHERE si.status='running' ORDER BY si.updated_at DESC`)
  defer rows.Close(); n:=0
  for rows.Next(){ n++; var id,sn,lc string; var t any; _=rows.Scan(&id,&sn,&lc,&t); fmt.Printf("  %s name=%s lottery=%s updated=%v\n", id, sn, lc, t) }
  if n==0 { fmt.Println("  (none)") }
  fmt.Println("\n=== paused/pending ===")
  rows2,_:=pool.Query(ctx,`SELECT id, scheme_name, lottery_code, status FROM scheme_instances WHERE status IN ('paused','pending') ORDER BY updated_at DESC LIMIT 10`)
  defer rows2.Close()
  for rows2.Next(){ var id,sn,lc,st string; _=rows2.Scan(&id,&sn,&lc,&st); fmt.Printf("  %s %s %s %s\n", st, id, sn, lc) }
}
