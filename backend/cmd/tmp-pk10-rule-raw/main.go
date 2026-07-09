package main

import (
	"context"
	"encoding/json"
	"fmt"

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
	tpl := data["3"]
	for _, g := range tpl.Groups {
		for _, team := range g.Team {
			for _, rule := range team.Rule {
				if rule.ID == "221" || rule.ID == "222" || rule.ID == "223" {
					b, _ := json.MarshalIndent(rule, "", "  ")
					fmt.Printf("--- rule %s ---\n%s\n", rule.ID, string(b))
				}
			}
		}
	}
}
