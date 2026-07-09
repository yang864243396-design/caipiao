package main
import (
  "context"
  "fmt"
  "strconv"
  "github.com/joho/godotenv"
  "caipiao/backend/internal/config"
  "caipiao/backend/internal/guaji/rulessync"
)
func main() {
  _ = godotenv.Load()
  cfg := config.Load()
  data, err := rulessync.FetchRulesV2(context.Background(), cfg.Guaji.HTTPBase)
  if err != nil { panic(err) }
  tpl := data["3"]
  for _, g := range tpl.Groups {
    for _, team := range g.Team {
      for _, rule := range team.Rule {
        id, _ := strconv.Atoi(rule.ID)
        if id >= 210 && id <= 225 {
          fmt.Printf("group=%-8s team=%-12s id=%-4s name=%-16s full=%s\n", g.Name, team.Name, rule.ID, rule.Name, rule.FullName)
        }
      }
    }
  }
}
