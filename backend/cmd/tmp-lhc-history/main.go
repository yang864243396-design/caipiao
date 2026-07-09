package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	key, _ := guaji.CredentialsKey(cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	var memberID int64
	_ = pool.QueryRow(ctx, `SELECT id FROM members WHERE account=$1`, "vs8888").Scan(&memberID)
	var tokenEnc string
	_ = pool.QueryRow(ctx, `SELECT access_token_enc FROM member_guaji_accounts WHERE member_id=$1 AND is_active=true ORDER BY id DESC LIMIT 1`, memberID).Scan(&tokenEnc)
	token, _ := guaji.DecryptSecret(key, tokenEnc)
	client := guaji.NewClient(cfg.Guaji)

	targetRules := map[string]bool{"385": true, "271": true, "301": true, "307": true, "313": true}
	targetGames := map[int]bool{78: true, 79: true, 80: true}

	fmt.Println("=== 历史注单抓包（rule 385/271/301/307/313）===")
	found := 0
	for page := 1; page <= 10; page++ {
		_, raw, err := client.FetchWebBetsRaw(ctx, token, 50, page)
		if err != nil {
			fmt.Println("fetch err:", err)
			break
		}
		var wrap struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(raw, &wrap); err != nil {
			fmt.Println("decode err:", err)
			break
		}
		if len(wrap.Data) == 0 {
			break
		}
		for _, row := range wrap.Data {
			gid := int(intNum(row["game_id"]))
			if !targetGames[gid] {
				continue
			}
			contents, _ := row["bet_contents"].([]any)
			for _, bc := range contents {
				m, _ := bc.(map[string]any)
				if m == nil {
					continue
				}
				rid := strings.TrimSpace(fmt.Sprint(m["rule_id"]))
				if !targetRules[rid] {
					continue
				}
				found++
				fmt.Printf("id=%v game=%d rule=%s content=%q bets=%v solo=%v amount=%v\n",
					row["id"], gid, rid, m["bet_content"], m["bets_nums"], m["solo"], m["bet_amount"])
			}
		}
		if len(wrap.Data) < 50 {
			break
		}
	}
	if found == 0 {
		fmt.Println("(无匹配历史注单)")
	}
	_ = os.Exit
}

func intNum(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}
