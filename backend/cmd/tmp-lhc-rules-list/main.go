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
	if err != nil {
		panic(err)
	}
	tpl := data["8"]
	fmt.Printf("type8 groups=%d\n", len(tpl.Groups))
	for _, g := range tpl.Groups {
		for _, team := range g.Team {
			for _, rule := range team.Rule {
				id, _ := strconv.Atoi(rule.ID)
				if id >= 250 && id <= 450 {
					fmt.Printf("group=%-10s team=%-14s id=%-4s name=%-10s full=%s\n",
						g.Name, team.Name, rule.ID, rule.Name, rule.FullName)
				}
			}
		}
	}
}
