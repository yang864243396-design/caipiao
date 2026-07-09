package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/guaji")
func main(){
  _=godotenv.Load(); c:=guaji.NewClient(config.Load().Guaji); ctx:=context.Background()
  for _, path := range []string{"lottery_log103s","lottery_log303s","lottery_log503s","lottery_log3s","lottery_log5s"} {
    logs, err := c.FetchHistoryDrawLogs(ctx, path, 1, 3)
    fmt.Printf("path=%s err=%v", path, err)
    if len(logs)>0 { fmt.Printf(" latest=%s", logs[0].Periods) }
    fmt.Println()
  }
}
