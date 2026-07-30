// audit-period-family：逐彩种比对「下注链路的期号族」与「开奖链路的期号族」。
//
// 两者不同族 = 能下注、能结算，但期号永远查不到开奖。这类 bug 不报错、不影响
// 下单成功率，只有把两条链路摆在一起才看得见，已经出过两次：
//   - 00136 哈希/波场极速彩 game_id 对调
//   - 00138 币安极速飞艇 guaji_ws_key 抄了波场线（102 笔注单 101 笔 cancel）
//
// 只读：不下单、不写库。下注侧取 /api/web_bets/lott/periods（按 game_id），
// 开奖侧取本地 lottery_draws 最新一期（由 guaji_ws_key 决定写入）。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guaji"
)

type row struct {
	code, name, gameID, wsKey string
	drawIssue                 string
	drawAgeMin                int
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	account := flag.String("account", "vs8888", "取上游期号所用的会员账号（需已绑定挂机 token）")
	staleMin := flag.Int("stale-min", 30, "本地开奖落后多少分钟算异常")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Fprintln(os.Stderr, "数据库不可达:", err)
		os.Exit(1)
	}
	defer pool.Close()

	rows, err := loadLotteries(ctx, pool)
	if err != nil {
		fmt.Fprintln(os.Stderr, "读取彩种失败:", err)
		os.Exit(1)
	}

	token, err := loadToken(ctx, pool, cfg, *account)
	if err != nil {
		fmt.Fprintln(os.Stderr, "取挂机 token 失败:", err)
		os.Exit(1)
	}
	client := guaji.NewClient(cfg.Guaji)

	var problems []string
	fmt.Printf("%-18s %-7s %-22s %-18s %-18s %s\n",
		"彩种", "game_id", "开奖线键", "下注期号(上游)", "开奖期号(本地)", "判定")

	var sealed []row
	for _, r := range rows {
		betPeriod, verdict, problem := auditOne(ctx, client, token, r, *staleMin)
		if verdict == verdictSealed {
			// 上游封盘窗口取不到期号是常态（极速线各彩种窗口有 30 秒偏移），
			// 攒到最后统一重试，别把"正好赶上封盘"报成配置错。
			sealed = append(sealed, r)
			continue
		}
		fmt.Printf("%-18s %-7s %-22s %-18s %-18s %s\n",
			r.code, r.gameID, r.wsKey, dashIfEmpty(betPeriod), dashIfEmpty(r.drawIssue), verdict)
		if problem != "" {
			problems = append(problems, problem)
		}
	}

	if len(sealed) > 0 {
		fmt.Printf("\n%d 个彩种取期时正在封盘，等 %s 后重试 ...\n", len(sealed), sealRetryWait)
		select {
		case <-ctx.Done():
		case <-time.After(sealRetryWait):
		}
		for _, r := range sealed {
			betPeriod, verdict, problem := auditOne(ctx, client, token, r, *staleMin)
			if verdict == verdictSealed {
				verdict = "← 重试后仍封盘"
				problem = fmt.Sprintf("%s（game_id=%s）两次取期都在封盘，无法判定期号族", r.code, r.gameID)
			}
			fmt.Printf("%-18s %-7s %-22s %-18s %-18s %s\n",
				r.code, r.gameID, r.wsKey, dashIfEmpty(betPeriod), dashIfEmpty(r.drawIssue), verdict)
			if problem != "" {
				problems = append(problems, problem)
			}
		}
	}

	fmt.Println()
	if len(problems) == 0 {
		fmt.Println("未发现期号族不一致。")
		return
	}
	fmt.Printf("发现 %d 项异常：\n", len(problems))
	for _, p := range problems {
		fmt.Println("  -", p)
	}
	os.Exit(1)
}

// 上游封盘窗口（40055）取不到期号，各极速彩种窗口还有 30 秒偏移，
// 单次采样必然撞上几个，重试一轮才能区分"正好封盘"和"配置错"。
const (
	verdictSealed = "封盘中"
	sealRetryWait = 40 * time.Second
)

func isSealedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "40055") || strings.Contains(msg, "封盘")
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func auditOne(
	ctx context.Context, client *guaji.Client, token string, r row, staleMin int,
) (betPeriod, verdict, problem string) {
	if r.gameID == "" {
		return "", "← 未配置 game_id", fmt.Sprintf("%s 未配置 outbound_lottery_code，无法下单", r.code)
	}
	gameID := 0
	if _, err := fmt.Sscanf(r.gameID, "%d", &gameID); err != nil || gameID <= 0 {
		return "", "← game_id 非法", fmt.Sprintf("%s 的 outbound_lottery_code=%q 不是数字", r.code, r.gameID)
	}

	periods, _, err := client.FetchLottPeriods(ctx, token, gameID, 2)
	if err != nil && isSealedErr(err) {
		return "", verdictSealed, ""
	}
	if err != nil || len(periods) == 0 {
		return "", "← 上游无期号", fmt.Sprintf("%s（game_id=%s）上游取不到期号: %v", r.code, r.gameID, err)
	}
	bet := periods[0].Period

	switch {
	case r.drawIssue == "":
		return bet, "← 本地无开奖",
			fmt.Sprintf("%s 本地无开奖记录（guaji_ws_key=%s 可能没在收）", r.code, r.wsKey)
	case len(bet) != len(r.drawIssue):
		return bet, fmt.Sprintf("← 位数不同 %d vs %d", len(bet), len(r.drawIssue)),
			fmt.Sprintf("%s 期号族不一致：下注 %s（%d 位）vs 开奖 %s（%d 位），guaji_ws_key=%s 指向了别的彩种线",
				r.code, bet, len(bet), r.drawIssue, len(r.drawIssue), r.wsKey)
	case bet[:3] != r.drawIssue[:3]:
		return bet, fmt.Sprintf("← 前缀不同 %s vs %s", bet[:3], r.drawIssue[:3]),
			fmt.Sprintf("%s 期号族不一致：下注 %s vs 开奖 %s，guaji_ws_key=%s 指向了别的彩种线",
				r.code, bet, r.drawIssue, r.wsKey)
	case r.drawAgeMin > staleMin:
		return bet, fmt.Sprintf("← 开奖落后 %d 分", r.drawAgeMin),
			fmt.Sprintf("%s 期号族一致但开奖落后 %d 分钟，guaji_ws_key=%s 这条线可能断了",
				r.code, r.drawAgeMin, r.wsKey)
	}
	return bet, "ok", ""
}

func loadLotteries(ctx context.Context, pool *db.Pool) ([]row, error) {
	rs, err := pool.Query(ctx, `
SELECT lc.code, lc.display_name, COALESCE(lc.outbound_lottery_code, ''), COALESCE(lc.guaji_ws_key, ''),
       COALESCE(d.issue_no, ''),
       COALESCE(round(extract(epoch FROM (now() - d.drawn_at)) / 60)::int, -1)
FROM lottery_catalog lc
LEFT JOIN LATERAL (
    SELECT issue_no, drawn_at FROM lottery_draws
    WHERE lottery_code = lc.code ORDER BY drawn_at DESC LIMIT 1
) d ON true
WHERE lc.sale_status = 'on_sale'
ORDER BY lc.code`)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	var out []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.code, &r.name, &r.gameID, &r.wsKey, &r.drawIssue, &r.drawAgeMin); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rs.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].code < out[j].code })
	return out, nil
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
