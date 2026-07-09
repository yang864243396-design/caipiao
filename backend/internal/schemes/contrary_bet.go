package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

type ContraryBetInput struct {
	LotteryCode        string
	PlanInverseNumbers string
	PlayMethod         string
	PlayTemplate       string
	TypeID             string
	SubID              string
}

func (s *Service) ContraryBet(ctx context.Context, account string, input ContraryBetInput) (ShareFollowActionResult, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return ShareFollowActionResult{}, ErrUnavailable
	}
	input.LotteryCode = strings.TrimSpace(input.LotteryCode)
	input.PlanInverseNumbers = strings.TrimSpace(input.PlanInverseNumbers)
	input.PlayMethod = strings.TrimSpace(input.PlayMethod)
	if input.LotteryCode == "" {
		return ShareFollowActionResult{}, fmt.Errorf("%w: lotteryCode 不能为空", ErrInvalidCreateRequest)
	}
	if input.PlanInverseNumbers == "" {
		return ShareFollowActionResult{}, fmt.Errorf("%w: planInverseNumbers 不能为空", ErrInvalidCreateRequest)
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShareFollowActionResult{}, member.ErrNotFound
		}
		return ShareFollowActionResult{}, err
	}

	existingNames, err := s.q.ListSchemeDefinitionNamesByMember(ctx, m.ID)
	if err != nil {
		return ShareFollowActionResult{}, err
	}

	baseName := "反买方案"
	if input.PlayMethod != "" {
		baseName = "反买-" + input.PlayMethod
	}
	schemeName := nextUniqueSchemeName(baseName, existingNames)

	lotteryLabel, err := s.lotteryLabel(ctx, input.LotteryCode)
	if err != nil {
		return ShareFollowActionResult{}, err
	}

	playTypeID := strings.TrimSpace(input.TypeID)
	subPlayID := strings.TrimSpace(input.SubID)
	if playTypeID == "" {
		playTypeID = "dingwei"
	}
	if subPlayID == "" {
		subPlayID = "dingwei_wan"
	}
	cfg := map[string]interface{}{
		"schemeName":         schemeName,
		"lotteryCode":        input.LotteryCode,
		"planInverseNumbers": input.PlanInverseNumbers,
		"runTypeId":          "run_std",
		"playTypeId":         playTypeID,
		"subPlayId":          subPlayID,
		"typeId":             playTypeID,
		"subId":              subPlayID,
	}
	if pt := strings.TrimSpace(input.PlayTemplate); pt != "" {
		cfg["playTemplate"] = pt
	}
	if input.PlayMethod != "" {
		cfg["playMethod"] = input.PlayMethod
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
		Kind:               "contrary",
		SchemeName:         schemeName,
		LotteryCode:        input.LotteryCode,
		LotteryLabel:       lotteryLabel,
		ShareStatus:        "private",
		ShareStatusLocked:  true,
		SourceSnapshotID:   pgtype.Text{},
		Config:             cfgBytes,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ShareFollowActionResult{}, ErrNameDuplicate
		}
		return ShareFollowActionResult{}, err
	}
	instRow, err := qtx.InsertSchemeInstance(ctx, sqlcdb.InsertSchemeInstanceParams{
		ID:           instID,
		DefinitionID: defID,
		MemberID:     m.ID,
		Kind:         "contrary",
		SchemeName:   schemeName,
		LotteryCode:  input.LotteryCode,
		LotteryLabel: lotteryLabel,
		Status:       "running",
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
