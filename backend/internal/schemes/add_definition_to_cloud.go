package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

const addCloudCooldown = time.Second

var (
	ErrDefinitionNotFound = errors.New("scheme definition not found")
	ErrAlreadyHasInstance = errors.New("scheme already has instance")
	ErrAddCloudTooFast    = errors.New("add to cloud too fast")
	ErrShareNotAllowed    = errors.New("share not allowed for scheme kind")
)

var addCloudLastAt sync.Map // key: memberAccount:definitionID -> unix ms

type AddToCloudConfigPatch struct {
	RunMode      string
	SchemeFunds  string
	StartTime    string
	EndTime      string
	SchemeGroups []string
	StopLoss     string
	TakeProfit   string
	BetUnit      string
	BetMode      string
	PlayTemplate string
	TypeID       string
	SubID        string
}

type AddToCloudResult struct {
	Definition      Definition `json:"definition"`
	Instance        Instance   `json:"instance"`
	ShareSnapshotID string     `json:"shareSnapshotId,omitempty"`
}

func (s *Service) AddDefinitionToCloud(
	ctx context.Context,
	account, definitionID, shareStatus string,
	patch AddToCloudConfigPatch,
) (AddToCloudResult, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return AddToCloudResult{}, ErrUnavailable
	}
	definitionID = strings.TrimSpace(definitionID)
	if definitionID == "" {
		return AddToCloudResult{}, ErrDefinitionNotFound
	}

	now := time.Now()
	if last, ok := addCloudLastAt.Load(addCloudKey(account, definitionID)); ok {
		if now.Sub(time.UnixMilli(last.(int64))) < addCloudCooldown {
			return AddToCloudResult{}, ErrAddCloudTooFast
		}
	}
	addCloudLastAt.Store(addCloudKey(account, definitionID), now.UnixMilli())

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AddToCloudResult{}, member.ErrNotFound
		}
		return AddToCloudResult{}, err
	}

	def, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       definitionID,
		MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AddToCloudResult{}, ErrDefinitionNotFound
		}
		return AddToCloudResult{}, err
	}

	if _, err := s.q.GetSchemeInstanceByDefinitionID(ctx, definitionID); err == nil {
		return AddToCloudResult{}, ErrAlreadyHasInstance
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return AddToCloudResult{}, err
	}

	resolvedShare := resolveShareStatus(def.Kind, shareStatus)
	if resolvedShare == "" {
		return AddToCloudResult{}, ErrShareNotAllowed
	}

	cfgBytes, err := mergeDefinitionConfig(def.Config, patch)
	if err != nil {
		return AddToCloudResult{}, err
	}

	simBet := configSimBet(cfgBytes)
	if patch.RunMode != "" {
		simBet = simBetFromClientRunMode(patch.RunMode)
	}
	nowMs := now.UnixMilli()
	instID := fmt.Sprintf("inst-%d-%d", m.ID, nowMs)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AddToCloudResult{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	defRow, err := qtx.UpdateSchemeDefinitionForCloud(ctx, sqlcdb.UpdateSchemeDefinitionForCloudParams{
		ID:          definitionID,
		MemberID:    m.ID,
		ShareStatus: resolvedShare,
		Config:      cfgBytes,
	})
	if err != nil {
		return AddToCloudResult{}, err
	}

	instRow, err := qtx.InsertSchemeInstance(ctx, sqlcdb.InsertSchemeInstanceParams{
		ID:           instID,
		DefinitionID: definitionID,
		MemberID:     m.ID,
		Kind:         def.Kind,
		SchemeName:   def.SchemeName,
		LotteryCode:  def.LotteryCode,
		LotteryLabel: def.LotteryLabel,
		Status:       "pending",
		SimBet:       simBet,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return AddToCloudResult{}, ErrNameDuplicate
		}
		return AddToCloudResult{}, err
	}

	var shareSnapshotID string
	if def.Kind == "custom" && resolvedShare == "public" {
		snapID := fmt.Sprintf("SD%d", nowMs)
		fundYuan := parseSchemeFunds(patch.SchemeFunds, cfgBytes)
		playMethod := extractPlayMethod(cfgBytes)
		if _, err := qtx.InsertSchemeShareSnapshot(ctx, sqlcdb.InsertSchemeShareSnapshotParams{
			ID:           snapID,
			SchemeName:   def.SchemeName,
			LotteryCode:  def.LotteryCode,
			LotteryLabel: def.LotteryLabel,
			PlayMethod:   playMethod,
			FundYuan:     fundYuan,
			Config:       cfgBytes,
		}); err != nil {
			return AddToCloudResult{}, err
		}
		shareSnapshotID = snapID
	}

	if err := tx.Commit(ctx); err != nil {
		return AddToCloudResult{}, err
	}

	return AddToCloudResult{
		Definition:      mapDefinitionFromUpdateRow(defRow, true),
		Instance:        mapInstanceFromInsertRow(instRow),
		ShareSnapshotID: shareSnapshotID,
	}, nil
}

