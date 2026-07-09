package main
import ("context";"fmt";"github.com/joho/godotenv";"caipiao/backend/internal/config";"caipiao/backend/internal/guaji")
func main(){
  _=godotenv.Load(); c:=guaji.NewClient(config.Load().Guaji); ctx:=context.Background()
  target:="1014016200388"
  for _, path := range []string{"lottery_logs","lottery_log103s","lottery_log101"} {
    logs, err := c.FetchHistoryDrawLogs(ctx, path, 1, 50)
    fmt.Printf("path=%s err=%v count=%d\n", path, err, len(logs))
    found:=false
    for _, l := range logs {
      if l.Periods==target { found=true; fmt.Printf("  FOUND period=%s balls=%+v\n", l.Periods, l.Balls) }
    }
    if !found { fmt.Println("  not in first 50") }
  }
}
