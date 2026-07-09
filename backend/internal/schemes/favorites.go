package schemes

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/timeutil"
)

// FavoriteRow 跟单大厅方案收藏（v8 §3.6 / P6）。
type FavoriteRow struct {
	SnapshotID   string `json:"snapshotId"`
	SchemeName   string `json:"schemeName"`
	LotteryCode  string `json:"lotteryCode"`
	LotteryLabel string `json:"lotteryLabel"`
	PlayMethod   string `json:"playMethod"`
	FavoredAt    string `json:"favoredAt"`
}

func (s *Service) ListFavorites(ctx context.Context, account string) ([]FavoriteRow, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, member.ErrNotFound
		}
		return nil, err
	}
	rows, err := s.q.ListMemberSchemeFavorites(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	out := make([]FavoriteRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, FavoriteRow{
			SnapshotID:   r.SnapshotID,
			SchemeName:   r.SchemeName,
			LotteryCode:  r.LotteryCode,
			LotteryLabel: r.LotteryLabel,
			PlayMethod:   r.PlayMethod,
			FavoredAt:    timeutil.FormatDisplayCST(r.CreatedAt.Time),
		})
	}
	return out, nil
}

func (s *Service) AddFavorite(ctx context.Context, account, snapshotID string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return ErrSnapshotNotFound
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return member.ErrNotFound
		}
		return err
	}
	if _, err := s.q.GetSchemeShareSnapshotByID(ctx, snapshotID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSnapshotNotFound
		}
		return err
	}
	_, err = s.q.InsertMemberSchemeFavorite(ctx, sqlcdb.InsertMemberSchemeFavoriteParams{
		MemberID:   m.ID,
		SnapshotID: snapshotID,
	})
	return err
}

func (s *Service) RemoveFavorite(ctx context.Context, account, snapshotID string) error {
	if s == nil || s.q == nil {
		return ErrUnavailable
	}
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return ErrSnapshotNotFound
	}
	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return member.ErrNotFound
		}
		return err
	}
	_, err = s.q.DeleteMemberSchemeFavorite(ctx, sqlcdb.DeleteMemberSchemeFavoriteParams{
		MemberID:   m.ID,
		SnapshotID: snapshotID,
	})
	return err
}
