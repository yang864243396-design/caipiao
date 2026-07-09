// 逐彩种×玩法真实下单矩阵测试。
// go run ./cmd/real-bet-matrix [-dry-run] [-limit N] [-resume N] [-lottery code] [-delay 3s]
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/games"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/guaji/periodsync"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemes"
)

type matrixRow struct {
	Index              int
	LotteryCode        string
	DisplayName        string
	PlayTemplate       string
	OutboundLottery    string
	TypeID             string
	SubID              string
	Label              string
	TypeLabel          string
	BetMode            string
	OutboundPlayCode   string
	SegmentRule        []byte
	GuajiRuleID        string
}

type runResult struct {
	Index           int     `json:"index"`
	LotteryCode     string  `json:"lotteryCode"`
	TypeID          string  `json:"typeId"`
	SubID           string  `json:"subId"`
	Label           string  `json:"label"`
	RuleID          string  `json:"ruleId"`
	Status          string  `json:"status"` // ok | skip | fail
	OrderNo         string  `json:"orderNo,omitempty"`
	IssueNo         string  `json:"issueNo,omitempty"`
	ThirdPartyBetID string  `json:"thirdPartyBetId,omitempty"`
	Amount          float64 `json:"amount,omitempty"`
	VerifyStatus    string  `json:"verifyStatus,omitempty"` // ok | mismatch | not_found | skipped
	VerifyDetail    string  `json:"verifyDetail,omitempty"`
	PeriodWaits     int     `json:"periodWaits,omitempty"`
	Error           string  `json:"error,omitempty"`
	At              string  `json:"at"`
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	dryRun := flag.Bool("dry-run", false, "只统计矩阵，不下单")
	limit := flag.Int("limit", 0, "最多测试条数，0=全部")
	resume := flag.Int("resume", 0, "从矩阵 index 继续")
	lotteryFilter := flag.String("lottery", "", "只测指定彩种 code")
	typeFilter := flag.String("type", "", "只测指定 type_id")
	subFilter := flag.String("sub", "", "只测指定 sub_id")
	amountUnit := flag.Float64("unit", 2, "单注金额（元）")
	delay := flag.Duration("delay", 3*time.Second, "每单间隔")
	maxPeriodWait := flag.Duration("max-period-wait", 5*time.Minute, "单注等待开盘/下一期最长时间")
	verifyBets := flag.Bool("verify", true, "成功后与第三方 web_bets 对账")
	account := flag.String("account", "", "会员账号，默认 CLIENT_DEMO_ACCOUNT")
	outPath := flag.String("out", "data/real-bet-matrix-report.jsonl", "结果 JSONL 路径")
	truncate := flag.Bool("truncate", false, "写入前清空 out 文件（按彩种分批跑时使用）")
	indexSpec := flag.String("indices", "", "只跑指定 index，如 89-101,128-140")
	patchOut := flag.Bool("patch", false, "将结果按 index 合并覆盖 out 已有行（与 -indices 联用）")
	flag.Parse()

	if *account == "" {
		*account = cfg.ClientDemoAccount
	}
	if *account == "" {
		*account = "vs8888"
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}
	defer pool.Close()

	rows, err := loadMatrix(ctx, pool, *lotteryFilter, *typeFilter, *subFilter)
	if err != nil {
		fmt.Println("matrix:", err)
		os.Exit(1)
	}
	if len(rows) == 0 {
		fmt.Println("no matrix rows (check sale_status / sub_plays.enabled / rule_id)")
		os.Exit(1)
	}

	fmt.Printf("matrix rows=%d account=%s unit=%.0f est_min_cost=%.0f\n",
		len(rows), *account, *amountUnit, *amountUnit*float64(len(rows)))

	if *dryRun {
		byLottery := map[string]int{}
		for _, r := range rows {
			byLottery[r.LotteryCode]++
		}
		fmt.Printf("lotteries=%d templates covered\n", len(byLottery))
		for code, n := range byLottery {
			fmt.Printf("  %s: %d plays\n", code, n)
		}
		return
	}

	if !cfg.Guaji.Enabled {
		fmt.Println("GUAJI_ENABLED=false，无法真实下单；请在 backend/.env 开启")
		os.Exit(1)
	}

	guajiClient := guaji.NewClient(cfg.Guaji)
	guajiAccounts := accountsvc.NewService(pool, guajiClient, cfg.Guaji.CredentialsKey, cfg.JWTSecret)
	if !guajiAccounts.Enabled() {
		fmt.Println("guaji account service not enabled")
		os.Exit(1)
	}
	periodSync := periodsync.NewSyncer(pool, guajiClient, guajiAccounts)

	gamesSvc := games.NewService(pool)
	gamesSvc.SetGuajiBetPlacer(guajiAccounts)

	if *lotteryFilter != "" && periodSync != nil {
		if err := periodSync.ForceRefreshForMember(ctx, *lotteryFilter, *account); err != nil {
			fmt.Printf("warn: force refresh periods lottery=%s err=%v\n", *lotteryFilter, err)
		}
	}

	indexSet, err := parseIndexRanges(*indexSpec)
	if err != nil {
		fmt.Println("indices:", err)
		os.Exit(1)
	}
	if *patchOut && len(indexSet) == 0 {
		fmt.Println("patch requires -indices")
		os.Exit(1)
	}
	if *patchOut && *truncate {
		fmt.Println("patch cannot be used with -truncate")
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Println("mkdir:", err)
		os.Exit(1)
	}
	flags := os.O_CREATE | os.O_APPEND | os.O_WRONLY
	if *truncate {
		flags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	}
	var report *os.File
	var enc *json.Encoder
	updates := map[int]runResult{}
	if *patchOut {
		// 重跑段：结果先缓存在内存，结束后合并写回 out。
	} else {
		report, err = os.OpenFile(*outPath, flags, 0o644)
		if err != nil {
			fmt.Println("report:", err)
			os.Exit(1)
		}
		defer report.Close()
		enc = json.NewEncoder(report)
	}

	tested := 0
	okCount, skipCount, failCount := 0, 0, 0

	for _, row := range rows {
		if !indexAllowed(indexSet, row.Index) {
			continue
		}
		if len(indexSet) == 0 && row.Index < *resume {
			continue
		}
		if len(indexSet) == 0 && *limit > 0 && tested >= *limit {
			break
		}

		res := runOne(ctx, runOneParams{
			gamesSvc:      gamesSvc,
			periodSync:    periodSync,
			guajiClient:   guajiClient,
			guajiAccounts: guajiAccounts,
			account:       *account,
			row:           row,
			amountUnit:    *amountUnit,
			maxPeriodWait: *maxPeriodWait,
			verify:        *verifyBets,
		})
		if *patchOut {
			updates[row.Index] = res
		} else {
			_ = enc.Encode(res)
			_, _ = report.WriteString("\n")
		}

		switch res.Status {
		case "ok":
			okCount++
			fmt.Printf("[%d] OK %s %s/%s order=%s issue=%s tp=%s verify=%s\n",
				row.Index, row.LotteryCode, row.TypeID, row.SubID, res.OrderNo, res.IssueNo, res.ThirdPartyBetID, res.VerifyStatus)
		case "skip":
			skipCount++
			fmt.Printf("[%d] SKIP %s %s/%s: %s\n", row.Index, row.LotteryCode, row.TypeID, row.SubID, res.Error)
		default:
			failCount++
			fmt.Printf("[%d] FAIL %s %s/%s: %s\n", row.Index, row.LotteryCode, row.TypeID, row.SubID, res.Error)
		}
		tested++
		if *delay > 0 && tested < len(rows) {
			time.Sleep(*delay)
		}
	}

	if *patchOut {
		if err := patchResultsJSONL(*outPath, updates); err != nil {
			fmt.Println("patch:", err)
			os.Exit(1)
		}
		fmt.Printf("patched %d rows into %s\n", len(updates), *outPath)
	}

	fmt.Printf("\ndone tested=%d ok=%d skip=%d fail=%d report=%s\n", tested, okCount, skipCount, failCount, *outPath)
	if failCount > 0 {
		os.Exit(2)
	}
}

