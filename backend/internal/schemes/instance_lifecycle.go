package schemes

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

var ErrInvalidInstanceAction = errors.New("invalid instance action for current status")
var ErrInstanceRunningSimBet = errors.New("cannot change simBet while instance is running")

type InstanceListResult struct {
	Items []Instance `json:"items"`
	Total int64      `json:"total,omitempty"`
	Page  PageMeta   `json:"page,omitempty"`
}

type InstanceListQuery struct {
	RunMode string
	Limit   int
	Cursor  string
	IDs     []string
	// Search 方案名称 / 彩种 / definitionId / 实例 id（服务端过滤，非仅已加载页）
	Search string
}

func (s *Service) ListInstances(ctx context.Context, account string, runMode string) (InstanceListResult, error) {
	return s.ListInstancesQuery(ctx, account, InstanceListQuery{RunMode: runMode})
}

func (s *Service) ListInstancesQuery(ctx context.Context, account string, q InstanceListQuery) (InstanceListResult, error) {
	if s == nil || s.q == nil {
		return InstanceListResult{}, ErrUnavailable
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InstanceListResult{}, member.ErrNotFound
		}
		return InstanceListResult{}, err
	}
	if len(q.IDs) > 0 {
		return s.listInstancesByIDs(ctx, m.ID, q.IDs)
	}
	if q.Limit > 0 {
		return s.listInstancesPaginated(ctx, m.ID, q.RunMode, q.Limit, q.Cursor, q.Search)
	}
	return s.listAllInstances(ctx, m.ID, q.RunMode)
}

func (s *Service) listAllInstances(ctx context.Context, memberID int64, runMode string) (InstanceListResult, error) {
	rows, err := s.q.ListSchemeInstancesByMember(ctx, memberID)
	if err != nil {
		return InstanceListResult{}, err
	}
	return s.mapInstanceRows(ctx, memberID, sqlcdb.SchemeInstanceFromListRows(rows), runMode, 0, PageMeta{})
}

func (s *Service) listInstancesByIDs(ctx context.Context, memberID int64, ids []string) (InstanceListResult, error) {
	rows, err := s.q.ListSchemeInstancesByMemberIDsEx(ctx, memberID, ids)
	if err != nil {
		return InstanceListResult{}, err
	}
	return s.mapInstanceRows(ctx, memberID, rows, "", 0, PageMeta{})
}

