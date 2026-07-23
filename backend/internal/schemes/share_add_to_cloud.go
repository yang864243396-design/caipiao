package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/timeutil"
)

var (
	ErrSnapshotNotFound = errors.New("share snapshot not found")
)

type Definition struct {
	ID                string                 `json:"id"`
	Kind              string                 `json:"kind"`
	SchemeName        string                 `json:"schemeName"`
	LotteryCode       string                 `json:"lotteryCode"`
	LotteryLabel      string                 `json:"lotteryLabel,omitempty"`
	ShareStatusLocked string                 `json:"shareStatusLocked"`
	Config            map[string]interface{} `json:"config,omitempty"`
	HasInstance       bool                   `json:"hasInstance"`
	CreatedAt         string                 `json:"createdAt"`
	UpdatedAt         string                 `json:"updatedAt"`
}

type Instance struct {
	ID           string  `json:"id"`
	DefinitionID string  `json:"definitionId"`
	Kind         string  `json:"kind"`
	SchemeName   string  `json:"schemeName"`
	LotteryCode  string  `json:"lotteryCode"`
	LotteryLabel string  `json:"lotteryLabel,omitempty"`
	Status       string  `json:"status"`
	StatusReason string  `json:"statusReason,omitempty"`
	StatusLabel  string  `json:"statusLabel"`
	RunMode      string  `json:"runMode"`
	Turnover     float64 `json:"turnover"`
	PnL          float64 `json:"pnl"`
	RunTimeSec   int     `json:"runTimeSec"`
	LookbackPnL  float64 `json:"lookbackPnl"`
	SessionPnL   float64 `json:"sessionPnl"`
	Multiplier   float64 `json:"multiplier"`
	CountdownSec     int     `json:"countdownSec"`
	CountdownCloseAt string  `json:"countdownCloseAt,omitempty"` // RFC3339，由 countdownEndTime 按 UTC 墙钟解析
	CountdownEndTime string  `json:"countdownEndTime,omitempty"` // 第三方 periods 原始 end_time（UTC 墙钟）
	CountdownPeriod  string  `json:"countdownPeriod,omitempty"`  // 倒计时对应期号（第三方 periods）
	CountdownWindowSec int   `json:"countdownWindowSec,omitempty"` // 单期投注窗口秒数（start→end），展示倒计时封顶
	CountdownLabel   string  `json:"countdownLabel,omitempty"`
	SimBet         bool    `json:"simBet"`
	// 方案币种（来自 definition config.schemeCurrency；缺省 USDT）
	SchemeCurrency string `json:"schemeCurrency,omitempty"`
	RunTypeID      string `json:"runTypeId,omitempty"`
	RunTypeLabel   string `json:"runTypeLabel,omitempty"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type ShareFollowActionResult struct {
	Definition Definition `json:"definition"`
	Instance   Instance   `json:"instance"`
}

type ShareAddToCloudInput struct {
	BetMultiplier map[string]interface{}
}

func (s *Service) ShareAddToCloud(ctx context.Context, account, snapshotID string, input ShareAddToCloudInput) (ShareFollowActionResult, error) {
	return s.insertFollowFromSnapshot(ctx, account, snapshotID, "pending", ShareFollowBetInput{}, input.BetMultiplier)
}

func (s *Service) ShareFollowBet(ctx context.Context, account, snapshotID string, input ShareFollowBetInput) (ShareFollowActionResult, error) {
	return s.insertFollowFromSnapshot(ctx, account, snapshotID, "running", input, nil)
}

type ShareFollowBetInput struct {
	LotteryCode  string
	PlayMethod   string
	PlayTemplate string
	TypeID       string
	SubID        string
}

func (s *Service) insertFollowFromSnapshot(
	ctx context.Context,
	account, snapshotID, instanceStatus string,
	input ShareFollowBetInput,
	betMultiplier map[string]interface{},
) (ShareFollowActionResult, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return ShareFollowActionResult{}, ErrUnavailable
	}
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return ShareFollowActionResult{}, ErrSnapshotNotFound
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShareFollowActionResult{}, member.ErrNotFound
		}
		return ShareFollowActionResult{}, err
	}

	snap, err := s.q.GetSchemeShareSnapshotByID(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShareFollowActionResult{}, ErrSnapshotNotFound
		}
		return ShareFollowActionResult{}, err
	}

	existingNames, err := s.q.ListSchemeDefinitionNamesByMember(ctx, m.ID)
	if err != nil {
		return ShareFollowActionResult{}, err
	}
	schemeName := nextUniqueSchemeName(snap.SchemeName, existingNames)

	lotteryCode := snap.LotteryCode
	lotteryLabel := snap.LotteryLabel
	if lc := strings.TrimSpace(input.LotteryCode); lc != "" {
		lotteryCode = lc
		if lbl, err := s.lotteryLabel(ctx, lotteryCode); err != nil {
			return ShareFollowActionResult{}, err
		} else {
			lotteryLabel = lbl
		}
	}

	cfg := mergeSnapshotConfig(snap)
	cfg["lotteryCode"] = lotteryCode
	if pm := strings.TrimSpace(input.PlayMethod); pm != "" {
		cfg["playMethod"] = pm
	}
	if pt := strings.TrimSpace(input.PlayTemplate); pt != "" {
		cfg["playTemplate"] = pt
	}
	if tid := strings.TrimSpace(input.TypeID); tid != "" {
		cfg["playTypeId"] = tid
		cfg["typeId"] = tid
	}
	if sid := strings.TrimSpace(input.SubID); sid != "" {
		cfg["subPlayId"] = sid
		cfg["subId"] = sid
	}
	if len(betMultiplier) > 0 {
		cfg["betMultiplier"] = betMultiplier
		if rounds := compileBetMultiplierRounds(betMultiplier, cfg); len(rounds) > 0 {
			cfg["rounds"] = rounds
		}
	}
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return ShareFollowActionResult{}, err
	}

	nowMs := time.Now().UnixMilli()
	defID := fmt.Sprintf("def-%d-%d", m.ID, nowMs)
	instID := fmt.Sprintf("inst-%d-%d", m.ID, nowMs)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ShareFollowActionResult{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	defRow, err := qtx.InsertSchemeDefinition(ctx, sqlcdb.InsertSchemeDefinitionParams{
		ID:                 defID,
		MemberID:           m.ID,
		Kind:               "follow",
		SchemeName:         schemeName,
		LotteryCode:        lotteryCode,
		LotteryLabel:       lotteryLabel,
		ShareStatus:        "private",
		ShareStatusLocked:  true,
		SourceSnapshotID:   pgtype.Text{String: snap.ID, Valid: true},
		Config:             cfgBytes,
	})
	if err != nil {
		return ShareFollowActionResult{}, err
	}
	instRow, err := qtx.InsertSchemeInstance(ctx, sqlcdb.InsertSchemeInstanceParams{
		ID:           instID,
		DefinitionID: defID,
		MemberID:     m.ID,
		Kind:         "follow",
		SchemeName:   schemeName,
		LotteryCode:  lotteryCode,
		LotteryLabel: lotteryLabel,
		Status:       instanceStatus,
		SimBet:       false,
	})
	if err != nil {
		return ShareFollowActionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ShareFollowActionResult{}, err
	}

	return ShareFollowActionResult{
		Definition: mapDefinitionRow(defRow, true),
		Instance:   mapInstanceFromInsertRow(instRow),
	}, nil
}

func mergeSnapshotConfig(snap sqlcdb.SchemeShareSnapshot) map[string]interface{} {
	cfg := map[string]interface{}{}
	if len(snap.Config) > 0 {
		_ = json.Unmarshal(snap.Config, &cfg)
	}
	cfg["schemeName"] = snap.SchemeName
	cfg["lotteryCode"] = snap.LotteryCode
	if snap.PlayMethod != "" {
		cfg["playMethod"] = snap.PlayMethod
	}
	if snap.FundYuan.Valid {
		if f, err := snap.FundYuan.Float64Value(); err == nil && f.Valid {
			cfg["fundYuan"] = f.Float64
		}
	}
	return cfg
}

func nextUniqueSchemeName(base string, existing []string) string {
	trimmed := strings.TrimSpace(base)
	if trimmed == "" {
		trimmed = "方案"
	}
	set := make(map[string]struct{}, len(existing))
	for _, n := range existing {
		set[strings.TrimSpace(n)] = struct{}{}
	}
	if _, ok := set[trimmed]; !ok {
		return trimmed
	}
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d", trimmed, i)
		if _, ok := set[candidate]; !ok {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d", trimmed, time.Now().UnixMilli())
}

func mapInstanceRow(row sqlcdb.SchemeInstance) Instance {
	betFailedDetail := ""
	if row.BetFailedDetail.Valid {
		betFailedDetail = row.BetFailedDetail.String
	}
	return Instance{
		ID:           row.ID,
		DefinitionID: row.DefinitionID,
		Kind:         row.Kind,
		SchemeName:   row.SchemeName,
		LotteryCode:  row.LotteryCode,
		LotteryLabel: row.LotteryLabel,
		Status:       row.Status,
		StatusReason: row.StatusReason,
		StatusLabel:  instanceStatusLabel(row.Status, row.StatusReason, betFailedDetail),
		RunMode:      runModeFromSimBet(row.SimBet),
		Turnover:     numericToFloat(row.Turnover),
		PnL:          numericToFloat(row.Pnl),
		RunTimeSec:   int(row.RunTimeSec),
		LookbackPnL:  numericToFloat(row.LookbackPnl),
		SessionPnL:   numericToFloat(row.SessionPnl),
		Multiplier:   numericToFloat(row.Multiplier),
		CountdownSec: int(row.CountdownSec),
		SimBet:       row.SimBet,
		CreatedAt:    timeutil.FormatISO(row.CreatedAt.Time),
		UpdatedAt:    timeutil.FormatISO(row.UpdatedAt.Time),
	}
}

func mapInstanceFromInsertRow(row sqlcdb.InsertSchemeInstanceRow) Instance {
	return mapInstanceRow(sqlcdb.SchemeInstanceFromInsertRow(row))
}

func mapInstanceFromAdminStatusRow(row sqlcdb.UpdateSchemeInstanceStatusByAdminRow) Instance {
	return mapInstanceRow(sqlcdb.SchemeInstanceFromAdminStatusRow(row))
}
