package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); cfg:=config.Load(); ctx:=context.Background(); pool,_:=db.Connect(ctx,cfg.DatabaseURL,cfg.DBMaxConns,cfg.DBMinConns); defer pool.Close()
  rows,_:=pool.Query(ctx,`SELECT type_id, label FROM play_types WHERE template_code='syxw_std' ORDER BY sort_order`)
  defer rows.Close()
  for rows.Next(){ var t,l string; _=rows.Scan(&t,&l); fmt.Println(t,l)}
}
