package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

var (
	ErrInstanceNotFound      = errors.New("scheme instance not found")
	ErrInvalidAdminAction    = errors.New("invalid admin action for current status")
	ErrSnapshotKindImmutable = errors.New("share snapshot kind immutable")
)

type AdminMonitorRow struct {
	InstanceID   string `json:"instanceId"`
	DefinitionID string `json:"definitionId"`
	MemberID     string `json:"memberId"`
	MemberName   string `json:"memberName"`
	Kind         string `json:"kind"`
	RunTypeID    string `json:"runTypeId,omitempty"`
	RunTypeLabel string `json:"runTypeLabel,omitempty"`
	PlayTypeID   string `json:"playTypeId,omitempty"`
	PlayTypeLabel string `json:"playTypeLabel,omitempty"`
	SchemeName   string `json:"schemeName"`
	LotteryCode  string `json:"lotteryCode"`
	LotteryLabel string `json:"lotteryLabel"`
	Status       string `json:"status"`
	StatusLabel  string `json:"statusLabel"`
	SimBet       bool   `json:"simBet"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type AdminMonitorQuery struct {
	Scope       string
	Keyword     string
	SearchField string
	Kind        string
	Status      string
	SimBet      string
	LotteryCode string
	Limit       int
}

type AdminMonitorListResult struct {
	Scope string        `json:"scope"`
	Items []interface{} `json:"items"`
}

type PatchShareSnapshotInput struct {
	Config       map[string]interface{}
	SchemeName   string
	LotteryCode  string
	LotteryLabel string
	PlayMethod   string
	FundYuan     *float64
}

func (s *Service) AdminMonitorList(ctx context.Context, q AdminMonitorQuery) (AdminMonitorListResult, error) {
	if s == nil || s.q == nil {
		return AdminMonitorListResult{}, ErrUnavailable
	}
	scope := strings.TrimSpace(q.Scope)
	if scope == "share" {
		items, err := s.adminListShareSnapshots(ctx, q)
		if err != nil {
			return AdminMonitorListResult{}, err
		}
		out := make([]interface{}, 0, len(items))
		for _, item := range items {
			out = append(out, item)
		}
		return AdminMonitorListResult{Scope: "share", Items: out}, nil
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}

	kw := pgText(q.Keyword)
	kind := pgText(ParseAdminKind(q.Kind))
	status := pgText(ParseAdminStatus(q.Status))
	simBet := parseAdminSimBetFilter(q.SimBet)

	rows, err := s.q.ListAdminSchemeInstances(ctx, sqlcdb.ListAdminSchemeInstancesParams{
		Keyword:     kw,
		SearchField: pgText(normalizeAdminSearchField(q.SearchField)),
		Kind:        kind,
		Status:      status,
		SimBet:      simBet,
		LotteryCode: pgText(strings.TrimSpace(q.LotteryCode)),
		RowLimit:    int32(limit),
	})
	if err != nil {
		return AdminMonitorListResult{}, err
	}

	items := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		item := mapAdminMonitorRow(row)
		if item.PlayTypeLabel == "" && item.PlayTypeID != "" {
			item.PlayTypeLabel = s.resolveAdminPlayTypeLabel(ctx, item.LotteryCode, item.PlayTypeID)
		}
		items = append(items, item)
	}
	return AdminMonitorListResult{Scope: "user", Items: items}, nil
}

func (s *Service) adminListShareSnapshots(ctx context.Context, q AdminMonitorQuery) ([]ShareSnapshot, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.q.ListAdminSchemeShareSnapshots(ctx, sqlcdb.ListAdminSchemeShareSnapshotsParams{
		Keyword:     pgText(q.Keyword),
		SearchField: pgText(normalizeAdminShareSearchField(q.SearchField)),
		LotteryCode: pgText(strings.TrimSpace(q.LotteryCode)),
		RowLimit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	items := make([]ShareSnapshot, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapShareSnapshotRow(row))
	}
	return items, nil
}

func (s *Service) AdminForceStop(ctx context.Context, instanceID string) (Instance, error) {
	return s.adminTransitionInstance(ctx, instanceID, "soft_stopped", "running")
}

func (s *Service) AdminReleaseStop(ctx context.Context, instanceID string) (Instance, error) {
	return s.adminTransitionInstance(ctx, instanceID, "paused", "soft_stopped")
}

func (s *Service) adminTransitionInstance(
	ctx context.Context,
	instanceID, nextStatus, requiredStatus string,
) (Instance, error) {
	if s == nil || s.q == nil {
		return Instance{}, ErrUnavailable
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return Instance{}, ErrInstanceNotFound
	}

	cur, err := s.q.GetSchemeInstanceByID(ctx, instanceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrInstanceNotFound
		}
		return Instance{}, err
	}
	if cur.Status != requiredStatus {
		return Instance{}, ErrInvalidAdminAction
	}

	row, err := s.q.UpdateSchemeInstanceStatusByAdmin(ctx, sqlcdb.UpdateSchemeInstanceStatusByAdminParams{
		ID:     instanceID,
		Status: nextStatus,
	})
	if err != nil {
		return Instance{}, err
	}
	return mapInstanceFromAdminStatusRow(row), nil
}

func (s *Service) AdminPatchShareSnapshot(ctx context.Context, snapshotID string, in PatchShareSnapshotInput) (ShareSnapshot, error) {
	if s == nil || s.q == nil {
		return ShareSnapshot{}, ErrUnavailable
	}
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return ShareSnapshot{}, ErrSnapshotNotFound
	}

	cur, err := s.q.GetSchemeShareSnapshotByID(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShareSnapshot{}, ErrSnapshotNotFound
		}
		return ShareSnapshot{}, err
	}
	if cur.Kind != "custom" {
		return ShareSnapshot{}, ErrSnapshotKindImmutable
	}

	cfg := map[string]interface{}{}
	if len(cur.Config) > 0 {
		_ = json.Unmarshal(cur.Config, &cfg)
	}
	for k, v := range in.Config {
		cfg[k] = v
	}

	schemeName := cur.SchemeName
	if in.SchemeName != "" {
		schemeName = in.SchemeName
	} else if v, ok := cfg["schemeName"].(string); ok && v != "" {
		schemeName = v
	}

	lotteryCode := cur.LotteryCode
	if in.LotteryCode != "" {
		lotteryCode = in.LotteryCode
	} else if v, ok := cfg["lotteryCode"].(string); ok && v != "" {
		lotteryCode = v
	}

	lotteryLabel := cur.LotteryLabel
	if in.LotteryLabel != "" {
		lotteryLabel = in.LotteryLabel
	}

	playMethod := cur.PlayMethod
	if in.PlayMethod != "" {
		playMethod = in.PlayMethod
	}

	fundYuan := numericToFloat(cur.FundYuan)
	if in.FundYuan != nil {
		fundYuan = *in.FundYuan
	}

	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return ShareSnapshot{}, err
	}

	row, err := s.q.UpdateSchemeShareSnapshotAdmin(ctx, sqlcdb.UpdateSchemeShareSnapshotAdminParams{
		ID:           snapshotID,
		SchemeName:   schemeName,
		LotteryCode:  lotteryCode,
		LotteryLabel: lotteryLabel,
		PlayMethod:   playMethod,
		FundYuan:     numericFromFloat(fundYuan),
		Config:       cfgBytes,
	})
	if err != nil {
		return ShareSnapshot{}, err
	}
	return mapShareSnapshotRow(row), nil
}

func (s *Service) AdminDeleteShareSnapshot(ctx context.Context, snapshotID string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return ErrSnapshotNotFound
	}
	n, err := s.q.DeleteSchemeShareSnapshot(ctx, snapshotID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSnapshotNotFound
	}
	return nil
}

func (s *Service) resolveAdminPlayTypeLabel(ctx context.Context, lotteryCode, playTypeID string) string {
	playTypeID = strings.TrimSpace(playTypeID)
	if playTypeID == "" {
		return ""
	}
	if label := adminStaticPlayTypeLabel(playTypeID); label != "" {
		return label
	}
	if s == nil || s.q == nil {
		return playTypeID
	}
	cat, err := s.q.GetLotteryCatalogByCode(ctx, lotteryCode)
	if err != nil {
		return playTypeID
	}
	template := strings.TrimSpace(cat.PlayTemplate.String)
	if template == "" {
		return playTypeID
	}
	if label := s.playTypeLabel(ctx, template, playTypeID); label != "" {
		return label
	}
	return playTypeID
}

func adminStaticPlayTypeLabel(typeID string) string {
	labels := map[string]string{
		"dingwei": "定位胆", "g006": "一星", "hou4": "后四", "sixing": "四星",
		"qian3": "前三", "zhong3": "中三", "hou3": "后三", "qian2": "前二", "hou2": "后二",
		"longhu": "龙虎", "renxuan": "任选", "wuxing": "五星", "tema": "特码",
	}
	return labels[typeID]
}

func mapAdminMonitorRow(row sqlcdb.ListAdminSchemeInstancesRow) AdminMonitorRow {
	runTypeID, runTypeLabel := "", ""
	if row.Kind == "custom" {
		runTypeID = NormalizeRunTypeID(fmt.Sprint(row.RunType))
		runTypeLabel = RunTypeLabels[runTypeID]
	}
	return AdminMonitorRow{
		InstanceID:   row.ID,
		DefinitionID: row.DefinitionID,
		MemberID:     fmt.Sprintf("%d", row.MemberID),
		MemberName:   row.Account,
		Kind:         row.Kind,
		RunTypeID:    runTypeID,
		RunTypeLabel: runTypeLabel,
		PlayTypeID:   strings.TrimSpace(fmt.Sprint(row.PlayTypeID)),
		PlayTypeLabel: strings.TrimSpace(fmt.Sprint(row.PlayTypeLabel)),
		SchemeName:   row.SchemeName,
		LotteryCode:  row.LotteryCode,
		LotteryLabel: row.LotteryLabel,
		Status:       row.Status,
		StatusLabel:  instanceStatusLabel(row.Status, row.StatusReason, ""),
		SimBet:       row.SimBet,
		CreatedAt:    timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt:    timeutil.FormatISO(row.UpdatedAt.Time),
	}
}

func pgText(v string) pgtype.Text {
	v = strings.TrimSpace(v)
	if v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

func normalizeAdminShareSearchField(raw string) string {
	switch strings.TrimSpace(raw) {
	case "snapshotId", "snapshot_id", "id":
		return "snapshotId"
	default:
		return "schemeName"
	}
}

func normalizeAdminSearchField(raw string) string {
	switch strings.TrimSpace(raw) {
	case "schemeName", "scheme_name":
		return "schemeName"
	default:
		return "account"
	}
}

func ParseAdminKind(label string) string {
	switch strings.TrimSpace(label) {
	case "自创", "custom":
		return "custom"
	case "反买", "contrary":
		return "contrary"
	case "跟单", "follow":
		return "follow"
	default:
		return ""
	}
}

func ParseAdminStatus(label string) string {
	switch strings.TrimSpace(label) {
	case "待开启", "pending":
		return "pending"
	case "运行中", "running":
		return "running"
	case "已暂停", "paused":
		return "paused"
	case "已封停", "soft_stopped":
		return "soft_stopped"
	default:
		return ""
	}
}

func parseAdminSimBetFilter(raw string) pgtype.Bool {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "all":
		return pgtype.Bool{}
	case "true", "1", "sim", "模拟":
		return pgtype.Bool{Bool: true, Valid: true}
	case "false", "0", "real", "正式", "prod":
		return pgtype.Bool{Bool: false, Valid: true}
	default:
		return pgtype.Bool{}
	}
}

func numericFromFloat(v float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", v))
	return n
}
