package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL,5,1); defer pool.Close(); ctx:=context.Background()
  var withTP, withoutTP, noScheme int
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records WHERE guaji_account_id=4 AND status='pending' AND third_party_bet_id IS NOT NULL AND TRIM(third_party_bet_id)<>''`).Scan(&withTP)
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records WHERE guaji_account_id=4 AND status='pending' AND (third_party_bet_id IS NULL OR TRIM(third_party_bet_id)='')`).Scan(&withoutTP)
  _=pool.QueryRow(ctx,`SELECT COUNT(*) FROM cloud_bet_records c WHERE c.guaji_account_id=4 AND c.status='pending' AND NOT EXISTS (SELECT 1 FROM scheme_instances si WHERE si.id=c.scheme_id)`).Scan(&noScheme)
  fmt.Printf("cloud pending guaji=4: with_tpid=%d without_tpid=%d no_scheme_instance=%d\n", withTP, withoutTP, noScheme)
}
