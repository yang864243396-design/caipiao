package betrecords

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemes"
	"caipiao/backend/internal/timeutil"
)

const (
	itemDetailDays       = 3
	drawSyncMinGap       = 5 * time.Second
	drawSyncTimeout      = 2 * time.Second
	drawSyncPlacedWithin = 2 * time.Hour
)

var (
	ErrItemNotFound    = errors.New("bet record not found")
	ErrItemOutOfWindow = errors.New("bet record out of query window")
)

type HistorySyncer interface {
	SyncLottery(ctx context.Context, lotteryCode string) error
}

type Service struct {
	q           *sqlcdb.Queries
	historySync HistorySyncer
	drawSyncMu  sync.Map // lotteryCode -> time.Time
}

func NewService(pool *db.Pool) *Service {
	if pool == nil {
		return &Service{}
	}
	return &Service{q: sqlcdb.New(pool)}
}

func (s *Service) SetHistorySync(syncer HistorySyncer) {
	if s == nil {
		return
	}
	s.historySync = syncer
}

type GroupsFilter struct {
	Mode        string
	Days        int
	DateFrom    string
	DateTo      string
	LotteryCode string
	Cursor      string
	Limit       int
}

var ErrInvalidQuery = errors.New("invalid bet record query")

func (s *Service) Groups(ctx context.Context, memberID int64, mode Mode, days int) GroupsResult {
	got, _ := s.GroupsWithFilter(ctx, memberID, GroupsFilter{
		Mode:  string(mode),
		Days:  days,
		Limit: -1,
	})
	return got
}

func (s *Service) GroupsWithFilter(ctx context.Context, memberID int64, f GroupsFilter) (GroupsResult, error) {
	since, until, dateFrom, dateTo, days, err := resolveGroupsRange(f)
	if err != nil {
		return GroupsResult{}, err
	}
	limit := f.Limit
	if limit < 0 {
		limit = 0
	} else if limit == 0 {
		limit = 20
	} else if limit > 200 {
		limit = 200
	}
	if s.q != nil && memberID > 0 {
		return s.groupsWithFilterFromDB(ctx, memberID, f, since, until, dateFrom, dateTo, days, limit)
	}
	return s.groupsWithFilterFromMemory(f, since, until, dateFrom, dateTo, days, limit)
}

func (s *Service) groupsWithFilterFromDB(
	ctx context.Context,
	memberID int64,
	f GroupsFilter,
	since, until time.Time,
	dateFrom, dateTo string,
	days, limit int,
) (GroupsResult, error) {
	offset, err := parsePageOffset(f.Cursor, limit)
	if err != nil {
		return GroupsResult{}, err
	}
	filter, err := s.databaseFilter(ctx, memberID, f.Mode, f.LotteryCode, since, until)
	if err != nil {
		return GroupsResult{}, err
	}
	stats, err := s.q.SummarizeCloudBetRecordGroups(ctx, filter)
	if err != nil {
		return GroupsResult{}, err
	}
	fetchLimit := limit
	if fetchLimit <= 0 {
		fetchLimit = int(^uint32(0) >> 1)
	} else {
		fetchLimit++
	}
	rows, err := s.q.ListCloudBetRecordGroupsPage(ctx, filter, int32(offset), int32(fetchLimit))
	if err != nil {
		return GroupsResult{}, err
	}
	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	groups := make([]Group, len(rows))
	for i, row := range rows {
		summary := summaryFromAggregate(row.CloudBetAggregate)
		groups[i] = Group{
			SchemeID: row.SchemeID, SchemeName: row.SchemeName,
			TotalBet: summary.TotalBet, TotalPrize: summary.TotalPrize,
			DayPnL: summary.DayPnL, WinRate: summary.WinRate,
		}
	}
	var next *string
	if hasMore {
		value := strconv.Itoa(offset + limit)
		next = &value
	}
	mode := Mode(f.Mode)
	if f.Mode == "" {
		mode = ""
	}
	return GroupsResult{
		Mode:     mode,
		Days:     days,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Summary:  summaryFromAggregate(stats),
		Groups: GroupsPage{
			Items: groups,
			Page:  PageMeta{NextCursor: next, HasMore: hasMore},
		},
	}, nil
}

