package games

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

type PublicLotteryRow struct {
	Code                string `json:"code"`
	DisplayName         string `json:"displayName"`
	CategoryCode        string `json:"categoryCode"`
	PlayTemplate        string `json:"playTemplate"`
	BallCount           int    `json:"ballCount"`
	DrawInterval        string `json:"drawInterval,omitempty"`
	SortOrder           int    `json:"sortOrder"`
	OutboundLotteryCode string `json:"outboundLotteryCode"`
}

type PlayTypeNode struct {
	TypeID    string        `json:"typeId"`
	Label     string        `json:"label"`
	SortOrder int           `json:"sortOrder"`
	PanelType string        `json:"panelType,omitempty"`
	SubPlays  []SubPlayNode `json:"subPlays"`
}

type SubPlayNode struct {
	SubID            string          `json:"subId"`
	Label            string          `json:"label"`
	SortOrder        int             `json:"sortOrder"`
	BetMode          string          `json:"betMode,omitempty"`
	OutboundPlayCode string          `json:"outboundPlayCode"`
	SegmentRule      json.RawMessage `json:"segmentRule,omitempty"`
}

type PlayTreeResponse struct {
	LotteryCode    string         `json:"lotteryCode"`
	DisplayName    string         `json:"displayName"`
	PlayTemplate   string         `json:"playTemplate"`
	RulesTypeName  string         `json:"rulesTypeName,omitempty"`
	PlayTypes      []PlayTypeNode `json:"playTypes"`
}

func (s *Service) PublicListLotteries(ctx context.Context) ([]PublicLotteryRow, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListLotteryCatalogOnSale(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PublicLotteryRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapPublicLotteryRow(sqlcdb.LotteryCatalogRowFromOnSale(row)))
	}
	return out, nil
}

func (s *Service) PublicPlayTree(ctx context.Context, code string) (PlayTreeResponse, error) {
	if s == nil || s.q == nil {
		return PlayTreeResponse{}, ErrUnavailable
	}
	cat, err := s.q.GetLotteryCatalogByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PlayTreeResponse{}, ErrLotteryNotFound
		}
		return PlayTreeResponse{}, err
	}
	if cat.SaleStatus == "maintenance" {
		return PlayTreeResponse{}, ErrLotteryMaintenance
	}
	if cat.SaleStatus != "on_sale" {
		return PlayTreeResponse{}, ErrLotteryNotFound
	}
	template := textVal(cat.PlayTemplate)
	if template == "" {
		return PlayTreeResponse{}, ErrLotteryNotFound
	}
	types, err := s.q.ListPlayTypesByTemplate(ctx, template)
	if err != nil {
		return PlayTreeResponse{}, err
	}
	subs, err := s.q.ListSubPlaysByTemplate(ctx, template)
	if err != nil {
		return PlayTreeResponse{}, err
	}
	rulesTypeName := template
	if tplRows, terr := s.q.ListPlayTemplates(ctx); terr == nil {
		for _, t := range tplRows {
			if t.Code == template {
				rulesTypeName = t.Label
				break
			}
		}
	}
	subsByType := make(map[string][]SubPlayNode)
	for _, sub := range subs {
		subsByType[sub.TypeID] = append(subsByType[sub.TypeID], SubPlayNode{
			SubID:            sub.SubID,
			Label:            sub.Label,
			SortOrder:        int(sub.SortOrder),
			BetMode:          textVal(sub.BetMode),
			OutboundPlayCode: textVal(sub.OutboundPlayCode),
			SegmentRule:      sub.SegmentRule,
		})
	}
	nodes := make([]PlayTypeNode, 0, len(types))
	for _, t := range types {
		nodes = append(nodes, PlayTypeNode{
			TypeID:    t.TypeID,
			Label:     t.Label,
			SortOrder: int(t.SortOrder),
			PanelType: textVal(t.PanelType),
			SubPlays:  subPlaysOrEmpty(subsByType[t.TypeID]),
		})
	}
	return PlayTreeResponse{
		LotteryCode:   cat.Code,
		DisplayName:   cat.DisplayName,
		PlayTemplate:  template,
		RulesTypeName: rulesTypeName,
		PlayTypes:     nodes,
	}, nil
}

func mapPublicLotteryRow(row sqlcdb.LotteryCatalogRow) PublicLotteryRow {
	ball := 0
	if row.BallCount.Valid {
		ball = int(row.BallCount.Int16)
	}
	return PublicLotteryRow{
		Code:                row.Code,
		DisplayName:         row.DisplayName,
		CategoryCode:        textVal(row.CategoryCode),
		PlayTemplate:        textVal(row.PlayTemplate),
		BallCount:           ball,
		DrawInterval:        textVal(row.DrawInterval),
		SortOrder:           int(row.SortOrder),
		OutboundLotteryCode: textVal(row.OutboundLotteryCode),
	}
}

func subPlaysOrEmpty(nodes []SubPlayNode) []SubPlayNode {
	if nodes == nil {
		return []SubPlayNode{}
	}
	return nodes
}

func textVal(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}
