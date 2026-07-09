package games

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemes"
	"caipiao/backend/internal/timeutil"
)

var (
	ErrUnavailable   = errors.New("games service unavailable")
	ErrInvalidQuery  = errors.New("invalid games query")
	ErrInvalidBet    = errors.New("invalid bet request")
	ErrLotteryNotFound = errors.New("lottery not found")
)

type Service struct {
	q         *sqlcdb.Queries
	pool      *db.Pool
	guajiBets GuajiBetPlacer
	detailSync *detailDisplaySync
}

func NewService(pool *db.Pool) *Service {
	if pool == nil {
		return nil
	}
	return &Service{q: sqlcdb.New(pool), pool: pool}
}

type BettingExecutionRow struct {
	Time    string `json:"time"`
	Scheme  string `json:"scheme"`
	Numbers string `json:"numbers"`
	Period  string `json:"period"`
	Draw    string `json:"draw"`
	Win     bool   `json:"win"`
}

type GameBetRecordRow struct {
	Period      string  `json:"period"`
	PlayMethod  string  `json:"playMethod"`
	Multiplier  string  `json:"multiplier"`
	Round       string  `json:"round"`
	Amount      string  `json:"amount"`
	ProfitLoss  float64 `json:"profitLoss"`
	Status      string  `json:"status"`
}

type PlanTrendRow struct {
	Period string `json:"period"`
	Win    bool   `json:"win"`
}

type PlanTrendChartPoint struct {
	Period string `json:"period"`
	Round  int    `json:"round"`
	Win    bool   `json:"win"`
}

type DetailQuery struct {
	LotteryCode string
	SchemeName  string
	PlayMethod  string
	SnapshotID  string
	Board       string
	PlayTypeID  string
	SubPlayID   string
}

type GameDetail struct {
	LotteryCode         string                `json:"lotteryCode"`
	LotteryLabel        string                `json:"lotteryLabel"`
	SchemeTitle         string                `json:"schemeTitle"`
	PlayMethod          string                `json:"playMethod"`
	CurrentIssue        string                `json:"currentIssue"`
	NextIssue           string                `json:"nextIssue"`
	CountdownSec         int                   `json:"countdownSec"`
	CountdownEndTime     string                `json:"countdownEndTime,omitempty"`
	CountdownCloseAt     string                `json:"countdownCloseAt,omitempty"`
	CountdownPeriod      string                `json:"countdownPeriod,omitempty"`
	CountdownWindowSec   int                   `json:"countdownWindowSec,omitempty"`
	CountdownLabel       string                `json:"countdownLabel,omitempty"`
	DrawPhase           string                `json:"drawPhase"`
	DrawnNumbers        []string              `json:"drawnNumbers"`
	PlanInverseDigits      string                `json:"planInverseDigits"`
	PlanInverseBetCount    int                   `json:"planInverseBetCount"`
	SchemeBetUnit          float64               `json:"schemeBetUnit,omitempty"`
	SchemeBetMultiplier    float64               `json:"schemeBetMultiplier,omitempty"`
	SchemeBetUnits         int                   `json:"schemeBetUnits,omitempty"`
	SchemeContraryBetUnits int                   `json:"schemeContraryBetUnits,omitempty"`
	SchemePickDigits       string                `json:"schemePickDigits,omitempty"`
	EstimatedPrize         float64               `json:"estimatedPrize,omitempty"`
	ContraryEstimatedPrize float64               `json:"contraryEstimatedPrize,omitempty"`
	BettingRows            []BettingExecutionRow `json:"bettingRows"`
	BetRecords          []GameBetRecordRow    `json:"betRecords"`
	PlanTrendGroupBets  int                   `json:"planTrendGroupBets"`
	PlanTrendHistory    []PlanTrendRow        `json:"planTrendHistory"`
	PlanTrendChart      []PlanTrendChartPoint `json:"planTrendChart"`
}