func (s *Service) groupsWithFilterFromMemory(
	f GroupsFilter,
	since, until time.Time,
	dateFrom, dateTo string,
	days, limit int,
) (GroupsResult, error) {
	rows := s.loadRowsFiltered(context.Background(), 0, f, since, until)
	page, err := paginateGroups(groupByScheme(rows), limit, f.Cursor)
	if err != nil {
		return GroupsResult{}, err
	}
	mode := Mode(f.Mode)
	if f.Mode == "" {
		mode = ""
	}
	return GroupsResult{
		Mode: mode, Days: days, DateFrom: dateFrom, DateTo: dateTo,
		Summary: summarize(rows), Groups: page,
	}, nil
}

func paginateGroups(groups []Group, limit int, cursor string) (GroupsPage, error) {
	if limit <= 0 {
		return GroupsPage{Items: groups, Page: PageMeta{HasMore: false}}, nil
	}
	offset := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 {
			return GroupsPage{}, fmt.Errorf("invalid cursor")
		}
		offset = n
	}
	if offset > len(groups) {
		offset = len(groups)
	}
	end := offset + limit
	if end > len(groups) {
		end = len(groups)
	}
	hasMore := end < len(groups)
	var next *string
	if hasMore {
		v := strconv.Itoa(end)
		next = &v
	}
	return GroupsPage{
		Items: groups[offset:end],
		Page:  PageMeta{NextCursor: next, HasMore: hasMore},
	}, nil
}

func (f GroupsFilter) Validate() error {
	if strings.TrimSpace(f.DateFrom) != "" || strings.TrimSpace(f.DateTo) != "" {
		return validateQuerySpan(f.DateFrom, f.DateTo)
	}
	return nil
}

