package betrecords

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/timeutil"
)

type Service struct {
	q *sqlcdb.Queries
}

func NewService(pool *db.Pool) *Service {
	if pool == nil {
		return &Service{}
	}
	return &Service{q: sqlcdb.New(pool)}
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
	rows := s.loadRowsFiltered(ctx, memberID, f, since, until)
	allGroups := groupByScheme(rows)
	page, err := paginateGroups(allGroups, limit, f.Cursor)
	if err != nil {
		return GroupsResult{}, err
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
		Summary:  summarize(rows),
		Groups:   page,
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
	all := filterScheme(s.loadRows(ctx, memberID, mode, days), schemeID)
	if len(all) == 0 {
		return DetailResult{}, false, nil
	}
	offset := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 {
			return DetailResult{}, false, fmt.Errorf("invalid cursor")
		}
		offset = n
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
			Period:     displayPeriod,
			Periods:    displayPeriod,
			PlayType:   r.PlayType,
			Multiplier: formatMultiplierDisplay(r.Multiplier),
			Round:      formatRoundDisplay(r.Round),
			Amount:     round2(r.Amount),
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
			Multiplier:      r.Multiplier,
			Round:           r.RoundLabel,
			Amount:          r.Amount,
			PnL:             r.Pnl,
			Status:          Status(r.Status),
			BetContent:      r.BetContent,
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
			Multiplier:      r.Multiplier,
			Round:           r.RoundLabel,
			Amount:          r.Amount,
			PnL:             r.Pnl,
			Status:          Status(r.Status),
			BetContent:      r.BetContent,
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
		totalBet += r.Amount
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
