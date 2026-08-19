package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
)

type operation struct {
	name  string
	batch int
}

func parseOperation(args []string) (operation, error) {
	if len(args) == 0 {
		return operation{}, errors.New("operation is required")
	}
	op := operation{name: args[0], batch: 5000}
	switch op.name {
	case "status", "validate", "ack-restart":
		if len(args) != 1 {
			return operation{}, fmt.Errorf("%s does not accept arguments", op.name)
		}
	case "backfill":
		fs := flag.NewFlagSet("backfill", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		fs.IntVar(&op.batch, "batch", 5000, "rows per table")
		if err := fs.Parse(args[1:]); err != nil {
			return operation{}, err
		}
		if fs.NArg() != 0 {
			return operation{}, errors.New("unexpected backfill argument")
		}
		if op.batch < 1 || op.batch > 100000 {
			return operation{}, errors.New("batch must be between 1 and 100000")
		}
	case "cutover":
		if len(args) != 2 || args[1] != "--confirm-cutover" {
			return operation{}, errors.New("cutover requires --confirm-cutover")
		}
	case "rollback":
		if len(args) != 2 || args[1] != "--confirm-rollback" {
			return operation{}, errors.New("rollback requires --confirm-rollback")
		}
	default:
		return operation{}, fmt.Errorf("unknown operation %q", op.name)
	}
	return op, nil
}

func main() {
	op, err := parseOperation(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/core-partition status|backfill [--batch N]|validate|cutover --confirm-cutover|rollback --confirm-rollback|ack-restart")
		log.Fatal(err)
	}
	_ = godotenv.Load()
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL or DB_HOST is not configured")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 1)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	ctx := context.Background()
	var payload []byte
	switch op.name {
	case "status":
		err = pool.QueryRow(ctx, `
SELECT jsonb_build_object(
    'phase', s.phase,
    'forwardSync', s.forward_sync,
    'reverseSync', s.reverse_sync,
    'restartRequired', s.restart_required,
    'lastValidation', s.last_validation,
    'lastValidatedAt', s.last_validated_at,
    'cutoverAt', s.cutover_at,
    'rollbackAt', s.rollback_at,
    'activeTablesPartitioned', jsonb_build_object(
        'betOrders', (SELECT relkind = 'p' FROM pg_class WHERE oid = 'bet_orders'::regclass),
        'cloudBetRecords', (SELECT relkind = 'p' FROM pg_class WHERE oid = 'cloud_bet_records'::regclass),
        'walletLedger', (SELECT relkind = 'p' FROM pg_class WHERE oid = 'wallet_ledger'::regclass)
    )
)
FROM core_partition_migration_state s
WHERE s.id = 1`).Scan(&payload)
	case "backfill":
		err = pool.QueryRow(ctx,
			"SELECT backfill_core_online_partitions($1)::text::bytea",
			op.batch,
		).Scan(&payload)
	case "validate":
		var phase string
		if scanErr := pool.QueryRow(ctx,
			"SELECT phase FROM core_partition_migration_state WHERE id = 1",
		).Scan(&phase); scanErr != nil {
			log.Fatal(scanErr)
		}
		fn := "validate_core_online_partitions"
		if phase == "cutover" || phase == "rollback_ready" {
			fn = "validate_core_online_partition_cutover"
		}
		err = pool.QueryRow(ctx, "SELECT "+fn+"()::text::bytea").Scan(&payload)
	case "cutover":
		err = pool.QueryRow(ctx, "SELECT cutover_core_online_partitions()::text::bytea").Scan(&payload)
	case "rollback":
		err = pool.QueryRow(ctx, "SELECT rollback_core_online_partitions()::text::bytea").Scan(&payload)
	case "ack-restart":
		_, err = pool.Exec(ctx, "SELECT acknowledge_core_partition_restart()")
		payload = []byte(`{"restartRequired":false}`)
	}
	if err != nil {
		log.Fatal(err)
	}
	if len(payload) == 0 {
		payload = []byte(`{"ok":true}`)
	}
	fmt.Println(string(payload))
}
