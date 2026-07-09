package schemes

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
)

func (s *Service) AdminUpdateShareSnapshot(ctx context.Context, snapshotID string, in AdminCreateShareSnapshotInput) (ShareSnapshot, error) {
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

	createIn := CreateDefinitionInput{
		Kind:        "custom",
		SchemeName:  strings.TrimSpace(in.SchemeName),
		LotteryCode: strings.TrimSpace(in.LotteryCode),
		RunTypeID:   strings.TrimSpace(in.RunTypeID),
		PlayTypeID:  strings.TrimSpace(in.PlayTypeID),
		SubPlayID:   strings.TrimSpace(in.SubPlayID),
	}
	normalizeCreateInput(&createIn)
	if createIn.SchemeName == "" {
		return ShareSnapshot{}, fmt.Errorf("%w: schemeName 不能为空", ErrInvalidCreateRequest)
	}
	if err := validateCreateInput(createIn); err != nil {
		return ShareSnapshot{}, err
	}
	if err := s.validateCreateRunTypePlay(ctx, createIn); err != nil {
		return ShareSnapshot{}, err
	}

	lotteryLabel, err := s.lotteryLabel(ctx, createIn.LotteryCode)
	if err != nil {
		return ShareSnapshot{}, err
	}

	baseCfg, err := s.buildCreateDefinitionConfig(ctx, createIn, createIn.SchemeName)
	if err != nil {
		return ShareSnapshot{}, err
	}
	cfgBytes, err := mergeAdminShareSnapshotConfigUpdate(cur.Config, baseCfg, in.Patch, in.Extra)
	if err != nil {
		return ShareSnapshot{}, err
	}

	playMethod := extractPlayMethod(cfgBytes)
	fundYuan := parseSchemeFunds(in.Patch.SchemeFunds, cfgBytes)

	row, err := s.q.UpdateSchemeShareSnapshotAdmin(ctx, sqlcdb.UpdateSchemeShareSnapshotAdminParams{
		ID:           snapshotID,
		SchemeName:   createIn.SchemeName,
		LotteryCode:  createIn.LotteryCode,
		LotteryLabel: lotteryLabel,
		PlayMethod:   playMethod,
		FundYuan:     fundYuan,
		Config:       cfgBytes,
	})
	if err != nil {
		return ShareSnapshot{}, err
	}
	return mapShareSnapshotRow(row), nil
}
