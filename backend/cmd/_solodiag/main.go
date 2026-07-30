package main
import (
  "encoding/json"
  "fmt"
  "caipiao/backend/internal/guajibet"
)
func main() {
  seg,_ := json.Marshal(map[string]string{"guajiGroup":"中三码","guajiTeam":"中三组选","guajiFullName":"中三组六","guajiRuleId":"261"})
  meta := guajibet.ParseRuleMeta("ssc_std","g002","261","中三组六","中三码",seg,"261")
  for _, c := range []string{"0","3,4","3,4,5","1,2,3,4"} {
    n := guajibet.CountBetNums(meta,c)
    fmt.Printf("%q bets=%d solo=%v\n", c, n, guajibet.ResolveSolo(meta,c,n))
  }
}