func (s *Service) listInstancesPaginated(ctx context.Context, memberID int64, runMode string, limit int, cursor, search string) (InstanceListResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	cursorAt, cursorID, err := decodeInstanceCursor(cursor)
	if err != nil {
		return InstanceListResult{}, ErrInvalidInstanceAction
	}
	search = sqlcdb.EscapeILIKEPattern(search)
	total, err := s.q.CountSchemeInstancesByMemberSearch(ctx, sqlcdb.CountSchemeInstancesByMemberSearchParams{
		MemberID: memberID,
		RunMode:  runMode,
		Search:   search,
	})
	if err != nil {
		return InstanceListResult{}, err
	}
	rows, err := s.q.ListSchemeInstancesByMemberPaginatedSearch(ctx, sqlcdb.ListSchemeInstancesByMemberPaginatedSearchParams{
		MemberID: memberID,
		RunMode:  runMode,
		CursorAt: cursorAt,
		CursorID: cursorID,
		Search:   search,
		Limit:    int32(limit + 1),
	})
	if err != nil {
		return InstanceListResult{}, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	page := PageMeta{HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		page.NextCursor = encodeInstanceCursor(last.UpdatedAt.Time, last.ID)
	}
	return s.mapInstanceRows(ctx, memberID, rows, runMode, total, page)
}

func (s *Service) mapInstanceRows(
	ctx context.Context,
	memberID int64,
	rows []sqlcdb.SchemeInstance,
	runMode string,
	total int64,
	page PageMeta,
) (InstanceListResult, error) {
	defIDs := make([]string, 0, len(rows))
	seenDef := map[string]bool{}
	for _, row := range rows {
		id := strings.TrimSpace(row.DefinitionID)
		if id == "" || seenDef[id] {
			continue
		}
		seenDef[id] = true
		defIDs = append(defIDs, id)
	}
	defMeta := map[string]sqlcdb.SchemeDefinitionRunTypeRow{}
	if len(defIDs) > 0 {
		if rtRows, rtErr := s.q.ListSchemeDefinitionRunTypesByIDs(ctx, memberID, defIDs); rtErr == nil {
			for _, r := range rtRows {
				defMeta[r.ID] = r
			}
		}
	}
	// 列表倒计时只用本地 periods 缓存，禁止在此同步拉第三方（易串行拖到数百 ms/彩种）。
	items := make([]Instance, 0, len(rows))
	now := time.Now()
	for _, row := range rows {
		if runMode != "" {
			wantSim := runMode == "sim"
			if row.SimBet != wantSim {
				continue
			}
		}
		row = s.maybeActivateAfterStartPeriod(ctx, row, now)
		meta := defMeta[row.DefinitionID]
		item := enrichInstanceListItem(row, now, meta.SchemeCurrency)
		if row.Kind == "custom" {
			rt := NormalizeRunTypeID(meta.RunType)
			item.RunTypeID = rt
			item.RunTypeLabel = RunTypeLabels[rt]
		}
		items = append(items, item)
	}
	return InstanceListResult{Items: items, Total: total, Page: page}, nil
}

func (s *Service) StopInstance(ctx context.Context, account, instanceID string) (Instance, error) {
	if s == nil || s.q == nil {
		return Instance{}, ErrUnavailable
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return Instance{}, ErrDefinitionNotFound
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, member.ErrNotFound
		}
		return Instance{}, err
	}

	cur, err := s.q.GetSchemeInstanceByIDAndMember(ctx, sqlcdb.GetSchemeInstanceByIDAndMemberParams{
		ID: instanceID, MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrDefinitionNotFound
		}
		return Instance{}, err
	}
	if cur.Status == "soft_stopped" {
		return Instance{}, ErrInvalidInstanceAction
	}
	if cur.Status != "running" {
		return Instance{}, ErrInvalidInstanceAction
	}

	row, err := s.q.UpdateSchemeInstanceStatusFromRunningToPending(ctx, sqlcdb.UpdateSchemeInstanceStatusFromRunningToPendingParams{
		ID: instanceID, MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrInvalidInstanceAction
		}
		return Instance{}, err
	}
	return s.enrichInstanceForDisplay(ctx, sqlcdb.SchemeInstanceFromRunningToPendingRow(row), time.Now()), nil
}

func (s *Service) PauseInstance(ctx context.Context, account, instanceID string) (Instance, error) {
	return s.transitionInstance(ctx, account, instanceID, "paused", "running")
}

func (s *Service) transitionInstance(
	ctx context.Context,
	account, instanceID, nextStatus, requiredStatus string,
) (Instance, error) {
	if s == nil || s.q == nil {
		return Instance{}, ErrUnavailable
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return Instance{}, ErrDefinitionNotFound
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, member.ErrNotFound
		}
		return Instance{}, err
	}

	cur, err := s.q.GetSchemeInstanceByIDAndMember(ctx, sqlcdb.GetSchemeInstanceByIDAndMemberParams{
		ID:       instanceID,
		MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrDefinitionNotFound
		}
		return Instance{}, err
	}

	if cur.Status == "soft_stopped" {
		return Instance{}, ErrInvalidInstanceAction
	}
	if cur.Status != requiredStatus {
		return Instance{}, ErrInvalidInstanceAction
	}

	row, err := s.q.UpdateSchemeInstanceStatusToPaused(ctx, sqlcdb.UpdateSchemeInstanceStatusToPausedParams{
		ID:       instanceID,
		MemberID: m.ID,
		Column3:  transitionStatusReason(nextStatus, requiredStatus),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrInvalidInstanceAction
		}
		return Instance{}, err
	}
	return s.enrichInstanceForDisplay(ctx, sqlcdb.SchemeInstanceFromPausedRow(row), time.Now()), nil
}

func transitionStatusReason(nextStatus, requiredStatus string) string {
	if nextStatus == "paused" && requiredStatus == "running" {
		return StatusReasonManual
	}
	return ""
}
