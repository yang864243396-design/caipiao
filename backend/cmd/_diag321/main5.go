package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,err:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); if err!=nil{panic(err)}; defer pool.Close(); ctx:=context.Background()
  var n int
  if err:=pool.QueryRow(ctx,`SELECT count(*) FROM scheme_instances`).Scan(&n); err!=nil{panic(err)}
  fmt.Println("count", n)
  rows, err := pool.Query(ctx, `SELECT id FROM scheme_instances`)
  if err != nil { panic(err) }
  defer rows.Close()
  for rows.Next() {
    var id string
    if err:=rows.Scan(&id); err!=nil{panic(err)}
    fmt.Println("id", id)
  }
  if err:=rows.Err(); err!=nil{panic(err)}
  fmt.Println("cloud pending count")
  _=pool.QueryRow(ctx,`SELECT count(*) FROM cloud_bet_records WHERE status='pending'`).Scan(&n)
  fmt.Println(n)
}