type DrawItem struct {
	PeriodShort string   `json:"periodShort"`
	Time        string   `json:"time"`
	Balls       []string `json:"balls"`
	Sum         int      `json:"sum"`
}

type DrawsQuery struct {
	LotteryCode string
	Cursor      string
	Limit       int
}

type DrawsResult struct {
	Items []DrawItem `json:"items"`
	Page  PageMeta   `json:"page"`
}

type PageMeta struct {
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

type PlaceBetInput struct {
	IssueNo    string
	Amount     float64
	Multiplier int
	BetMode    string
	PlayMethod string
	RunMode    string // real（默认，走第三方）/ sim（本地，不调第三方）
	BetPayload schemes.BetPayload
}

type PlaceBetResult struct {
	OrderNo         string  `json:"orderNo"`
	IssueNo         string  `json:"issueNo"`
	Amount          float64 `json:"amount"`
	Status          string  `json:"status"`
	PlacedAt        string  `json:"placedAt"`
	ThirdPartyBetID string  `json:"thirdPartyBetId,omitempty"`
}

func (s *Service) Detail(ctx context.Context, q DetailQuery) (GameDetail, error) {
	if s == nil || s.q == nil {
		return GameDetail{}, ErrUnavailable
	}
	code := strings.TrimSpace(q.LotteryCode)
	if code == "" {
		return GameDetail{}, ErrInvalidQuery
	}
	cat, catErr := s.q.GetLotteryCatalogByCode(ctx, code)
	if catErr == nil && sqlcdb.SaleStatusString(cat.SaleStatus) == "maintenance" {
		return GameDetail{}, ErrLotteryMaintenance
	}
	label := s.catalogLabel(ctx, code)
	catRow := sqlcdb.LotteryCatalogRowFromByCode(cat)
	lhc := isCatalogLHC(catRow, catErr)

	schemeName := strings.TrimSpace(q.SchemeName)
	if schemeName == "" {
		schemeName = defaultDemoSchemeName(lhc)
	}
	playMethod := strings.TrimSpace(q.PlayMethod)
	if playMethod == "" {
		playMethod = defaultDemoPlayMethod(lhc)
	}

	s.ensureDetailDisplayFresh(ctx, code)

	period, err := s.resolvePeriodDisplay(ctx, code, lhc)
	if err != nil {
		return GameDetail{}, err
	}
	currentIssue := period.CurrentIssue
	nextIssue := period.NextIssue
	drawPhase := period.DrawPhase
	drawnNumbers := period.DrawnNumbers
	countdownSec := period.CountdownSec
	countdownEndTime := period.CountdownEndTime
	countdownCloseAt := period.CountdownCloseAt
	countdownPeriod := period.CountdownPeriod
	countdownWindowSec := period.CountdownWindowSec
	countdownLabel := period.CountdownLabel

	title := schemeName
	if playMethod != "" && !strings.Contains(title, playMethod) {
		title = fmt.Sprintf("%s - %s", schemeName, playMethod)
	}

	bettingRows, err := s.loadBettingRows(ctx, q, countdownPeriod, currentIssue)
	if err != nil {
		return GameDetail{}, err
	}

	planInverseDigits, planInverseBetCount := s.loadPlanInverse(ctx, q, lhc, countdownPeriod, currentIssue)

	previewExtras, err := s.loadDetailPreviewExtras(ctx, q, countdownPeriod, currentIssue, playMethod)
	if err != nil {
		return GameDetail{}, err
	}

	betRecords := previewExtras.BetRecords
	planTrendGroupBets := previewExtras.PlanTrendGroupBets
	planTrendHistory := previewExtras.PlanTrendHistory
	planTrendChart := previewExtras.PlanTrendChart
	if strings.TrimSpace(q.SnapshotID) == "" {
		betRecords = demoBetRecords(playMethod, lhc)
		planTrendGroupBets = 7
		planTrendHistory = demoPlanTrendHistory()
		planTrendChart = nil
	}

	schemeDock := s.loadSchemeDock(ctx, q, countdownPeriod, currentIssue)

	return GameDetail{
		LotteryCode:            code,
		LotteryLabel:           label,
		SchemeTitle:            title,
		PlayMethod:             playMethod,
		CurrentIssue:           currentIssue,
		NextIssue:              nextIssue,
		CountdownSec:           countdownSec,
		CountdownEndTime:       countdownEndTime,
		CountdownCloseAt:       countdownCloseAt,
		CountdownPeriod:        countdownPeriod,
		CountdownWindowSec:     countdownWindowSec,
		CountdownLabel:         countdownLabel,
		DrawPhase:              drawPhase,
		DrawnNumbers:           drawnNumbers,
		PlanInverseDigits:      planInverseDigits,
		PlanInverseBetCount:    planInverseBetCount,
		SchemeBetUnit:          schemeDock.BetUnitYuan,
		SchemeBetMultiplier:    schemeDock.BetMultiplier,
		SchemeBetUnits:         schemeDock.PlanBetUnits,
		SchemeContraryBetUnits: schemeDock.ContraryBetUnits,
		SchemePickDigits:       schemeDock.SchemePickDisplay,
		EstimatedPrize:         schemeDock.EstimatedPrize,
		ContraryEstimatedPrize: schemeDock.ContraryEstimatedPrize,
		BettingRows:            bettingRows,
		BetRecords:             betRecords,
		PlanTrendGroupBets:     planTrendGroupBets,
		PlanTrendHistory:       planTrendHistory,
		PlanTrendChart:         planTrendChart,
	}, nil
}

func (s *Service) Draws(ctx context.Context, q DrawsQuery) (DrawsResult, error) {
	if s == nil || s.q == nil {
		return DrawsResult{}, ErrUnavailable
	}
	code := strings.TrimSpace(q.LotteryCode)
	if code == "" {
		return DrawsResult{}, ErrInvalidQuery
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var rows []sqlcdb.ListLotteryDrawsRow
	var err error
	if q.Cursor == "" {
		rows, err = s.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
			LotteryCode: code,
			RowLimit:    int32(limit + 1),
		})
	} else {
		cursorID, cursorTime, parseErr := parseDrawCursorFromDB(ctx, s.q, code, q.Cursor)
		if parseErr != nil {
			return DrawsResult{}, parseErr
		}
		cursorRows, cursorErr := s.q.ListLotteryDrawsAfterCursor(ctx, sqlcdb.ListLotteryDrawsAfterCursorParams{
			LotteryCode: code,
			CursorTime:  cursorTime,
			CursorID:    cursorID,
			RowLimit:    int32(limit + 1),
		})
		if cursorErr != nil {
			return DrawsResult{}, cursorErr
		}
		rows = make([]sqlcdb.ListLotteryDrawsRow, 0, len(cursorRows))
		for _, row := range cursorRows {
			rows = append(rows, sqlcdb.ListLotteryDrawsRow{
				ID: row.ID, LotteryCode: row.LotteryCode, IssueNo: row.IssueNo,
				PeriodShort: row.PeriodShort, Balls: row.Balls, SumValue: row.SumValue, DrawnAt: row.DrawnAt,
			})
		}
		err = nil
	}
	if err != nil {
		return DrawsResult{}, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	items := make([]DrawItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapDrawItem(row))
	}

	nextCursor := ""
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = fmt.Sprintf("%d", last.ID)
	}

	return DrawsResult{Items: items, Page: PageMeta{NextCursor: nextCursor, HasMore: hasMore}}, nil
}

