package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  for _, code := range []string{"tron_ffc_15s","tron_ffc_3s","tron_ffc_1m","hash_ffc_1m"} {
    var ws,out string; var n int
    _=pool.QueryRow(ctx,`SELECT COALESCE(guaji_ws_key,''), COALESCE(outbound_lottery_code,'') FROM lottery_catalog WHERE code=$1`, code).Scan(&ws,&out)
    _=pool.QueryRow(ctx,`SELECT count(*) FROM lottery_draws WHERE lottery_code=$1 AND issue_no='1014016200388'`, code).Scan(&n)
    fmt.Printf("%s ws=%s outbound=%s draws=%d\n", code, ws, out, n)
  }
}