func loadMatrix(ctx context.Context, pool *db.Pool, lotteryFilter, typeFilter, subFilter string) ([]matrixRow, error) {
	q := `
SELECT lc.code, lc.display_name, lc.play_template,
       COALESCE(NULLIF(TRIM(lc.outbound_lottery_code), ''), lc.code),
       sp.type_id, sp.sub_id, sp.label, COALESCE(sp.bet_mode, ''),
       COALESCE(sp.outbound_play_code, ''), sp.segment_rule,
       COALESCE(pt.label, '')
FROM lottery_catalog lc
JOIN sub_plays sp ON sp.template_code = lc.play_template
LEFT JOIN play_types pt ON pt.template_code = sp.template_code AND pt.type_id = sp.type_id
WHERE lc.sale_status = 'on_sale' AND sp.enabled = true`
	args := []any{}
	argN := 1
	if lotteryFilter != "" {
		q += fmt.Sprintf(` AND lc.code = $%d`, argN)
		args = append(args, lotteryFilter)
		argN++
	}
	if typeFilter != "" {
		q += fmt.Sprintf(` AND sp.type_id = $%d`, argN)
		args = append(args, typeFilter)
		argN++
	}
	if subFilter != "" {
		q += fmt.Sprintf(` AND sp.sub_id = $%d`, argN)
		args = append(args, subFilter)
		argN++
	}
	q += ` ORDER BY lc.sort_order, lc.code, sp.type_id, sp.sort_order, sp.sub_id`

	dbRows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer dbRows.Close()

	var out []matrixRow
	idx := 0
	for dbRows.Next() {
		var r matrixRow
		var seg pgtype.Text
		if err := dbRows.Scan(&r.LotteryCode, &r.DisplayName, &r.PlayTemplate, &r.OutboundLottery,
			&r.TypeID, &r.SubID, &r.Label, &r.BetMode, &r.OutboundPlayCode, &seg, &r.TypeLabel); err != nil {
			return nil, err
		}
		if seg.Valid {
			r.SegmentRule = []byte(seg.String)
		}
		r.GuajiRuleID = guajibet.ExtractGuajiRuleID(r.OutboundPlayCode, r.SegmentRule, r.SubID)
		if !guajibet.IsNumericGuajiRuleID(r.GuajiRuleID) {
			continue
		}
		r.Index = idx
		idx++
		out = append(out, r)
	}
	return out, dbRows.Err()
}