func (s *Service) Detail(
	ctx context.Context,
	memberID int64,
	schemeID string,
	mode Mode,
	days, limit int,
	cursor string,
) (DetailResult, bool, error) {
	if days <= 0 {
		days = 3
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset, err := parsePageOffset(cursor, limit)
	if err != nil {
		return DetailResult{}, false, err
	}
	if s.q != nil && memberID > 0 {
		return s.detailFromDB(ctx, memberID, schemeID, mode, days, limit, offset)
	}
	return s.detailFromMemory(ctx, memberID, schemeID, mode, days, limit, offset)
}

func (s *Service) detailFromDB(
	ctx context.Context,
	memberID int64,
	schemeID string,
	mode Mode,
	days, limit, offset int,
) (DetailResult, bool, error) {
	dateFrom, dateTo, since, until := timeutil.NaturalDaysMeta(days)
	filter, err := s.databaseFilter(ctx, memberID, string(mode), "", since, until)
	if err != nil {
		return DetailResult{}, false, err
	}
	stats, err := s.q.SummarizeCloudBetRecordsByScheme(ctx, filter, schemeID)
	if err != nil {
		return DetailResult{}, false, err
	}
	if stats.TotalRows == 0 {
		return DetailResult{}, false, nil
	}
	rows, err := s.q.ListCloudBetRecordsBySchemePage(ctx, filter, schemeID, int32(offset), int32(limit+1))
	if err != nil {
		return DetailResult{}, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	var next *string
	if hasMore {
		value := strconv.Itoa(offset + limit)
		next = &value
	}
	return DetailResult{
		SchemeID: schemeID, SchemeName: stats.SchemeName,
		Mode: mode, Days: days, DateFrom: dateFrom, DateTo: dateTo,
		Summary: summaryFromAggregate(stats),
		Records: Page{
			Items: itemsFromFilteredRows(rows),
			Page:  PageMeta{NextCursor: next, HasMore: hasMore},
		},
	}, true, nil
}

func (s *Service) detailFromMemory(
	ctx context.Context,
	memberID int64,
	schemeID string,
	mode Mode,
	days, limit, offset int,
) (DetailResult, bool, error) {
	all := filterScheme(s.loadRows(ctx, memberID, mode, days), schemeID)
	if len(all) == 0 {
		return DetailResult{}, false, nil
	}
	if offset > len(all) {
		offset = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	slice := all[offset:end]
	items := make([]Item, len(slice))
	for i, r := range slice {
		displayPeriod := resolveBetRecordPeriods(r)
		items[i] = Item{
			ID:         thirdPartyBetOrderNo(r.ThirdPartyBetID),
			RecordNo:   strings.TrimSpace(r.ID),
			Period:     displayPeriod,
			Periods:    displayPeriod,
			PlayType:   r.PlayType,
			Multiplier: formatMultiplierDisplay(r.Multiplier),
			Round:      formatRoundDisplay(r.Round),
			Amount:     truncateBetAmount(r.Amount),
			PnL:        round2(r.PnL),
			Status:     r.Status,
			BetContent: r.BetContent,
		}
	}
	hasMore := end < len(all)
	var next *string
	if hasMore {
		v := strconv.Itoa(end)
		next = &v
	}
	dateFrom, dateTo, _, _ := timeutil.NaturalDaysMeta(days)
	return DetailResult{
		SchemeID:   schemeID,
		SchemeName: all[0].SchemeName,
		Mode:       mode,
		Days:       days,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Summary:    summarize(all),
		Records: Page{
			Items: items,
			Page:  PageMeta{NextCursor: next, HasMore: hasMore},
		},
	}, true, nil
}

func (s *Service) databaseFilter(
	ctx context.Context,
	memberID int64,
	mode, lotteryCode string,
	since, until time.Time,
) (sqlcdb.CloudBetRecordFilter, error) {
	filter := sqlcdb.CloudBetRecordFilter{
		MemberID: memberID,
		SinceAt:  pgtype.Timestamptz{Time: since, Valid: true},
		UntilAt:  pgtype.Timestamptz{Time: until, Valid: true},
	}
	if mode = strings.TrimSpace(mode); mode != "" {
		filter.SimBet = pgtype.Bool{Bool: mode == string(ModeSim), Valid: true}
	}
	if lotteryCode = strings.TrimSpace(lotteryCode); lotteryCode != "" {
		filter.LotteryCode = pgtype.Text{String: lotteryCode, Valid: true}
	}
	guajiID, err := member.LookupActiveGuajiAccountID(ctx, s.q, memberID)
	if err != nil {
		return sqlcdb.CloudBetRecordFilter{}, err
	}
	filter.GuajiAccountID = guajiID
	return filter, nil
}

func parsePageOffset(cursor string, limit int) (int, error) {
	if limit <= 0 || strings.TrimSpace(cursor) == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(cursor)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return offset, nil
}

func summaryFromAggregate(aggregate sqlcdb.CloudBetAggregate) Summary {
	winRate := 0.0
	if aggregate.TotalRows > 0 {
		winRate = round1(float64(aggregate.HitRows) / float64(aggregate.TotalRows) * 100)
	}
	return Summary{
		TotalBet: round2(aggregate.TotalBet), TotalPrize: round2(aggregate.TotalPrize),
		DayPnL: round2(aggregate.PnL), WinRate: winRate,
	}
}

func itemsFromFilteredRows(rows []sqlcdb.ListCloudBetRecordsFilteredRow) []Item {
	items := make([]Item, len(rows))
	for i, row := range rowsFromDBFiltered(rows) {
		displayPeriod := resolveBetRecordPeriods(row)
		items[i] = Item{
			ID: thirdPartyBetOrderNo(row.ThirdPartyBetID), RecordNo: strings.TrimSpace(row.ID),
			Period: displayPeriod, Periods: displayPeriod, PlayType: row.PlayType,
			Multiplier: formatMultiplierDisplay(row.Multiplier), Round: formatRoundDisplay(row.Round),
			Amount: truncateBetAmount(row.Amount), PnL: round2(row.PnL),
			Status: row.Status, BetContent: row.BetContent,
		}
	}
	return items
}

func (s *Service) loadRows(ctx context.Context, memberID int64, mode Mode, days int) []Row {
	return s.loadRowsFiltered(ctx, memberID, GroupsFilter{Mode: string(mode), Days: days}, time.Time{}, time.Time{})
}

func (s *Service) loadRowsFiltered(
	ctx context.Context,
	memberID int64,
	f GroupsFilter,
	since, until time.Time,
) []Row {
	if since.IsZero() || until.IsZero() {
		var err error
		since, until, _, _, _, err = resolveGroupsRange(f)
		if err != nil {
			return nil
		}
	}
	if s.q != nil && memberID > 0 {
		simBet := pgtype.Bool{}
		if mode := strings.TrimSpace(f.Mode); mode != "" {
			simBet = pgtype.Bool{Bool: mode == string(ModeSim), Valid: true}
		}
		lotteryCode := pgtype.Text{}
		if code := strings.TrimSpace(f.LotteryCode); code != "" {
			lotteryCode = pgtype.Text{String: code, Valid: true}
		}
		guajiID, gerr := member.LookupActiveGuajiAccountID(ctx, s.q, memberID)
		if gerr != nil {
			return nil
		}
		dbRows, err := s.q.ListCloudBetRecordsFiltered(ctx, sqlcdb.ListCloudBetRecordsFilteredParams{
			MemberID:       memberID,
			SinceAt:        pgtype.Timestamptz{Time: since, Valid: true},
			UntilAt:        pgtype.Timestamptz{Time: until, Valid: true},
			SimBet:         simBet,
			LotteryCode:    lotteryCode,
			GuajiAccountID: guajiID,
		})
		if err == nil {
			return rowsFromDBFiltered(dbRows)
		}
	}
	if strings.TrimSpace(f.Mode) == "" {
		real := s.rows(ModeReal)
		sim := s.rows(ModeSim)
		rows := append(real, sim...)
		return filterRowsByLottery(rows, f.LotteryCode)
	}
	return filterRowsByLottery(s.rows(ParseMode(f.Mode)), f.LotteryCode)
}

func resolveGroupsRange(f GroupsFilter) (since, until time.Time, dateFrom, dateTo string, days int, err error) {
	if strings.TrimSpace(f.DateFrom) != "" || strings.TrimSpace(f.DateTo) != "" {
		if err = validateQuerySpan(f.DateFrom, f.DateTo); err != nil {
			return
		}
		since, until, err = timeutil.ParseDateRange(f.DateFrom, f.DateTo)
		if err != nil {
			err = fmt.Errorf("%w: %v", ErrInvalidQuery, err)
			return
		}
		dateFrom = strings.TrimSpace(f.DateFrom)
		dateTo = strings.TrimSpace(f.DateTo)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		if loc == nil {
			loc = time.FixedZone("CST", 8*3600)
		}
		start, _ := time.ParseInLocation("2006-01-02", dateFrom, loc)
		endDay, _ := time.ParseInLocation("2006-01-02", dateTo, loc)
		days = int(endDay.Sub(start).Hours()/24) + 1
		return
	}
	days = f.Days
	if days <= 0 {
		days = 3
	}
	dateFrom, dateTo, since, until = timeutil.NaturalDaysMeta(days)
	return
}

func validateQuerySpan(from, to string) error {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(from), loc)
	if err != nil {
		return fmt.Errorf("%w: dateFrom 格式须为 YYYY-MM-DD", ErrInvalidQuery)
	}
	endDay, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(to), loc)
	if err != nil {
		return fmt.Errorf("%w: dateTo 格式须为 YYYY-MM-DD", ErrInvalidQuery)
	}
	if endDay.Before(start) {
		return fmt.Errorf("%w: dateTo 不能早于 dateFrom", ErrInvalidQuery)
	}
	return nil
}

func filterRowsByLottery(rows []Row, lotteryCode string) []Row {
	code := strings.TrimSpace(lotteryCode)
	if code == "" {
		return rows
	}
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		if r.LotteryCode == code {
			out = append(out, r)
		}
	}
	return out
}

