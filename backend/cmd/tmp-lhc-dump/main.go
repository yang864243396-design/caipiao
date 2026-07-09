package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/db")
func main(){
  _=godotenv.Load(); cfg:=config.Load(); ctx:=context.Background()
  pool,_:=db.Connect(ctx,cfg.DatabaseURL,cfg.DBMaxConns,cfg.DBMinConns); defer pool.Close()
  ids:=[]string{"323","325","331","333","339","345","347","380","382","384"}
  rows,_:=pool.Query(ctx,`SELECT sp.type_id,sp.sub_id,sp.label,sp.outbound_play_code,COALESCE(pt.label,''),sp.segment_rule::text FROM sub_plays sp LEFT JOIN play_types pt ON pt.template_code=sp.template_code AND pt.type_id=sp.type_id WHERE sp.template_code='lhc_std' AND sp.outbound_play_code = ANY($1) ORDER BY sp.outbound_play_code`,ids)
  defer rows.Close()
  for rows.Next(){var a,b,c,d,e,f string; rows.Scan(&a,&b,&c,&d,&e,&f); fmt.Printf("%s/%s label=%s teamType=%s seg=%s\n",a,b,c,e,f)}
}
