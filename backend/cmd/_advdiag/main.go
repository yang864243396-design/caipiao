package main
import (
  "context"
  "encoding/json"
  "fmt"
  "github.com/joho/godotenv"
  "caipiao/backend/internal/config"
  "caipiao/backend/internal/db"
)
func main() {
  _ = godotenv.Load()
  cfg := config.Load()
  pool, _ := db.Connect(context.Background(), cfg.DatabaseURL, 4, 1)
  defer pool.Close()
  defID := "def-1-1785294141525"
  var raw []byte
  var updated interface{}
  _ = pool.QueryRow(context.Background(), `SELECT config, updated_at FROM scheme_definitions WHERE id=$1`, defID).Scan(&raw, &updated)
  fmt.Println("config bytes", len(raw), "updated", updated)
  fmt.Println(string(raw)[:min(2000, len(raw))])
  fmt.Println("--- templates member 1 ---")
  rows, _ := pool.Query(context.Background(), `
SELECT id, name, member_id, COALESCE(definition_id::text,''), config, created_at, updated_at
FROM scheme_templates WHERE member_id=1 OR definition_id=$1 OR name ILIKE '%新%'
ORDER BY updated_at DESC NULLS LAST LIMIT 40`, defID)
  defer rows.Close()
  for rows.Next() {
    var id, name, did string
    var mid *int64
    var conf []byte
    var cAt, uAt interface{}
    _ = rows.Scan(&id, &name, &mid, &did, &conf, &cAt, &uAt)
    fmt.Printf("id=%s name=%q mid=%v def=%s upd=%v\n", id, name, mid, did, uAt)
    var m map[string]interface{}
    _ = json.Unmarshal(conf, &m)
    b, _ := json.Marshal(m["rounds"])
    fmt.Println("  rounds", string(b))
  }
}
func min(a,b int) int { if a<b { return a }; return b }
