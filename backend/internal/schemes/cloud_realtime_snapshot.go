package schemes

import (
	"context"
	"sort"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

type RealtimeInstanceRef struct {
	MemberID   int64
	InstanceID string
}

type RealtimeSchemeSnapshotResult struct {
	ItemsByMember   map[int64][]Instance
	RemovedByMember map[int64][]string
}

func (s *Service) LoadRealtimeSchemeSnapshots(ctx context.Context, refs []RealtimeInstanceRef) (RealtimeSchemeSnapshotResult, error) {
	if s == nil || s.q == nil {
		return RealtimeSchemeSnapshotResult{}, ErrUnavailable
	}
	normalizedRefs, instanceIDs := normalizeRealtimeInstanceRefs(refs)
	if len(instanceIDs) == 0 {
		return groupRealtimeSchemeSnapshots(nil, nil, nil, time.Now()), nil
	}
	rows, err := s.q.ListSchemeInstancesRealtimeByIDs(ctx, instanceIDs)
	if err != nil {
		return RealtimeSchemeSnapshotResult{}, err
	}
	definitionIDs := make([]string, 0, len(rows))
	definitionSet := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.DefinitionID == "" {
			continue
		}
		if _, ok := definitionSet[row.DefinitionID]; ok {
			continue
		}
		definitionSet[row.DefinitionID] = struct{}{}
		definitionIDs = append(definitionIDs, row.DefinitionID)
	}
	sort.Strings(definitionIDs)
	meta, err := s.q.ListSchemeDefinitionRealtimeMeta(ctx, definitionIDs)
	if err != nil {
		return RealtimeSchemeSnapshotResult{}, err
	}
	return groupRealtimeSchemeSnapshots(normalizedRefs, rows, meta, time.Now()), nil
}

func normalizeRealtimeInstanceRefs(refs []RealtimeInstanceRef) ([]RealtimeInstanceRef, []string) {
	normalized := make([]RealtimeInstanceRef, 0, len(refs))
	pairs := make(map[RealtimeInstanceRef]struct{}, len(refs))
	ids := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		ref.InstanceID = strings.TrimSpace(ref.InstanceID)
		if ref.MemberID <= 0 || ref.InstanceID == "" {
			continue
		}
		if _, ok := pairs[ref]; !ok {
			pairs[ref] = struct{}{}
			normalized = append(normalized, ref)
		}
		ids[ref.InstanceID] = struct{}{}
	}
	instanceIDs := make([]string, 0, len(ids))
	for id := range ids {
		instanceIDs = append(instanceIDs, id)
	}
	sort.Strings(instanceIDs)
	return normalized, instanceIDs
}

func (s *Service) LoadRealtimeStats(ctx context.Context, memberIDs []int64) (map[int64]CloudCenterStats, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	memberIDs = normalizeRealtimeMemberIDs(memberIDs)
	result := make(map[int64]CloudCenterStats, len(memberIDs))
	for _, memberID := range memberIDs {
		result[memberID] = emptyCloudCenterStats()
	}
	if len(memberIDs) == 0 {
		return result, nil
	}
	rows, err := s.q.ListCloudRealtimeStats(ctx, memberIDs, shanghaiTodayDate(time.Now()))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, requested := result[row.MemberID]; !requested {
			continue
		}
		result[row.MemberID] = mapCloudRealtimeStatsRow(row)
	}
	return result, nil
}

func normalizeRealtimeMemberIDs(memberIDs []int64) []int64 {
	set := make(map[int64]struct{}, len(memberIDs))
	for _, memberID := range memberIDs {
		if memberID > 0 {
			set[memberID] = struct{}{}
		}
	}
	result := make([]int64, 0, len(set))
	for memberID := range set {
		result = append(result, memberID)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func mapCloudRealtimeStatsRow(row sqlcdb.CloudRealtimeStatsRow) CloudCenterStats {
	generatedAt := ""
	if !row.GeneratedAt.IsZero() {
		generatedAt = row.GeneratedAt.UTC().Format(time.RFC3339Nano)
	}
	return CloudCenterStats{
		GeneratedAt: generatedAt,
		Formal: mapCloudCenterChannelStats(sqlcdb.MemberCloudCenterStatsRow{
			TotalTurnover:     row.FormalTotalTurnover,
			TotalSessionPnl:   row.FormalTotalSessionPnl,
			RunningSessionPnl: row.FormalRunningSessionPnl,
		}),
		Sim: mapCloudCenterChannelStats(sqlcdb.MemberCloudCenterStatsRow{
			TotalTurnover:     row.SimTotalTurnover,
			TotalSessionPnl:   row.SimTotalSessionPnl,
			RunningSessionPnl: row.SimRunningSessionPnl,
		}),
		SimQuota: SimSchemeQuota{
			TodayStarts:      int(row.TodaySimSchemeStarts),
			TodayStartsLimit: maxSimSchemeDailyStarts,
			Running:          int(row.RunningSimSchemes),
			RunningLimit:     maxSimSchemeConcurrent,
		},
	}
}

func emptyCloudCenterStats() CloudCenterStats {
	return CloudCenterStats{SimQuota: SimSchemeQuota{
		TodayStartsLimit: maxSimSchemeDailyStarts,
		RunningLimit:     maxSimSchemeConcurrent,
	}}
}

func groupRealtimeSchemeSnapshots(
	refs []RealtimeInstanceRef,
	rows []sqlcdb.SchemeInstance,
	definitionMeta []sqlcdb.CloudRealtimeDefinitionMeta,
	now time.Time,
) RealtimeSchemeSnapshotResult {
	result := RealtimeSchemeSnapshotResult{
		ItemsByMember:   make(map[int64][]Instance),
		RemovedByMember: make(map[int64][]string),
	}
	rowsByID := make(map[string]sqlcdb.SchemeInstance, len(rows))
	for _, row := range rows {
		rowsByID[row.ID] = row
	}
	metaByDefinitionID := make(map[string]sqlcdb.CloudRealtimeDefinitionMeta, len(definitionMeta))
	for _, meta := range definitionMeta {
		metaByDefinitionID[meta.ID] = meta
	}
	for _, ref := range refs {
		row, ok := rowsByID[ref.InstanceID]
		if !ok || row.MemberID != ref.MemberID {
			result.RemovedByMember[ref.MemberID] = append(result.RemovedByMember[ref.MemberID], ref.InstanceID)
			continue
		}
		meta := metaByDefinitionID[row.DefinitionID]
		if meta.MemberID != row.MemberID {
			meta = sqlcdb.CloudRealtimeDefinitionMeta{}
		}
		item := enrichInstanceListItem(row, now, meta.SchemeCurrency)
		if row.Kind == "custom" && meta.ID != "" {
			item.RunTypeID = NormalizeRunTypeID(meta.RunType)
			item.RunTypeLabel = RunTypeLabels[item.RunTypeID]
		}
		result.ItemsByMember[ref.MemberID] = append(result.ItemsByMember[ref.MemberID], item)
	}
	for memberID := range result.ItemsByMember {
		sort.Slice(result.ItemsByMember[memberID], func(i, j int) bool {
			return result.ItemsByMember[memberID][i].ID < result.ItemsByMember[memberID][j].ID
		})
	}
	for memberID := range result.RemovedByMember {
		sort.Strings(result.RemovedByMember[memberID])
	}
	return result
}