type runOneParams struct {
	gamesSvc      *games.Service
	periodSync    *periodsync.Syncer
	guajiClient   *guaji.Client
	guajiAccounts *accountsvc.Service
	account       string
	row           matrixRow
	amountUnit    float64
	maxPeriodWait time.Duration
	verify        bool
}

func runOne(ctx context.Context, p runOneParams) runResult {
	row := p.row
	res := runResult{
		Index:       row.Index,
		LotteryCode: row.LotteryCode,
		TypeID:      row.TypeID,
		SubID:       row.SubID,
		Label:       row.Label,
		RuleID:      row.GuajiRuleID,
		At:          time.Now().Format(time.RFC3339),
	}

	meta := guajibet.ParseRuleMeta(
		row.PlayTemplate, row.TypeID, row.SubID, row.Label, row.TypeLabel,
		row.SegmentRule, row.GuajiRuleID,
	)
	if reason := guajibet.MatrixSkipReason(meta); reason != "" {
		res.Status = "skip"
		res.Error = reason
		return res
	}
	content := guajibet.SampleGroupContent(meta)
	wire := guajibet.FormatBetContentForRule(meta, content)
	if err := validateMatrixPayload(row, content); err != nil {
		res.Status = "skip"
		res.Error = err.Error()
		return res
	}
	betsNums := guajibet.ResolveBetsNums(meta, wire, 0, p.amountUnit, 1)
	if betsNums <= 0 {
		betsNums = 1
	}
	amount := p.amountUnit * float64(betsNums)
	res.Amount = amount

	playMethod := strings.TrimSpace(row.Label)
	if playMethod == "" {
		playMethod = row.TypeID + "/" + row.SubID
	}

	gameID := strings.TrimSpace(row.OutboundLottery)
	const maxPeriodAttempts = 24
	var lastClosedIssue string

	for attempt := 0; attempt < maxPeriodAttempts; attempt++ {
		issueNo, polls, err := waitForOpenIssue(ctx, p.periodSync, p.account, row.LotteryCode, lastClosedIssue, p.maxPeriodWait)
		res.PeriodWaits += polls
		if err != nil {
			res.Status = "fail"
			res.Error = err.Error()
			return res
		}

		result, err := p.gamesSvc.PlaceBet(ctx, p.account, row.LotteryCode, games.PlaceBetInput{
			IssueNo:    issueNo,
			Amount:     amount,
			Multiplier: 1,
			BetMode:    row.BetMode,
			PlayMethod: playMethod,
			RunMode:    "real",
			BetPayload: schemes.BetPayload{
				PlayTemplate: row.PlayTemplate,
				TypeID:       row.TypeID,
				SubID:        row.SubID,
				BetMode:      row.BetMode,
				PlayMethod:   playMethod,
				GroupContent: content,
			},
		})
		if err == nil {
			res.Status = "ok"
			res.OrderNo = result.OrderNo
			res.IssueNo = result.IssueNo
			res.ThirdPartyBetID = result.ThirdPartyBetID
			if p.verify {
				out := verifyThirdPartyBet(ctx, p.guajiClient, p.guajiAccounts, p.account,
					result.ThirdPartyBetID, gameID, result.IssueNo, row.GuajiRuleID, amount)
				res.VerifyStatus = out.Status
				res.VerifyDetail = out.Detail
				if out.Status == "mismatch" {
					res.Status = "fail"
					res.Error = "第三方对账不一致: " + out.Detail
				}
			}
			return res
		}
		if isTransientPeriodErr(err) {
			if issueNo != "" {
				lastClosedIssue = issueNo
			}
			continue
		}
		res.Status = classifyErr(err)
		res.Error = err.Error()
		return res
	}

	res.Status = "fail"
	res.Error = fmt.Sprintf("超过 %d 次期号重试仍未成功下单", maxPeriodAttempts)
	return res
}

func validateMatrixPayload(row matrixRow, content string) error {
	if _, err := schemes.NormalizeBetPayload(schemes.BetPayload{
		PlayTemplate: row.PlayTemplate,
		TypeID:       row.TypeID,
		SubID:        row.SubID,
		BetMode:      row.BetMode,
		PlayMethod:   row.Label,
		GroupContent: content,
	}); err != nil {
		return fmt.Errorf("normalize: %v", err)
	}
	return nil
}

func classifyErr(err error) string {
	switch {
	case errors.Is(err, guajibet.ErrPeriodClosed):
		return "skip"
	case strings.Contains(strings.ToLower(err.Error()), "封盘"):
		return "skip"
	case strings.Contains(strings.ToLower(err.Error()), "period"):
		return "skip"
	default:
		return "fail"
	}
}

// 供 CSV 导出调试（未在 main 使用，保留扩展）
func writeCSV(path string, rows []runResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"index", "lottery", "typeId", "subId", "status", "orderNo", "error"})
	for _, r := range rows {
		_ = w.Write([]string{
			fmt.Sprintf("%d", r.Index), r.LotteryCode, r.TypeID, r.SubID, r.Status, r.OrderNo, r.Error,
		})
	}
	w.Flush()
	return w.Error()
}