func addCloudKey(account, definitionID string) string {
	return account + ":" + definitionID
}

func resolveShareStatus(kind, shareStatus string) string {
	shareStatus = strings.TrimSpace(strings.ToLower(shareStatus))
	if kind != "custom" {
		return "private"
	}
	if shareStatus == "public" {
		return "public"
	}
	if shareStatus == "" || shareStatus == "private" {
		return "private"
	}
	return ""
}

func mergeDefinitionConfig(existing []byte, patch AddToCloudConfigPatch) ([]byte, error) {
	cfg := map[string]interface{}{}
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &cfg)
	}
	if patch.StartTime != "" {
		cfg["startTime"] = patch.StartTime
	}
	if patch.EndTime != "" {
		cfg["endTime"] = patch.EndTime
	}
	if len(patch.SchemeGroups) > 0 {
		cfg["schemeGroups"] = patch.SchemeGroups
	}
	if patch.RunMode != "" {
		setConfigSimBet(cfg, simBetFromClientRunMode(patch.RunMode))
	}
	if patch.SchemeFunds != "" {
		cfg["schemeFunds"] = patch.SchemeFunds
	}
	if patch.StopLoss != "" {
		cfg["stopLoss"] = patch.StopLoss
	}
	if patch.TakeProfit != "" {
		cfg["takeProfit"] = patch.TakeProfit
	}
	if patch.BetUnit != "" {
		cfg["betUnit"] = patch.BetUnit
	} else if patch.BetMode != "" && isBetUnitArtifact(patch.BetMode) {
		cfg["betUnit"] = patch.BetMode
	} else if patch.BetMode != "" {
		cfg["betMode"] = patch.BetMode
	}
	if patch.PlayTemplate != "" {
		cfg["playTemplate"] = patch.PlayTemplate
	}
	if patch.TypeID != "" {
		cfg["typeId"] = patch.TypeID
		cfg["playTypeId"] = patch.TypeID
	}
	if patch.SubID != "" {
		cfg["subId"] = patch.SubID
		cfg["subPlayId"] = patch.SubID
	}
	normalizeSchemeConfigBetFields(cfg)
	return json.Marshal(cfg)
}

func parseSchemeFunds(raw string, cfgBytes []byte) pgtype.Numeric {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		cfg := map[string]interface{}{}
		_ = json.Unmarshal(cfgBytes, &cfg)
		if v, ok := cfg["schemeFunds"]; ok {
			raw = fmt.Sprint(v)
		}
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || f < 0 {
		// 资金留空/非法时默认 0，避免写入 NOT NULL 的 fund_yuan 列失败
		f = 0
	}
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func extractPlayMethod(cfgBytes []byte) string {
	cfg := map[string]interface{}{}
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return ""
	}
	pm, _ := cfg["playMethod"].(string)
	pt, _ := cfg["playTypeId"].(string)
	if pt == "" {
		pt, _ = cfg["typeId"].(string)
	}
	sp, _ := cfg["subPlayId"].(string)
	if sp == "" {
		sp, _ = cfg["subId"].(string)
	}
	return PlayMethodDisplay(pm, pt, sp)
}

func mapDefinitionFromUpdateRow(row sqlcdb.UpdateSchemeDefinitionForCloudRow, hasInstance bool) Definition {
	return mapDefinitionFields(
		row.ID, row.Kind, row.SchemeName, row.LotteryCode, row.LotteryLabel,
		row.ShareStatus, row.Config, row.CreatedAt, row.UpdatedAt, hasInstance,
	)
}
