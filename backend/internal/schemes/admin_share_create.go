package schemes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

type AdminCreateShareSnapshotInput struct {
	SchemeName  string
	LotteryCode string
	RunTypeID   string
	PlayTypeID  string
	SubPlayID   string
	Patch       AddToCloudConfigPatch
	Extra       AdminShareConfigExtra
}

func (s *Service) AdminCreateShareSnapshot(ctx context.Context, in AdminCreateShareSnapshotInput) (ShareSnapshot, error) {
	if s == nil || s.q == nil {
		return ShareSnapshot{}, ErrUnavailable
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
	cfgBytes, err := mergeAdminShareSnapshotConfig(baseCfg, in.Patch, in.Extra)
	if err != nil {
		return ShareSnapshot{}, err
	}

	playMethod := extractPlayMethod(cfgBytes)
	fundYuan := parseSchemeFunds(in.Patch.SchemeFunds, cfgBytes)
	snapID := fmt.Sprintf("SD%d", time.Now().UnixMilli())

	row, err := s.q.InsertSchemeShareSnapshot(ctx, sqlcdb.InsertSchemeShareSnapshotParams{
		ID:           snapID,
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
