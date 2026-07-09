package schemes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

var ErrNoInstanceForFork = errors.New("scheme has no instance to fork")

type ForkToCloudResult struct {
	SourceDefinitionID string     `json:"sourceDefinitionId"`
	Definition         Definition `json:"definition"`
	Instance           Instance   `json:"instance"`
}

func (s *Service) ForkDefinitionToCloud(
	ctx context.Context,
	account, sourceDefinitionID string,
	patch AddToCloudConfigPatch,
) (ForkToCloudResult, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return ForkToCloudResult{}, ErrUnavailable
	}
	sourceDefinitionID = strings.TrimSpace(sourceDefinitionID)
	if sourceDefinitionID == "" {
		return ForkToCloudResult{}, ErrDefinitionNotFound
	}

	now := time.Now()
	if last, ok := addCloudLastAt.Load(addCloudKey(account, sourceDefinitionID)); ok {
		if now.Sub(time.UnixMilli(last.(int64))) < addCloudCooldown {
			return ForkToCloudResult{}, ErrAddCloudTooFast
		}
	}
	addCloudLastAt.Store(addCloudKey(account, sourceDefinitionID), now.UnixMilli())

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ForkToCloudResult{}, member.ErrNotFound
		}
		return ForkToCloudResult{}, err
	}

	src, err := s.q.GetSchemeDefinitionByIDAndMember(ctx, sqlcdb.GetSchemeDefinitionByIDAndMemberParams{
		ID:       sourceDefinitionID,
		MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ForkToCloudResult{}, ErrDefinitionNotFound
		}
		return ForkToCloudResult{}, err
	}

	if _, err := s.q.GetSchemeInstanceByDefinitionID(ctx, sourceDefinitionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ForkToCloudResult{}, ErrNoInstanceForFork
		}
		return ForkToCloudResult{}, err
	}

	cfgBytes, err := mergeDefinitionConfig(src.Config, patch)
	if err != nil {
		return ForkToCloudResult{}, err
	}

	existingNames, err := s.q.ListSchemeDefinitionNamesByMember(ctx, m.ID)
	if err != nil {
		return ForkToCloudResult{}, err
	}
	forkName := nextUniqueSchemeName(src.SchemeName, existingNames)

	simBet := configSimBet(cfgBytes)

	nowMs := now.UnixMilli()
	forkDefID := fmt.Sprintf("def-%d-%d", m.ID, nowMs)
	instID := fmt.Sprintf("inst-%d-%d", m.ID, nowMs+1)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ForkToCloudResult{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	defRow, err := qtx.InsertSchemeDefinition(ctx, sqlcdb.InsertSchemeDefinitionParams{
		ID:                forkDefID,
		MemberID:          m.ID,
		Kind:              src.Kind,
		SchemeName:        forkName,
		LotteryCode:       src.LotteryCode,
		LotteryLabel:      src.LotteryLabel,
		ShareStatus:       "private",
		ShareStatusLocked: true,
		SourceSnapshotID:  src.SourceSnapshotID,
		Config:            cfgBytes,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ForkToCloudResult{}, ErrNameDuplicate
		}
		return ForkToCloudResult{}, err
	}

	instRow, err := qtx.InsertSchemeInstance(ctx, sqlcdb.InsertSchemeInstanceParams{
		ID:           instID,
		DefinitionID: forkDefID,
		MemberID:     m.ID,
		Kind:         src.Kind,
		SchemeName:   forkName,
		LotteryCode:  src.LotteryCode,
		LotteryLabel: src.LotteryLabel,
		Status:       "pending",
		SimBet:       simBet,
	})
	if err != nil {
		return ForkToCloudResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ForkToCloudResult{}, err
	}

	return ForkToCloudResult{
		SourceDefinitionID: sourceDefinitionID,
		Definition:         mapDefinitionRow(defRow, true),
		Instance:           mapInstanceFromInsertRow(instRow),
	}, nil
}
