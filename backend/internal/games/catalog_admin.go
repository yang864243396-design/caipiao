package games

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
)

var ErrCatalogNotFound = errors.New("lottery catalog row not found")

type CatalogRow struct {
	Code                string `json:"code"`
	DisplayName         string `json:"displayName"`
	CategoryCode        string `json:"categoryCode,omitempty"`
	PlayTemplate        string `json:"playTemplate,omitempty"`
	BallCount           int    `json:"ballCount,omitempty"`
	DrawInterval        string `json:"drawInterval,omitempty"`
	SortOrder           int    `json:"sortOrder"`
	OnSale              bool   `json:"onSale"`
	SaleStatus          string `json:"saleStatus"`
	OutboundLotteryCode string `json:"outboundLotteryCode,omitempty"`
}

type PlayTemplateRow struct {
	Code    string `json:"code"`
	Label   string `json:"label"`
	Version int    `json:"version"`
}

type AdminPlayTreeRow struct {
	TemplateCode string         `json:"templateCode"`
	PlayTypes    []PlayTypeNode `json:"playTypes"`
}

func (s *Service) AdminListCatalog(ctx context.Context) ([]CatalogRow, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListLotteryCatalog(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CatalogRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapCatalogRow(sqlcdb.LotteryCatalogRowFromList(row)))
	}
	return out, nil
}

func (s *Service) AdminListPlayTemplates(ctx context.Context) ([]PlayTemplateRow, error) {
	if s == nil || s.q == nil {
		return nil, ErrUnavailable
	}
	rows, err := s.q.ListPlayTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PlayTemplateRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, PlayTemplateRow{
			Code:    row.Code,
			Label:   row.Label,
			Version: int(row.Version),
		})
	}
	return out, nil
}

func (s *Service) AdminPlayTree(ctx context.Context, templateCode string) (AdminPlayTreeRow, error) {
	if s == nil || s.q == nil {
		return AdminPlayTreeRow{}, ErrUnavailable
	}
	types, err := s.q.ListPlayTypesByTemplate(ctx, templateCode)
	if err != nil {
		return AdminPlayTreeRow{}, err
	}
	subs, err := s.q.ListSubPlaysByTemplate(ctx, templateCode)
	if err != nil {
		return AdminPlayTreeRow{}, err
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
	return AdminPlayTreeRow{TemplateCode: templateCode, PlayTypes: nodes}, nil
}

func mapCatalogRow(row sqlcdb.LotteryCatalogRow) CatalogRow {
	ball := 0
	if row.BallCount.Valid {
		ball = int(row.BallCount.Int16)
	}
	return CatalogRow{
		Code:                row.Code,
		DisplayName:         row.DisplayName,
		CategoryCode:        textVal(row.CategoryCode),
		PlayTemplate:        textVal(row.PlayTemplate),
		BallCount:           ball,
		DrawInterval:        textVal(row.DrawInterval),
		SortOrder:           int(row.SortOrder),
		OnSale:              row.OnSale,
		SaleStatus:          row.SaleStatus,
		OutboundLotteryCode: textVal(row.OutboundLotteryCode),
	}
}

func (s *Service) GetCatalogSaleStatus(ctx context.Context, code string) (string, error) {
	row, err := s.q.GetLotteryCatalogByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrCatalogNotFound
		}
		return "", err
	}
	return sqlcdb.SaleStatusString(row.SaleStatus), nil
}