func rowsFromDB(in []sqlcdb.ListCloudBetRecordsRow) []Row {
	out := make([]Row, len(in))
	for i, r := range in {
		out[i] = Row{
			ID:               r.RecordNo,
			ThirdPartyBetID:  pgtextFromAny(r.ThirdPartyBetID),
			SchemeID:         r.SchemeID,
			SchemeName:       r.SchemeName,
			Period:           r.PeriodNo,
			ThirdPartyPeriod: pgtextString(r.ThirdPartyPeriod),
			PlayType:         r.PlayType,
			Multiplier:       r.Multiplier,
			Round:            r.RoundLabel,
			Amount:           r.Amount,
			PnL:              r.Pnl,
			Status:           Status(r.Status),
			BetContent:       r.BetContent,
		}
	}
	return out
}

func rowsFromDBFiltered(in []sqlcdb.ListCloudBetRecordsFilteredRow) []Row {
	out := make([]Row, len(in))
	for i, r := range in {
		out[i] = Row{
			ID:               r.RecordNo,
			ThirdPartyBetID:  pgtextFromAny(r.ThirdPartyBetID),
			SchemeID:         r.SchemeID,
			SchemeName:       r.SchemeName,
			LotteryCode:      r.LotteryCode,
			Period:           r.PeriodNo,
			ThirdPartyPeriod: pgtextString(r.ThirdPartyPeriod),
			PlayType:         r.PlayType,
			Multiplier:       r.Multiplier,
			Round:            r.RoundLabel,
			Amount:           r.Amount,
			PnL:              r.Pnl,
			Status:           Status(r.Status),
			BetContent:       r.BetContent,
		}
	}
	return out
}

