package sqlcdb

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

// LotteryCatalogRow 彩种目录行（games 模块展示用，统一多 sqlc 查询返回类型）。
type LotteryCatalogRow struct {
	Code                string
	DisplayName         string
	CategoryCode        pgtype.Text
	PlayTemplate        pgtype.Text
	BallCount           pgtype.Int2
	DrawInterval        pgtype.Text
	SortOrder           int32
	OnSale              bool
	SaleStatus          string
	OutboundLotteryCode pgtype.Text
}

// SaleStatusString 将 sqlc sale_status（interface{}）转为 string。
func SaleStatusString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprint(v)
}

func lotteryCatalogFromSQL(
	code, displayName string,
	categoryCode, playTemplate pgtype.Text,
	ballCount pgtype.Int2,
	drawInterval pgtype.Text,
	sortOrder int32,
	onSale bool,
	saleStatus interface{},
	outboundLotteryCode pgtype.Text,
) LotteryCatalogRow {
	return LotteryCatalogRow{
		Code:                code,
		DisplayName:         displayName,
		CategoryCode:        categoryCode,
		PlayTemplate:        playTemplate,
		BallCount:           ballCount,
		DrawInterval:        drawInterval,
		SortOrder:           sortOrder,
		OnSale:              onSale,
		SaleStatus:          SaleStatusString(saleStatus),
		OutboundLotteryCode: outboundLotteryCode,
	}
}

func LotteryCatalogRowFromByCode(r GetLotteryCatalogByCodeRow) LotteryCatalogRow {
	return lotteryCatalogFromSQL(
		r.Code, r.DisplayName, r.CategoryCode, r.PlayTemplate, r.BallCount, r.DrawInterval,
		r.SortOrder, r.OnSale, r.SaleStatus, r.OutboundLotteryCode,
	)
}

func LotteryCatalogRowFromList(r ListLotteryCatalogRow) LotteryCatalogRow {
	return lotteryCatalogFromSQL(
		r.Code, r.DisplayName, r.CategoryCode, r.PlayTemplate, r.BallCount, r.DrawInterval,
		r.SortOrder, r.OnSale, r.SaleStatus, r.OutboundLotteryCode,
	)
}

func LotteryCatalogRowFromPatch(r PatchLotteryCatalogMaintenanceRow) LotteryCatalogRow {
	return lotteryCatalogFromSQL(
		r.Code, r.DisplayName, r.CategoryCode, r.PlayTemplate, r.BallCount, r.DrawInterval,
		r.SortOrder, r.OnSale, r.SaleStatus, r.OutboundLotteryCode,
	)
}

func LotteryCatalogRowFromMaintenance(r SetLotteryCatalogMaintenanceRow) LotteryCatalogRow {
	return lotteryCatalogFromSQL(
		r.Code, r.DisplayName, r.CategoryCode, r.PlayTemplate, r.BallCount, r.DrawInterval,
		r.SortOrder, r.OnSale, r.SaleStatus, r.OutboundLotteryCode,
	)
}

func LotteryCatalogRowFromOnSale(r ListLotteryCatalogOnSaleRow) LotteryCatalogRow {
	return lotteryCatalogFromSQL(
		r.Code, r.DisplayName, r.CategoryCode, r.PlayTemplate, r.BallCount, r.DrawInterval,
		r.SortOrder, r.OnSale, r.SaleStatus, r.OutboundLotteryCode,
	)
}
