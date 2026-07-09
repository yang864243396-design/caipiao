// guaji-bet-audit：拉取第三方历史注单原始 JSON，打印 rule_id / bet_content / solo / bets_nums。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	account := flag.String("account", "vs8888", "member account")
	gameID := flag.Int("game", 0, "filter game_id (0=all)")
	ruleFilter := flag.String("rule", "", "filter rule_id")
	limit := flag.Int("limit", 100, "page limit")
	pages := flag.Int("pages", 1, "max pages to scan")
	rawOnly := flag.Bool("raw", false, "print full JSON only")
	flag.Parse()

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	token, err := loadToken(ctx, pool, cfg, *account)
	if err != nil {
		panic(err)
	}

	client := guaji.NewClient(cfg.Guaji)
	if *rawOnly {
		env, body, err := client.FetchWebBetsRaw(ctx, token, *limit, 1)
		if err != nil {
			panic(err)
		}
		_ = env
		fmt.Println(string(body))
		return
	}

	totalRows := 0
	matched := 0
	for page := 1; page <= *pages; page++ {
		env, body, err := client.FetchWebBetsRaw(ctx, token, *limit, page)
		if err != nil {
			panic(err)
		}
		_ = env
		var wrap struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(body, &wrap); err != nil {
			fmt.Println(string(body))
			return
		}
		if len(wrap.Data) == 0 {
			break
		}
		totalRows += len(wrap.Data)
		for _, row := range wrap.Data {
			gid := intNum(row["game_id"])
			if *gameID > 0 && gid != *gameID {
				continue
			}
			for _, item := range extractBetContentItems(row) {
				rid := strVal(item["rule_id"])
				if *ruleFilter != "" && rid != *ruleFilter {
					continue
				}
				matched++
				fmt.Printf("id=%v game=%v rule=%s content=%q bets=%v solo=%v amount=%v periods=%v status=%v name=%v\n",
					row["id"], gid, rid, strVal(item["bet_content"]), item["bets_nums"], item["solo"], item["bet_amount"], row["periods"], row["status"], item["rule_full_name"])
			}
		}
		if len(wrap.Data) < *limit {
			break
		}
	}
	fmt.Fprintf(os.Stderr, "scanned=%d matched=%d pages=%d\n", totalRows, matched, *pages)
}

func loadToken(ctx context.Context, pool *db.Pool, cfg config.Config, account string) (string, error) {
	key, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	var memberID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, account).Scan(&memberID); err != nil {
		return "", err
	}
	var tokenEnc string
	if err := pool.QueryRow(ctx, `
SELECT access_token_enc FROM member_guaji_accounts
WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc); err != nil {
		return "", err
	}
	return guaji.DecryptSecret(key, tokenEnc)
}

func strVal(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func intNum(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return 0
	}
}

// extractBetContentItems 兼容 web_bets 列表两种结构：bet_contents[] 或 bet_content.bet_content{}。
func extractBetContentItems(row map[string]any) []map[string]any {
	out := make([]map[string]any, 0, 1)
	if contents, ok := row["bet_contents"].([]any); ok {
		for _, c := range contents {
			if m, ok := c.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	if bc, ok := row["bet_content"].(map[string]any); ok {
		if inner, ok := bc["bet_content"].(map[string]any); ok {
			out = append(out, inner)
		}
	}
	return out
}