func ParseMode(raw string) Mode {
	if strings.TrimSpace(raw) == string(ModeSim) {
		return ModeSim
	}
	return ModeReal
}

func summarize(rows []Row) Summary {
	if len(rows) == 0 {
		return Summary{}
	}
	var totalBet, dayPnL, totalPrize float64
	hits := 0
	for _, r := range rows {
		totalBet += truncateBetAmount(r.Amount)
		dayPnL += r.PnL
		if r.Status == StatusHit {
			hits++
			totalPrize += r.Amount + r.PnL
		}
	}
	return Summary{
		TotalBet:   round2(totalBet),
		TotalPrize: round2(totalPrize),
		DayPnL:     round2(dayPnL),
		WinRate:    round1(float64(hits) / float64(len(rows)) * 100),
	}
}

func groupByScheme(rows []Row) []Group {
	order := make([]string, 0)
	m := make(map[string][]Row)
	for _, r := range rows {
		if _, ok := m[r.SchemeID]; !ok {
			order = append(order, r.SchemeID)
		}
		m[r.SchemeID] = append(m[r.SchemeID], r)
	}
	groups := make([]Group, 0, len(order))
	for _, id := range order {
		rs := m[id]
		sum := summarize(rs)
		groups = append(groups, Group{
			SchemeID:   id,
			SchemeName: rs[0].SchemeName,
			TotalBet:   sum.TotalBet,
			TotalPrize: sum.TotalPrize,
			DayPnL:     sum.DayPnL,
			WinRate:    sum.WinRate,
		})
	}
	return groups
}

