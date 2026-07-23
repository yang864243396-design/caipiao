package schemes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

var ErrLotteryOptionsNotFound = errors.New("lottery scheme options not found")

type SchemeOptionItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type LotterySchemeOptionsResult struct {
	LotteryCode string             `json:"lotteryCode"`
	RunTypes    []SchemeOptionItem `json:"runTypes"`
	PlayTypes   []SchemeOptionItem `json:"playTypes"`
	SubPlays    []SchemeOptionItem `json:"subPlays"`
}

func (s *Service) GetSchemeOptions(ctx context.Context, lotteryCode string) (LotterySchemeOptionsResult, error) {
	if s == nil || s.q == nil {
		return LotterySchemeOptionsResult{}, ErrUnavailable
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	if lotteryCode == "" {
		return LotterySchemeOptionsResult{}, ErrLotteryOptionsNotFound
	}

	row, err := s.q.GetLotterySchemeOptionSet(ctx, lotteryCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			row, err = s.q.GetLotterySchemeOptionSet(ctx, "_default")
		}
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return LotterySchemeOptionsResult{}, ErrLotteryOptionsNotFound
			}
			return LotterySchemeOptionsResult{}, err
		}
	}

	runTypes, err := parseOptionItems(row.RunTypes)
	if err != nil {
		return LotterySchemeOptionsResult{}, err
	}
	runTypes = normalizeRunTypeOptionLabels(runTypes)
	playTypes, err := parseOptionItems(row.PlayTypes)
	if err != nil {
		return LotterySchemeOptionsResult{}, err
	}
	subPlays, err := parseOptionItems(row.SubPlays)
	if err != nil {
		return LotterySchemeOptionsResult{}, err
	}

	code := lotteryCode
	if row.LotteryCode != "_default" {
		code = row.LotteryCode
	}

	return LotterySchemeOptionsResult{
		LotteryCode: code,
		RunTypes:    runTypes,
		PlayTypes:   playTypes,
		SubPlays:    subPlays,
	}, nil
}

// normalizeRunTypeOptionLabels 以 RunTypeLabels 为准覆盖 DB 种子中的旧展示名。
func normalizeRunTypeOptionLabels(items []SchemeOptionItem) []SchemeOptionItem {
	if len(items) == 0 {
		return items
	}
	out := make([]SchemeOptionItem, len(items))
	for i, it := range items {
		out[i] = it
		if label, ok := RunTypeLabels[strings.TrimSpace(it.Value)]; ok {
			out[i].Label = label
		}
	}
	return out
}

func parseOptionItems(raw []byte) ([]SchemeOptionItem, error) {
	if len(raw) == 0 {
		return []SchemeOptionItem{}, nil
	}
	var items []SchemeOptionItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	if items == nil {
		return []SchemeOptionItem{}, nil
	}
	return items, nil
}
