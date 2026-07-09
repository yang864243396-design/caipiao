package games

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

var (
	ErrCatalogNotEditable = errors.New("lottery catalog editable only in maintenance")
	ErrCatalogInvalidPatch = errors.New("invalid lottery catalog patch")
	ErrLotteryMaintenance  = errors.New("lottery in maintenance")
)

type PatchCatalogInput struct {
	DisplayName         string
	OutboundLotteryCode string
	SortOrder           int
	SaleStatus          string
	EnterMaintenance    bool
}

func (s *Service) AdminPatchCatalog(ctx context.Context, code string, in PatchCatalogInput) (CatalogRow, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return CatalogRow{}, ErrUnavailable
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return CatalogRow{}, ErrCatalogNotFound
	}

	current, err := s.q.GetLotteryCatalogByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CatalogRow{}, ErrCatalogNotFound
		}
		return CatalogRow{}, err
	}

	switch sqlcdb.SaleStatusString(current.SaleStatus) {
	case "on_sale":
		if !in.EnterMaintenance || (strings.TrimSpace(in.SaleStatus) != "" && in.SaleStatus != "maintenance") {
			return CatalogRow{}, ErrCatalogNotEditable
		}
		updated, err := s.q.SetLotteryCatalogMaintenance(ctx, code)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return CatalogRow{}, ErrCatalogNotEditable
			}
			return CatalogRow{}, err
		}
		return mapCatalogRow(sqlcdb.LotteryCatalogRowFromMaintenance(updated)), nil
	case "maintenance":
		patch, err := normalizeMaintenancePatch(sqlcdb.LotteryCatalogRowFromByCode(current), in)
		if err != nil {
			return CatalogRow{}, err
		}
		return s.applyMaintenancePatch(ctx, code, current.DisplayName, patch)
	default:
		return CatalogRow{}, ErrCatalogInvalidPatch
	}
}

func normalizeMaintenancePatch(current sqlcdb.LotteryCatalogRow, in PatchCatalogInput) (PatchCatalogInput, error) {
	out := PatchCatalogInput{
		DisplayName:         strings.TrimSpace(in.DisplayName),
		OutboundLotteryCode: strings.TrimSpace(in.OutboundLotteryCode),
		SortOrder:           in.SortOrder,
		SaleStatus:          strings.TrimSpace(in.SaleStatus),
	}
	if out.DisplayName == "" {
		out.DisplayName = current.DisplayName
	}
	if out.OutboundLotteryCode == "" {
		out.OutboundLotteryCode = textVal(current.OutboundLotteryCode)
	}
	if out.OutboundLotteryCode == "" {
		out.OutboundLotteryCode = current.Code
	}
	if out.SortOrder <= 0 {
		out.SortOrder = int(current.SortOrder)
	}
	if out.SaleStatus == "" {
		out.SaleStatus = "maintenance"
	}
	if out.SaleStatus != "on_sale" && out.SaleStatus != "maintenance" {
		return PatchCatalogInput{}, fmt.Errorf("%w: saleStatus", ErrCatalogInvalidPatch)
	}
	if out.DisplayName == "" {
		return PatchCatalogInput{}, fmt.Errorf("%w: displayName", ErrCatalogInvalidPatch)
	}
	return out, nil
}

func (s *Service) applyMaintenancePatch(ctx context.Context, code, oldDisplayName string, in PatchCatalogInput) (CatalogRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CatalogRow{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	updated, err := qtx.PatchLotteryCatalogMaintenance(ctx, sqlcdb.PatchLotteryCatalogMaintenanceParams{
		Code:        code,
		DisplayName: in.DisplayName,
		OutboundLotteryCode: pgtype.Text{
			String: in.OutboundLotteryCode,
			Valid:  strings.TrimSpace(in.OutboundLotteryCode) != "",
		},
		SortOrder: int32(in.SortOrder),
		Column5:   in.SaleStatus,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CatalogRow{}, ErrCatalogNotEditable
		}
		return CatalogRow{}, err
	}

	if strings.TrimSpace(oldDisplayName) != strings.TrimSpace(in.DisplayName) {
		if err := syncLotteryLabels(ctx, qtx, code, in.DisplayName); err != nil {
			return CatalogRow{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CatalogRow{}, err
	}
	return mapCatalogRow(sqlcdb.LotteryCatalogRowFromPatch(updated)), nil
}

func syncLotteryLabels(ctx context.Context, q *sqlcdb.Queries, code, label string) error {
	if err := q.UpdateSchemeDefinitionsLotteryLabel(ctx, sqlcdb.UpdateSchemeDefinitionsLotteryLabelParams{
		LotteryCode: code, LotteryLabel: label,
	}); err != nil {
		return err
	}
	if err := q.UpdateSchemeInstancesLotteryLabel(ctx, sqlcdb.UpdateSchemeInstancesLotteryLabelParams{
		LotteryCode: code, LotteryLabel: label,
	}); err != nil {
		return err
	}
	return q.UpdateSchemeShareSnapshotsLotteryLabel(ctx, sqlcdb.UpdateSchemeShareSnapshotsLotteryLabelParams{
		LotteryCode: code, LotteryLabel: label,
	})
}

type LotteryRouteStatus struct {
	Code       string `json:"code"`
	Exists     bool   `json:"exists"`
	Legacy     bool   `json:"legacy"`
	SaleStatus string `json:"saleStatus,omitempty"`
}

func (s *Service) PublicLotteryRouteStatus(ctx context.Context, code string) LotteryRouteStatus {
	code = strings.TrimSpace(code)
	if code == "" {
		return LotteryRouteStatus{Code: code, Exists: false}
	}
	for _, legacy := range LegacyLotteryCodes {
		if legacy == code {
			return LotteryRouteStatus{Code: code, Exists: false, Legacy: true}
		}
	}
	row, err := s.q.GetLotteryCatalogByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LotteryRouteStatus{Code: code, Exists: false}
		}
		return LotteryRouteStatus{Code: code, Exists: false}
	}
	return LotteryRouteStatus{
		Code:       row.Code,
		Exists:     true,
		SaleStatus: sqlcdb.SaleStatusString(row.SaleStatus),
	}
}

func (s *Service) MemberLotteryFilterOptions(ctx context.Context) ([]CatalogRow, error) {
	return s.AdminListCatalog(ctx)
}
