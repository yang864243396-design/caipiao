package games



import (

	"context"

	"errors"

	"fmt"



	"github.com/jackc/pgx/v5"



	"caipiao/backend/internal/db"

	"caipiao/backend/internal/db/sqlcdb"

)



var ErrCatalogPurgeBlocked = errors.New("lottery catalog purge precondition failed")



// RunLegacyCatalogPurge startup 一次幂等 purge（C19、C31、C45）。

func RunLegacyCatalogPurge(ctx context.Context, pool *db.Pool) error {

	if pool == nil {

		return nil

	}

	q := sqlcdb.New(pool)

	if _, err := q.GetLotteryCatalogPurgeState(ctx); err == nil {

		return nil

	} else if !errors.Is(err, pgx.ErrNoRows) {

		return fmt.Errorf("purge state: %w", err)

	}



	catalogCount, err := q.CountLotteryCatalogWithTemplate(ctx)

	if err != nil {

		return fmt.Errorf("purge precondition catalog count: %w", err)

	}

	subCount, err := q.CountSubPlays(ctx)

	if err != nil {

		return fmt.Errorf("purge precondition sub_plays count: %w", err)

	}

	if catalogCount != expectedCatalogSeedCount || subCount != expectedSubPlayCount {

		return fmt.Errorf(

			"%w: need %d catalog with template and %d sub_plays, got catalog=%d sub_plays=%d",

			ErrCatalogPurgeBlocked,

			expectedCatalogSeedCount,

			expectedSubPlayCount,

			catalogCount,

			subCount,

		)

	}



	tx, err := pool.Begin(ctx)

	if err != nil {

		return fmt.Errorf("purge begin tx: %w", err)

	}

	defer tx.Rollback(ctx)



	qtx := q.WithTx(tx)

	codes := LegacyLotteryCodes

	steps := []struct {

		name string

		fn   func(context.Context, []string) error

	}{

		{"wallet_ledger(bet)", qtx.DeleteWalletLedgerForBetOrders},

		{"wallet_ledger(chase)", qtx.DeleteWalletLedgerForChaseOrders},

		{"bet_orders", qtx.DeleteBetOrdersByLotteryCodes},

		{"chase_orders", qtx.DeleteChaseOrdersByLotteryCodes},

		{"cloud_bet_records", qtx.DeleteCloudBetRecordsForLotteryCodes},

		{"scheme_definitions", qtx.DeleteSchemeDefinitionsByLotteryCodes},

		{"copy_hall_rank_slots", qtx.DeleteCopyHallRankSlotsByLotteryCodes},

		{"scheme_share_snapshots", qtx.DeleteSchemeShareSnapshotsByLotteryCodes},

		{"scheme_templates", qtx.DeleteSchemeTemplatesByLotteryCodes},

		{"lottery_draws", qtx.DeleteLotteryDrawsByLotteryCodes},

		{"lottery_scheme_option_sets", qtx.DeleteLotterySchemeOptionSetsByLotteryCodes},

		{"admin_audit_logs", qtx.DeleteAdminAuditLogsForLegacyLotteries},

		{"lottery_catalog", qtx.DeleteLotteryCatalogByCodes},

	}

	for _, step := range steps {

		if err := step.fn(ctx, codes); err != nil {

			return fmt.Errorf("purge %s: %w", step.name, err)

		}

	}

	if err := qtx.InsertLotteryCatalogPurgeState(ctx, "legacy 9 lottery purge completed"); err != nil {

		return fmt.Errorf("purge marker: %w", err)

	}

	if err := tx.Commit(ctx); err != nil {

		return fmt.Errorf("purge commit: %w", err)

	}

	return nil

}

