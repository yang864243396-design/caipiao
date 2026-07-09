package copyhall

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemes"
)

var (
	ErrUnavailable  = errors.New("copy hall service unavailable")
	ErrInvalidQuery = errors.New("invalid copy hall query")
)

type Service struct {
	q    *sqlcdb.Queries
	pool *db.Pool
}

func NewService(pool *db.Pool) *Service {
	if pool == nil {
		return nil
	}
	return &Service{q: sqlcdb.New(pool), pool: pool}
}

type RankSlot struct {
	Rank         int    `json:"rank"`
	LotteryCode  string `json:"lotteryCode"`
	LotteryLabel string `json:"lotteryLabel,omitempty"`
	SchemeID     string `json:"schemeId"`
	SchemeName   string `json:"schemeName"`
	PlayMethod   string `json:"playMethod"`
	PlayTypeID   string `json:"playTypeId"`
	SubPlayID    string `json:"subPlayId"`
}

type RankingsResult struct {
	LotteryCode string     `json:"lotteryCode,omitempty"`
	Board       string     `json:"board"`
	Slots       []RankSlot `json:"slots"`
}

func (s *Service) ResolveLotteryLabel(ctx context.Context, code string) string {
	row, err := s.q.GetLotteryCatalogByCode(ctx, code)
	if err == nil && row.DisplayName != "" {
		return row.DisplayName
	}
	return LotteryLabel(code)
}

func (s *Service) isOnSaleLottery(ctx context.Context, code string) bool {
	row, err := s.q.GetLotteryCatalogByCode(ctx, code)
	return err == nil && row.SaleStatus == "on_sale"
}

func (s *Service) rankSlotsFromDBRows(ctx context.Context, rows []sqlcdb.ListCopyHallRankSlotsRow) []RankSlot {
	slots := make([]RankSlot, 0, len(rows))
	for _, row := range rows {
		slots = append(slots, RankSlot{
			Rank:         int(row.Rank),
			LotteryCode:  row.LotteryCode,
			LotteryLabel: s.ResolveLotteryLabel(ctx, row.LotteryCode),
			SchemeID:     row.SchemeID,
			SchemeName:   row.SchemeName,
			PlayMethod:   schemes.PlayMethodDisplay(row.PlayMethod, row.PlayTypeID, row.SubPlayID),
			PlayTypeID:   row.PlayTypeID,
			SubPlayID:    row.SubPlayID,
		})
	}
	return slots
}

func (s *Service) Rankings(ctx context.Context, lotteryCode, board string) (RankingsResult, error) {
	if s == nil || s.q == nil {
		return RankingsResult{}, ErrUnavailable
	}
	if lotteryCode != "" && !s.isOnSaleLottery(ctx, lotteryCode) {
		return RankingsResult{}, ErrInvalidQuery
	}
	if board != "master" && board != "contrary" {
		return RankingsResult{}, ErrInvalidQuery
	}

	rows, err := s.q.ListCopyHallRankSlots(ctx, board)
	if err != nil {
		return RankingsResult{}, err
	}

	slots := s.rankSlotsFromDBRows(ctx, rows)
	if lotteryCode != "" {
		filtered := make([]RankSlot, 0, len(slots))
		for _, slot := range slots {
			if slot.LotteryCode == lotteryCode {
				filtered = append(filtered, slot)
			}
		}
		slots = filtered
	}
	return RankingsResult{
		LotteryCode: lotteryCode,
		Board:       board,
		Slots:       slots,
	}, nil
}

func (s *Service) validateOnSaleLottery(ctx context.Context, code string) error {
	if !s.isOnSaleLottery(ctx, code) {
		if _, err := s.q.GetLotteryCatalogByCode(ctx, code); errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidQuery
		}
		return ErrInvalidQuery
	}
	return nil
}
