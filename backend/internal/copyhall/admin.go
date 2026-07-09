package copyhall

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemes"
)

var ErrInvalidBoard = errors.New("invalid copy hall board")

const defaultGlobalLotteryCode = "tron_ffc_1m"

type AdminRankingsState struct {
	Master   []RankSlot `json:"master"`
	Contrary []RankSlot `json:"contrary"`
}

type AdminRankingsBoard struct {
	Board string     `json:"board"`
	Slots []RankSlot `json:"slots"`
}

func (s *Service) AdminRankingsBoard(ctx context.Context, boardKind string) (AdminRankingsBoard, error) {
	if s == nil || s.q == nil {
		return AdminRankingsBoard{}, ErrUnavailable
	}
	boardKind = strings.TrimSpace(boardKind)
	if boardKind != "master" && boardKind != "contrary" {
		return AdminRankingsBoard{}, ErrInvalidQuery
	}

	rows, err := s.q.ListCopyHallRankSlots(ctx, boardKind)
	if err != nil {
		return AdminRankingsBoard{}, err
	}

	return AdminRankingsBoard{
		Board: boardKind,
		Slots: adminBoardSlotsFromDB(s.rankSlotsFromDBRows(ctx, rows)),
	}, nil
}

func (s *Service) AdminRankingsState(ctx context.Context) (AdminRankingsState, error) {
	if s == nil || s.q == nil {
		return AdminRankingsState{}, ErrUnavailable
	}
	masterBoard, err := s.AdminRankingsBoard(ctx, "master")
	if err != nil {
		return AdminRankingsState{}, err
	}
	contraryBoard, err := s.AdminRankingsBoard(ctx, "contrary")
	if err != nil {
		return AdminRankingsState{}, err
	}
	return AdminRankingsState{
		Master:   masterBoard.Slots,
		Contrary: contraryBoard.Slots,
	}, nil
}

