package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db";"caipiao/backend/internal/db/sqlcdb")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close()
  q:=sqlcdb.New(pool)
  rows,_:=q.ListPendingGuajiBetOrders(context.Background(), 500)
  fmt.Println("pending count", len(rows))
  for _,r:=range rows { if r.OrderNo=="BO11782905494687"{ fmt.Printf("FOUND pos in queue tpid=%s guaji=%v\n", r.ThirdPartyBetID.String, r.GuajiAccountID) } }
  var cfg []byte
  _=pool.QueryRow(context.Background(),`SELECT config FROM scheme_definitions WHERE id='def-1-1782904422829'`).Scan(&cfg)
  fmt.Println("def config", string(cfg))
}
