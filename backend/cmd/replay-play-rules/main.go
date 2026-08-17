// replay-play-rules compares historical formal settlements with the current
// declarative catalogue rules. It is intentionally read-only.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/playrules"
	"caipiao/backend/internal/schemes"
)

type replayTotals struct {
	Candidates int `json:"candidates"`
	Evaluated  int `json:"evaluated"`
	Matched    int `json:"matched"`
	Mismatched int `json:"mismatched"`
	Skipped    int `json:"skipped"`
}

type ruleTotals struct {
	Rule       string `json:"rule"`
	Evaluated  int    `json:"evaluated"`
	Matched    int    `json:"matched"`
	Mismatched int    `json:"mismatched"`
}

type replayReport struct {
	Totals            replayTotals `json:"totals"`
	Rules             []ruleTotals `json:"rules"`
	RulesFullyMatched int          `json:"rulesFullyMatched"`
	RulesWithMismatch int          `json:"rulesWithMismatch"`
}

func main() {
	limit := flag.Int("limit", 0, "maximum historical rows to replay; 0 means all")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cfg := config.Load()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()
	rows, err := sqlcdb.New(pool).ListHistoricalRuleReplayCandidates(ctx)
	if err != nil {
		log.Fatalf("list historical candidates: %v", err)
	}
	totals := replayTotals{Candidates: len(rows)}
	byRule := map[string]*ruleTotals{}
	for _, row := range rows {
		if *limit > 0 && totals.Evaluated+totals.Skipped >= *limit {
			break
		}
		input, balls, ok := replayInput(row.Config, row.Balls)
		if !ok {
			totals.Skipped++
			continue
		}
		evaluation, err := schemes.EvaluateCatalogRule(input, balls, row.BetContent)
		if err != nil {
			totals.Skipped++
			continue
		}
		totals.Evaluated++
		key := input.Locator.TemplateCode + "/" + input.Locator.TypeID + "/" + input.Locator.SubID
		stats := byRule[key]
		if stats == nil {
			stats = &ruleTotals{Rule: key}
			byRule[key] = stats
		}
		stats.Evaluated++
		if evaluation.Hit == (strings.TrimSpace(row.Status) == "hit") {
			totals.Matched++
			stats.Matched++
		} else {
			totals.Mismatched++
			stats.Mismatched++
		}
	}
	rules := make([]ruleTotals, 0, len(byRule))
	fullyMatched, withMismatch := 0, 0
	for _, stats := range byRule {
		rules = append(rules, *stats)
		if stats.Mismatched == 0 {
			fullyMatched++
		} else {
			withMismatch++
		}
	}
	_ = json.NewEncoder(osStdout{}).Encode(replayReport{
		Totals: totals, Rules: rules, RulesFullyMatched: fullyMatched, RulesWithMismatch: withMismatch,
	})
}

func replayInput(rawConfig, rawBalls []byte) (schemes.CatalogRuleCompileInput, []string, bool) {
	var cfg map[string]interface{}
	if json.Unmarshal(rawConfig, &cfg) != nil {
		return schemes.CatalogRuleCompileInput{}, nil, false
	}
	var balls []string
	if json.Unmarshal(rawBalls, &balls) != nil || len(balls) == 0 {
		return schemes.CatalogRuleCompileInput{}, nil, false
	}
	value := func(keys ...string) string {
		for _, key := range keys {
			if v, ok := cfg[key].(string); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	input := schemes.CatalogRuleCompileInput{
		Locator: playrules.Locator{TemplateCode: value("playTemplate"), TypeID: value("typeId", "playTypeId"), SubID: value("subId", "subPlayId")},
		BetMode: value("betMode"), PlayMethod: value("playMethod", "playMethodLabel", "subPlayLabel", "guajiFullName"),
	}
	if input.Locator.TemplateCode == "" || input.Locator.TypeID == "" || input.Locator.SubID == "" {
		return schemes.CatalogRuleCompileInput{}, nil, false
	}
	return input, balls, true
}

// osStdout keeps the replay command easy to exercise in tests without adding
// a second output dependency.
type osStdout struct{}

func (osStdout) Write(p []byte) (int, error) { return fmt.Print(string(p)) }