func (s *Service) PlaceBet(ctx context.Context, account, lotteryCode string, in PlaceBetInput) (PlaceBetResult, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return PlaceBetResult{}, ErrUnavailable
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return PlaceBetResult{}, ErrInvalidBet
	}
	if in.Amount <= 0 {
		return PlaceBetResult{}, fmt.Errorf("%w: amount 须大于 0", ErrInvalidBet)
	}
	if in.Multiplier <= 0 {
		return PlaceBetResult{}, fmt.Errorf("%w: multiplier 须大于 0", ErrInvalidBet)
	}
	issueNo := strings.TrimSpace(in.IssueNo)
	if issueNo == "" {
		latest, err := s.q.GetLatestLotteryDraw(ctx, lotteryCode)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				issueNo = "20231103033"
			} else {
				return PlaceBetResult{}, err
			}
		} else {
			issueNo = bumpIssueNo(latest.IssueNo)
		}
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PlaceBetResult{}, member.ErrNotFound
		}
		return PlaceBetResult{}, err
	}

	cat, err := s.q.GetLotteryCatalogByCode(ctx, lotteryCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PlaceBetResult{}, ErrLotteryNotFound
		}
		return PlaceBetResult{}, err
	}
	if cat.SaleStatus == "maintenance" {
		return PlaceBetResult{}, ErrLotteryMaintenance
	}
	if cat.SaleStatus != "on_sale" {
		return PlaceBetResult{}, ErrLotteryNotFound
	}
	label := cat.DisplayName
	if label == "" {
		label = lotteryCode
	}
	template := textVal(cat.PlayTemplate)
	if template == "" {
		return PlaceBetResult{}, ErrLotteryNotFound
	}

	amount := member.RoundMoney(in.Amount)
	orderNo := fmt.Sprintf("BO%d%d", m.ID, time.Now().UnixMilli())
	in.BetPayload.PlayMethod = strings.TrimSpace(in.PlayMethod)
	if in.BetPayload.PlayMethod == "" {
		in.BetPayload.PlayMethod = "定位胆万位"
	}
	if in.BetPayload.PlayTemplate == "" {
		in.BetPayload.PlayTemplate = template
	}
	if in.BetPayload.TypeID == "" && in.BetPayload.PlayTypeID != "" {
		in.BetPayload.TypeID = in.BetPayload.PlayTypeID
	}
	if in.BetPayload.SubID == "" && in.BetPayload.SubPlayID != "" {
		in.BetPayload.SubID = in.BetPayload.SubPlayID
	}

	var outboundPlayCode pgtype.Text
	var subPlay sqlcdb.GetSubPlayRow
	var hasSubPlay bool
	if in.BetPayload.TypeID != "" && in.BetPayload.SubID != "" {
		sub, subErr := s.q.GetSubPlay(ctx, sqlcdb.GetSubPlayParams{
			TemplateCode: template,
			TypeID:       in.BetPayload.TypeID,
			SubID:        in.BetPayload.SubID,
		})
		if subErr != nil {
			if errors.Is(subErr, pgx.ErrNoRows) {
				return PlaceBetResult{}, fmt.Errorf("%w: 玩法不存在", ErrInvalidBet)
			}
			return PlaceBetResult{}, subErr
		}
		subPlay = sub
		hasSubPlay = true
		if code := textVal(sub.OutboundPlayCode); code != "" {
			outboundPlayCode = pgtype.Text{String: code, Valid: true}
		}
		in.BetPayload.BetMode = textVal(sub.BetMode)
	}

	payload, err := schemes.NormalizeBetPayload(in.BetPayload)
	if err != nil {
		return PlaceBetResult{}, fmt.Errorf("%w: %v", ErrInvalidBet, err)
	}
	playMethod := pgtype.Text{String: in.BetPayload.PlayMethod, Valid: true}
	lotteryCategory := categoryCodeToBetCategory(textVal(cat.CategoryCode))
	outboundLotteryCode := pgtype.Text{String: textVal(cat.OutboundLotteryCode), Valid: textVal(cat.OutboundLotteryCode) != ""}

	// T4：real 模式且第三方已启用 → 走 web_bets/lott 接单，不扣本地钱包；
	// sim 或第三方未启用（开发降级）→ 本地钱包扣款（保持现状）。
	useThirdParty := !strings.EqualFold(strings.TrimSpace(in.RunMode), "sim") && s.guajiRealEnabled()

	var guajiAccountID pgtype.Int8
	var thirdPartyBetID pgtype.Text
	var currency pgtype.Text

	if useThirdParty {
		gameID := outboundLotteryCode.String
		if gameID == "" {
			gameID = lotteryCode
		}
		ruleID := outboundPlayCode.String
		if ruleID == "" && in.BetPayload.TypeID != "" && in.BetPayload.SubID != "" {
			ruleID = fmt.Sprintf("%s:%s:%s", template, in.BetPayload.TypeID, in.BetPayload.SubID)
		}
		subLabel := strings.TrimSpace(in.BetPayload.PlayMethod)
		var segmentRule []byte
		if hasSubPlay {
			subLabel = strings.TrimSpace(subPlay.Label)
			if subLabel == "" {
				subLabel = in.BetPayload.PlayMethod
			}
			segmentRule = subPlay.SegmentRule
			if code := guajibet.ExtractGuajiRuleID(textVal(subPlay.OutboundPlayCode), segmentRule, subPlay.SubID); code != "" {
				ruleID = code
			}
		}
		ruleMeta := guajibet.ParseRuleMeta(
			template,
			in.BetPayload.TypeID,
			in.BetPayload.SubID,
			subLabel,
			"",
			segmentRule,
			ruleID,
		)
		guajiContent := guajibet.FormatBetContentForRule(ruleMeta, in.BetPayload.GroupContent)
		amountUnit := 2.0
		betsNums := guajibet.ResolveBetsNums(ruleMeta, guajiContent, amount, amountUnit, in.Multiplier)
		betRes, betErr := s.guajiBets.PlaceRealBet(ctx, account, GuajiBetRequest{
			LotteryCode: lotteryCode,
			GameID:     gameID,
			RuleID:     ruleID,
			IssueNo:    issueNo,
			Content:    guajiContent,
			PlayMethod: in.BetPayload.PlayMethod,
			Amount:     amount,
			Multiplier: in.Multiplier,
			AmountUnit: amountUnit,
			BetsNums:   betsNums,
			RuleMeta:   ruleMeta,
		})
		if betErr != nil {
			return PlaceBetResult{}, betErr
		}
		guajiAccountID = pgtype.Int8{Int64: betRes.GuajiAccountID, Valid: betRes.GuajiAccountID != 0}
		thirdPartyBetID = pgtype.Text{String: betRes.ThirdPartyBetID, Valid: betRes.ThirdPartyBetID != ""}
		currency = pgtype.Text{String: betRes.Currency, Valid: betRes.Currency != ""}
		if p := strings.TrimSpace(betRes.Periods); p != "" {
			issueNo = p
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PlaceBetResult{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	// real 第三方接单成功后不扣本地钱包（实账在第三方，C8）；sim/降级仍扣本地。
	if !useThirdParty {
		if err := member.DebitWalletForBet(ctx, qtx, m.ID, orderNo, amount); err != nil {
			return PlaceBetResult{}, err
		}
	}

	row, err := qtx.InsertBetOrder(ctx, sqlcdb.InsertBetOrderParams{
		OrderNo:             orderNo,
		MemberID:            m.ID,
		LotteryCode:         lotteryCode,
		LotteryName:         label,
		LotteryCategory:     lotteryCategory,
		IssueNo:             issueNo,
		Amount:              member.NumericFromFloat(amount),
		PlayMethod:          playMethod,
		BetPayload:          payload,
		OutboundLotteryCode: outboundLotteryCode,
		OutboundPlayCode:    outboundPlayCode,
		GuajiAccountID:      guajiAccountID,
		ThirdPartyBetID:     thirdPartyBetID,
		Currency:            currency,
	})
	if err != nil {
		return PlaceBetResult{}, err
	}

	if useThirdParty && guajiAccountID.Valid && s.guajiBets != nil {
		cur := strings.TrimSpace(currency.String)
		if !currency.Valid || cur == "" {
			cur = "CNY"
		}
		if err := s.guajiBets.MirrorBetDebitLedger(ctx, qtx, m.ID, orderNo, amount, guajiAccountID.Int64, cur); err != nil {
			return PlaceBetResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return PlaceBetResult{}, err
	}

	tpBetID := ""
	if thirdPartyBetID.Valid {
		tpBetID = strings.TrimSpace(thirdPartyBetID.String)
	}
	return PlaceBetResult{
		OrderNo:         row.OrderNo,
		IssueNo:         row.IssueNo,
		Amount:          row.Amount,
		Status:          row.Status,
		PlacedAt:        timeutil.FormatISO(row.PlacedAt.Time),
		ThirdPartyBetID: tpBetID,
	}, nil
}

func mapDrawItem(row sqlcdb.ListLotteryDrawsRow) DrawItem {
	return DrawItem{
		PeriodShort: schemes.ThirdPartyPeriodDisplay(row.IssueNo),
		Time:        row.DrawnAt.Time.Format("2006-01-02 15:04:05"),
		Balls:       parseBalls(row.Balls),
		Sum:         int(row.SumValue),
	}
}

func parseBalls(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var balls []string
	if err := json.Unmarshal(raw, &balls); err != nil {
		return []string{}
	}
	return balls
}

func parseDrawCursorFromDB(ctx context.Context, q *sqlcdb.Queries, lotteryCode string, cursor string) (int64, pgtype.Timestamptz, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(cursor), 10, 64)
	if err != nil || id <= 0 {
		return 0, pgtype.Timestamptz{}, ErrInvalidQuery
	}
	row, err := q.GetLotteryDrawByID(ctx, sqlcdb.GetLotteryDrawByIDParams{ID: id, LotteryCode: lotteryCode})
	if err != nil {
		return 0, pgtype.Timestamptz{}, ErrInvalidQuery
	}
	return row.ID, row.DrawnAt, nil
}

func isCatalogLHC(cat sqlcdb.LotteryCatalogRow, err error) bool {
	if err == nil {
		if textVal(cat.CategoryCode) == "lhc" || textVal(cat.PlayTemplate) == "lhc_std" {
			return true
		}
	}
	return false
}

func defaultDemoSchemeName(lhc bool) string {
	if lhc {
		return "特码方案"
	}
	return "禄螭万位"
}

func defaultDemoPlayMethod(lhc bool) string {
	if lhc {
		return "特码A"
	}
	return "定位胆万位"
}

func defaultDemoCurrentIssue(lhc bool) string {
	if lhc {
		return "20260608031"
	}
	return "20231103032"
}

func defaultPlanInverseDigits(lhc bool) string {
	if lhc {
		return "01,13,25"
	}
	return "xxx,xxx,xxx,xxx,xxx"
}

func defaultPlanInverseBetCount(lhc bool) int {
	if lhc {
		return 3
	}
	return 3
}

func defaultDemoDrawnBalls(cat sqlcdb.LotteryCatalogRow, err error) []string {
	if err == nil {
		template := textVal(cat.PlayTemplate)
		category := textVal(cat.CategoryCode)
		if category == "lhc" || template == "lhc_std" {
			return []string{"03", "12", "25", "33", "41", "07", "49"}
		}
		switch template {
		case "syxw_std":
			return []string{"01", "04", "06", "08", "11"}
		case "pk10_std":
			return []string{"3", "7", "1", "9", "5", "2", "8", "4", "6", "10"}
		case "k3_std":
			return []string{"2", "4", "6"}
		case "pc28_std":
			return []string{"3", "5", "7"}
		}
		if cat.BallCount.Valid {
			switch cat.BallCount.Int16 {
			case 7:
				return []string{"03", "12", "25", "33", "41", "07", "49"}
			case 10:
				return []string{"3", "7", "1", "9", "5", "2", "8", "4", "6", "10"}
			case 5:
				if category == "syxw" {
					return []string{"01", "04", "06", "08", "11"}
				}
			case 3:
				if category == "k3" {
					return []string{"2", "4", "6"}
				}
				if category == "pc28" {
					return []string{"3", "5", "7"}
				}
			}
		}
	}
	return []string{"0", "1", "9", "2", "3"}
}

func (s *Service) catalogLabel(ctx context.Context, code string) string {
	if s == nil || s.q == nil {
		return code
	}
	row, err := s.q.GetLotteryCatalogByCode(ctx, code)
	if err != nil || row.DisplayName == "" {
		return code
	}
	return row.DisplayName
}

func bumpIssueNo(issue string) string {
	n, err := strconv.ParseInt(strings.TrimSpace(issue), 10, 64)
	if err != nil {
		return issue + "1"
	}
	return strconv.FormatInt(n+1, 10)
}

func demoBettingRows(schemeName string, lhc bool) []BettingExecutionRow {
	if lhc {
		return []BettingExecutionRow{
			{Time: "031-032", Scheme: schemeName, Numbers: "01 13 49", Period: "031", Draw: "03 12 25 33 41 07 49", Win: true},
			{Time: "030-031", Scheme: schemeName, Numbers: "07 18 24", Period: "030", Draw: "07 18 27 34 40 13 45", Win: false},
			{Time: "029-030", Scheme: schemeName, Numbers: "01 16 48", Period: "029", Draw: "01 16 24 35 42 09 48", Win: true},
			{Time: "028-029", Scheme: schemeName, Numbers: "05 14 46", Period: "028", Draw: "05 14 22 31 38 11 46", Win: true},
			{Time: "027-028", Scheme: schemeName, Numbers: "08 19 26", Period: "027", Draw: "08 19 26 33 41 02 49", Win: false},
		}
	}
	return []BettingExecutionRow{
		{Time: "031-032", Scheme: schemeName, Numbers: "1 3 7", Period: "031", Draw: "1 6 5 8 3", Win: true},
		{Time: "030-031", Scheme: schemeName, Numbers: "4 5 9", Period: "030", Draw: "2 4 9 1 5", Win: false},
		{Time: "029-030", Scheme: schemeName, Numbers: "2 8 9", Period: "029", Draw: "3 7 8 4 9", Win: true},
		{Time: "028-029", Scheme: schemeName, Numbers: "1 2 5", Period: "028", Draw: "0 1 7 2 1", Win: true},
		{Time: "027-028", Scheme: schemeName, Numbers: "3 6 0", Period: "027", Draw: "2 8 3 5 5", Win: false},
	}
}

func demoBetRecords(playMethod string, lhc bool) []GameBetRecordRow {
	if lhc {
		return []GameBetRecordRow{
			{Period: "20260608032", PlayMethod: playMethod, Multiplier: "2", Round: "1", Amount: "12.00", ProfitLoss: 88.5, Status: "已结算"},
			{Period: "20260608031", PlayMethod: playMethod, Multiplier: "1", Round: "2", Amount: "6.00", ProfitLoss: -6, Status: "已结算"},
			{Period: "20260608033", PlayMethod: playMethod, Multiplier: "5", Round: "1", Amount: "30.00", ProfitLoss: 0, Status: "待开奖"},
		}
	}
	return []GameBetRecordRow{
		{Period: "20231103032", PlayMethod: playMethod, Multiplier: "2", Round: "1", Amount: "12.00", ProfitLoss: 88.5, Status: "已结算"},
		{Period: "20231103031", PlayMethod: playMethod, Multiplier: "1", Round: "2", Amount: "6.00", ProfitLoss: -6, Status: "已结算"},
		{Period: "20231103033", PlayMethod: playMethod, Multiplier: "5", Round: "1", Amount: "30.00", ProfitLoss: 0, Status: "待开奖"},
	}
}

func demoPlanTrendHistory() []PlanTrendRow {
	return []PlanTrendRow{
		{Period: "032", Win: false},
		{Period: "029 - 031", Win: true},
		{Period: "028", Win: false},
		{Period: "025 - 027", Win: true},
		{Period: "024", Win: false},
		{Period: "021 - 023", Win: true},
	}
}