func (s *Service) AdminSaveBoard(ctx context.Context, boardKind string, slots []RankSlot) (AdminRankingsBoard, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return AdminRankingsBoard{}, ErrUnavailable
	}
	boardKind = strings.TrimSpace(boardKind)
	if boardKind != "master" && boardKind != "contrary" {
		return AdminRankingsBoard{}, ErrInvalidQuery
	}
	normalized, err := validateAdminSaveSlots(slots)
	if err != nil {
		return AdminRankingsBoard{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AdminRankingsBoard{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	if err := qtx.DeleteCopyHallBoardSlots(ctx, boardKind); err != nil {
		return AdminRankingsBoard{}, err
	}

	for _, slot := range normalized {
		if strings.TrimSpace(slot.SchemeID) == "" {
			continue
		}
		lotteryCode := strings.TrimSpace(slot.LotteryCode)
		if lotteryCode == "" {
			return AdminRankingsBoard{}, fmt.Errorf("%w: lotteryCode required when schemeId set", ErrInvalidBoard)
		}
		if err := s.validateOnSaleLottery(ctx, lotteryCode); err != nil {
			return AdminRankingsBoard{}, err
		}
		playTypeID := slot.PlayTypeID
		subPlayID := slot.SubPlayID
		if playTypeID == "" {
			playTypeID, subPlayID = schemes.PlayIDsFromMethod(slot.PlayMethod)
		}
		displayPlayMethod := schemes.PlayMethodDisplay(slot.PlayMethod, playTypeID, subPlayID)
		if err := qtx.UpsertCopyHallRankSlot(ctx, sqlcdb.UpsertCopyHallRankSlotParams{
			LotteryCode: lotteryCode,
			BoardKind:   boardKind,
			Rank:        int32(slot.Rank),
			SchemeID:    slot.SchemeID,
			SchemeName:  slot.SchemeName,
			PlayMethod:  displayPlayMethod,
			PlayTypeID:  playTypeID,
			SubPlayID:   subPlayID,
		}); err != nil {
			return AdminRankingsBoard{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminRankingsBoard{}, err
	}
	return s.AdminRankingsBoard(ctx, boardKind)
}

func (s *Service) AdminResetBoard(ctx context.Context, boardKind string) (AdminRankingsBoard, error) {
	slots := defaultBoardSlots(boardKind)
	return s.AdminSaveBoard(ctx, boardKind, slots)
}

func (s *Service) AdminResetAll(ctx context.Context) (AdminRankingsState, error) {
	if s == nil || s.q == nil || s.pool == nil {
		return AdminRankingsState{}, ErrUnavailable
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AdminRankingsState{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	if err := qtx.DeleteAllCopyHallRankSlots(ctx); err != nil {
		return AdminRankingsState{}, err
	}
	for _, kind := range []struct {
		name  string
		slots []RankSlot
	}{
		{"master", defaultBoardSlots("master")},
		{"contrary", defaultBoardSlots("contrary")},
	} {
		for _, slot := range kind.slots {
			playTypeID := slot.PlayTypeID
			subPlayID := slot.SubPlayID
			if playTypeID == "" {
				playTypeID, subPlayID = schemes.PlayIDsFromMethod(slot.PlayMethod)
			}
			if err := qtx.UpsertCopyHallRankSlot(ctx, sqlcdb.UpsertCopyHallRankSlotParams{
				LotteryCode: slot.LotteryCode,
				BoardKind:   kind.name,
				Rank:        int32(slot.Rank),
				SchemeID:    slot.SchemeID,
				SchemeName:  slot.SchemeName,
				PlayMethod:  slot.PlayMethod,
				PlayTypeID:  playTypeID,
				SubPlayID:   subPlayID,
			}); err != nil {
				return AdminRankingsState{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminRankingsState{}, err
	}
	return s.AdminRankingsState(ctx)
}

// adminBoardSlotsFromDB 管理端只展示库内真实配置；缺位补空行，不回填演示 mock。
func adminBoardSlotsFromDB(slots []RankSlot) []RankSlot {
	byRank := map[int]RankSlot{}
	for _, slot := range slots {
		if slot.Rank < 1 || slot.Rank > 10 {
			continue
		}
		byRank[slot.Rank] = slot
	}
	out := make([]RankSlot, 10)
	for i := range out {
		rank := i + 1
		if slot, ok := byRank[rank]; ok {
			out[i] = slot
			out[i].Rank = rank
			continue
		}
		out[i] = RankSlot{Rank: rank}
	}
	return out
}

func validateAdminSaveSlots(slots []RankSlot) ([]RankSlot, error) {
	if len(slots) != 10 {
		return nil, fmt.Errorf("%w: slots must contain 10 items", ErrInvalidBoard)
	}
	byRank := map[int]RankSlot{}
	for _, slot := range slots {
		if slot.Rank < 1 || slot.Rank > 10 {
			return nil, fmt.Errorf("%w: invalid rank", ErrInvalidBoard)
		}
		if _, ok := byRank[slot.Rank]; ok {
			return nil, fmt.Errorf("%w: duplicate rank", ErrInvalidBoard)
		}
		byRank[slot.Rank] = slot
	}
	out := make([]RankSlot, 10)
	seenScheme := map[string]int{}
	for i := range out {
		rank := i + 1
		slot, ok := byRank[rank]
		if !ok || strings.TrimSpace(slot.SchemeID) == "" {
			out[i] = RankSlot{Rank: rank}
			continue
		}
		if strings.TrimSpace(slot.SchemeName) == "" {
			return nil, fmt.Errorf("%w: schemeName required when schemeId set", ErrInvalidBoard)
		}
		if strings.TrimSpace(slot.LotteryCode) == "" {
			return nil, fmt.Errorf("%w: lotteryCode required when schemeId set", ErrInvalidBoard)
		}
		schemeID := strings.TrimSpace(slot.SchemeID)
		if prevRank, dup := seenScheme[schemeID]; dup {
			_ = prevRank
			out[i] = RankSlot{Rank: rank}
			continue
		}
		seenScheme[schemeID] = rank
		out[i] = slot
		out[i].Rank = rank
	}
	return out, nil
}

func validateBoardSlots(slots []RankSlot, boardKind string) ([]RankSlot, error) {
	if len(slots) != 10 {
		return nil, fmt.Errorf("%w: slots must contain 10 items", ErrInvalidBoard)
	}
	normalized := normalizeRankSlots(slots, boardKind)
	seen := map[int]struct{}{}
	for _, slot := range normalized {
		if slot.SchemeID == "" || slot.SchemeName == "" {
			return nil, fmt.Errorf("%w: schemeId and schemeName required", ErrInvalidBoard)
		}
		if _, ok := seen[slot.Rank]; ok {
			return nil, fmt.Errorf("%w: duplicate rank", ErrInvalidBoard)
		}
		seen[slot.Rank] = struct{}{}
	}
	return normalized, nil
}

func normalizeRankSlots(slots []RankSlot, boardKind string) []RankSlot {
	sorted := append([]RankSlot(nil), slots...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Rank < sorted[j].Rank })
	if len(sorted) > 10 {
		sorted = sorted[:10]
	}
	def := defaultBoardSlots(boardKind)
	for len(sorted) < 10 {
		rank := len(sorted) + 1
		fallback := def[rank-1]
		sorted = append(sorted, fallback)
	}
	out := make([]RankSlot, 10)
	for i := range out {
		out[i] = sorted[i]
		out[i].Rank = i + 1
		if out[i].SchemeID == "" {
			out[i].SchemeID = fallbackSchemeID(boardKind, i+1)
		}
		if out[i].LotteryCode == "" {
			out[i].LotteryCode = def[i].LotteryCode
		}
		if out[i].PlayMethod == "" {
			out[i].PlayMethod = def[i].PlayMethod
		}
		if out[i].PlayTypeID == "" {
			out[i].PlayTypeID = def[i].PlayTypeID
		}
		if out[i].SubPlayID == "" && out[i].PlayTypeID == def[i].PlayTypeID {
			out[i].SubPlayID = def[i].SubPlayID
		}
	}
	return out
}

func fallbackSchemeID(boardKind string, rank int) string {
	return defaultSchemeID(boardKind, rank)
}