func filterScheme(rows []Row, schemeID string) []Row {
	out := make([]Row, 0)
	for _, r := range rows {
		if r.SchemeID == schemeID {
			out = append(out, r)
		}
	}
	return out
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// truncateBetAmount 对齐第三方实际下注金额：第三位小数起直接截断。
func truncateBetAmount(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0
	}
	return math.Floor(v*100+1e-9) / 100
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func pgtextString(t pgtype.Text) string {
	if t.Valid {
		return strings.TrimSpace(t.String)
	}
	return ""
}

func pgtextFromAny(v interface{}) pgtype.Text {
	switch x := v.(type) {
	case pgtype.Text:
		return x
	case string:
		s := strings.TrimSpace(x)
		return pgtype.Text{String: s, Valid: s != ""}
	case []byte:
		s := strings.TrimSpace(string(x))
		return pgtype.Text{String: s, Valid: s != ""}
	case nil:
		return pgtype.Text{}
	default:
		s := strings.TrimSpace(fmt.Sprint(x))
		return pgtype.Text{String: s, Valid: s != "" && s != "<nil>"}
	}
}

// resolveBetRecordPeriods 返回第三方 periods（模拟投注 period_no 即为第三方期号）。
func resolveBetRecordPeriods(r Row) string {
	if p := strings.TrimSpace(r.ThirdPartyPeriod); p != "" {
		return p
	}
	return strings.TrimSpace(r.Period)
}

// ItemByRecordNo 单笔投注详情；校验会员归属与最近 3 天窗口。
func (s *Service) ItemByRecordNo(ctx context.Context, memberID int64, recordNo string) (ItemDetail, error) {
	recordNo = strings.TrimSpace(recordNo)
	if s == nil || s.q == nil || memberID <= 0 || recordNo == "" {
		return ItemDetail{}, ErrItemNotFound
	}
	row, err := s.q.GetCloudBetRecordByMemberRecordNo(ctx, memberID, recordNo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ItemDetail{}, ErrItemNotFound
		}
		return ItemDetail{}, err
	}
	if !row.PlacedAt.Valid {
		return ItemDetail{}, ErrItemNotFound
	}
	since, until := timeutil.NaturalDaysRange(itemDetailDays)
	placed := row.PlacedAt.Time
	if placed.Before(since) || !placed.Before(until) {
		return ItemDetail{}, ErrItemOutOfWindow
	}

	period := strings.TrimSpace(pgtextString(row.ThirdPartyPeriod))
	if period == "" {
		period = strings.TrimSpace(row.PeriodNo)
	}
	status := strings.TrimSpace(row.Status)
	statusLabel := detailStatusLabel(status)

	drawNumbers := ""
	if status == string(StatusHit) || status == string(StatusMiss) {
		drawNumbers, _ = s.lookupDrawNumbers(ctx, row)
		if drawNumbers == "" && !row.SimBet {
			drawNumbers = s.maybeSyncAndLookupDraw(ctx, row)
		}
	}

	var betUnits *int
	if row.BetUnits.Valid && row.BetUnits.Int32 > 0 {
		n := int(row.BetUnits.Int32)
		betUnits = &n
	}

	var payoutAmount *float64
	switch Status(status) {
	case StatusPending:
		payoutAmount = nil
	case StatusMiss:
		z := 0.0
		payoutAmount = &z
	case StatusHit:
		if row.PayoutAmount.Valid {
			if f, ok := numericToFloat64(row.PayoutAmount); ok {
				// 仅展示第三方毛派奖；0/无效视为未回传 → 前端 —
				if f > 1e-9 {
					v := round2(f)
					payoutAmount = &v
				}
			}
		}
		// 中奖但无第三方毛派奖 → 前端显示 —（禁止用 amount+pnl 本地推算）
	}

	currency := strings.ToUpper(strings.TrimSpace(row.Currency))
	if currency == "" {
		currency = "USDT"
	}

	return ItemDetail{
		RecordNo:     row.RecordNo,
		ThirdPartyID: thirdPartyBetOrderNo(row.ThirdPartyBetID),
		Period:       period,
		LotteryLabel: strings.TrimSpace(row.LotteryLabel),
		PlayType:     strings.TrimSpace(row.PlayType),
		Status:       status,
		StatusLabel:  statusLabel,
		DrawNumbers:  drawNumbers,
		BetUnits:     betUnits,
		Multiplier:   formatMultiplierDisplay(row.Multiplier),
		Round:        formatRoundDisplay(row.RoundLabel),
		Amount:       truncateBetAmount(row.Amount),
		Currency:     currency,
		PayoutAmount: payoutAmount,
		PlacedAt:     timeutil.FormatDisplayCST(placed),
		BetContent:   row.BetContent,
		BetContentLines: schemes.FormatBetContentLines(
			row.SchemeKind, []byte(row.SchemeConfig), row.BetContent,
		),
		SimBet: row.SimBet,
	}, nil
}

func detailStatusLabel(status string) string {
	switch Status(status) {
	case StatusHit:
		return "中奖"
	case StatusMiss:
		return "未中奖"
	case StatusPending:
		return "未开奖"
	default:
		return status
	}
}

func (s *Service) lookupDrawNumbers(ctx context.Context, row sqlcdb.CloudBetRecordDetailRow) (string, error) {
	if s == nil || s.q == nil {
		return "", nil
	}
	tp := ""
	if row.ThirdPartyPeriod.Valid {
		tp = row.ThirdPartyPeriod.String
	}
	return s.q.LookupDrawBallsJoined(ctx, row.LotteryCode, row.PeriodNo, tp)
}

func (s *Service) maybeSyncAndLookupDraw(ctx context.Context, row sqlcdb.CloudBetRecordDetailRow) string {
	if s == nil || s.historySync == nil || !row.PlacedAt.Valid {
		return ""
	}
	if time.Since(row.PlacedAt.Time) > drawSyncPlacedWithin {
		return ""
	}
	code := strings.TrimSpace(row.LotteryCode)
	if code == "" {
		return ""
	}
	if last, ok := s.drawSyncMu.Load(code); ok {
		if t, ok := last.(time.Time); ok && time.Since(t) < drawSyncMinGap {
			return ""
		}
	}
	s.drawSyncMu.Store(code, time.Now())
	syncCtx, cancel := context.WithTimeout(ctx, drawSyncTimeout)
	defer cancel()
	_ = s.historySync.SyncLottery(syncCtx, code)
	got, _ := s.lookupDrawNumbers(ctx, row)
	return got
}

func numericToFloat64(n pgtype.Numeric) (float64, bool) {
	if !n.Valid {
		return 0, false
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0, false
	}
	return f.Float64, true
}
