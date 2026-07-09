package main
import ("context";"encoding/json";"fmt";"io";"net/http";"strings")
func main() {
  ctx := context.Background()
  req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.v6hs1.com/api/games/rules/v2", nil)
  resp, err := http.DefaultClient.Do(req)
  if err != nil { panic(err) }
  defer resp.Body.Close()
  raw, _ := io.ReadAll(resp.Body)
  var env struct { Data map[string]json.RawMessage `json:"data"` }
  json.Unmarshal(raw, &env)
  var tpl struct { Groups []struct { Name string; Team []struct { Name string; Rule []map[string]any `json:"rule"` } `json:"team"` } `json:"groups"` }
  json.Unmarshal(env.Data["3"], &tpl)
  for _, g := range tpl.Groups {
    for _, team := range g.Team {
      for _, rule := range team.Rule {
        id := fmt.Sprint(rule["id"])
        if id=="221"||id=="222"||id=="223"||id=="217"||id=="207" {
          b,_:=json.MarshalIndent(rule,"","  ")
          fmt.Printf("=== %s / %s id=%s ===\n%s\n", g.Name, team.Name, id, string(b))
        }
      }
    }
  }
  text := string(env.Data["3"])
  for _, pat := range []string{"bet_type","content_type","input_type","221"} {
    idx := strings.Index(text, pat)
    fmt.Printf("pat %s idx=%d\n", pat, idx)
  }
}
