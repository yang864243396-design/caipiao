package main
import (
  "context"; "encoding/json"; "fmt"
  "github.com/joho/godotenv"
  "caipiao/backend/internal/config"; "caipiao/backend/internal/db"
)
func main() {
  _=godotenv.Load()
  pool,_:=db.Connect(context.Background(), config.Load().DatabaseURL, 3, 1); defer pool.Close()
  ctx:=context.Background()
  defID:="def-1-1784610856831"
  var cfg []byte
  if err:=pool.QueryRow(ctx,`SELECT config FROM scheme_definitions WHERE id=$1`, defID).Scan(&cfg); err!=nil { panic(err) }
  var m map[string]any
  _=json.Unmarshal(cfg,&m)
  hcw,_:=m["hotColdWarm"].(map[string]any)
  if hcw==nil { panic("no hcw") }
  hcw["faultCount"]=1
  m["hotColdWarm"]=hcw
  out,_:=json.Marshal(m)
  tag,err:=pool.Exec(ctx,`UPDATE scheme_definitions SET config=$2::jsonb, updated_at=now() WHERE id=$1`, defID, out)
  if err!=nil { panic(err) }
  fmt.Println("faultCount -> 1, rows=", tag.RowsAffected())
}
