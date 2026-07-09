package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  rows,_:=pool.Query(ctx,`SELECT id, scheme_name, lottery_code, status, status_reason, updated_at FROM scheme_instances ORDER BY updated_at DESC LIMIT 10`)
  defer rows.Close()
  for rows.Next(){ var id,sn,lc,st string; var reason *string; var t any; _=rows.Scan(&id,&sn,&lc,&st,&reason,&t); r:=""; if reason!=nil{r=*reason}; fmt.Printf("%s [%s] %s lottery=%s reason=%q updated=%v\n", id, st, sn, lc, r, t) }
}
