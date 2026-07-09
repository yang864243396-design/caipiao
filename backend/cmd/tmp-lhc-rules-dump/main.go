package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/guaji/rulessync"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	data, err := rulessync.FetchRulesV2(context.Background(), cfg.Guaji.HTTPBase)
	if err != nil {
		panic(err)
	}
	tpl := data["8"]
	for _, g := range tpl.Groups {
		gn := g.Name
		if !strings.Contains(gn, "特") && gn != "七码" && gn == "生肖" {
			// still dump 生肖 for 总肖 context
		}
		if gn != "特码" && gn != "正特码" && gn != "特码头尾" && gn != "七码" && gn != "生肖" {
			continue
		}
		fmt.Printf("\n=== group %s ===\n", gn)
		for _, team := range g.Team {
			for _, rule := range team.Rule {
				b, _ := json.Marshal(rule)
				fmt.Printf("  id=%-4s team=%-12s name=%-8s json=%s\n", rule.ID, team.Name, rule.Name, string(b))
			}
		}
	}
	// dump raw rules API keys for type 8 top-level
	b, _ := json.MarshalIndent(tpl, "", "  ")
	_ = os.WriteFile("data/tmp-lhc-rules-type8.json", b, 0644)
	fmt.Println("\n(wrote data/tmp-lhc-rules-type8.json)")
}
