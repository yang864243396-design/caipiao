package schemes

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/timeutil"
)

func mapDefinitionFields(
	id, kind, schemeName, lotteryCode, lotteryLabel, shareStatus string,
	config []byte,
	createdAt, updatedAt pgtype.Timestamptz,
	hasInstance bool,
) Definition {
	cfg := map[string]interface{}{}
	if len(config) > 0 {
		_ = json.Unmarshal(config, &cfg)
	}
	return Definition{
		ID:                id,
		Kind:              kind,
		SchemeName:        schemeName,
		LotteryCode:       lotteryCode,
		LotteryLabel:      lotteryLabel,
		ShareStatusLocked: shareStatus,
		Config:            cfg,
		HasInstance:       hasInstance,
		CreatedAt:         timeutil.FormatISO(createdAt.Time),
		UpdatedAt:         timeutil.FormatISO(updatedAt.Time),
	}
}

func mapDefinitionRow(row sqlcdb.InsertSchemeDefinitionRow, hasInstance bool) Definition {
	return mapDefinitionFields(
		row.ID, row.Kind, row.SchemeName, row.LotteryCode, row.LotteryLabel,
		row.ShareStatus, row.Config, row.CreatedAt, row.UpdatedAt, hasInstance,
	)
}
