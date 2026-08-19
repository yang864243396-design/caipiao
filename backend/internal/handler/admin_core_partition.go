package handler

import (
	"encoding/json"
	"net/http"

	"caipiao/backend/internal/apix"
)

func (h *Handler) AdminCorePartitionStatus(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u6570\u636e\u5e93\u672a\u5c31\u7eea")
		return
	}
	var payload []byte
	err := h.db.QueryRow(r.Context(), `
WITH active AS (
    SELECT
        (SELECT relkind = 'p' FROM pg_class WHERE oid = 'bet_orders'::regclass) AS bet_partitioned,
        (SELECT relkind = 'p' FROM pg_class WHERE oid = 'cloud_bet_records'::regclass) AS cloud_partitioned,
        (SELECT relkind = 'p' FROM pg_class WHERE oid = 'wallet_ledger'::regclass) AS ledger_partitioned
),
parents AS (
    SELECT
        CASE WHEN active.bet_partitioned THEN to_regclass('bet_orders')
             ELSE to_regclass('bet_orders_partitioned') END AS bet_parent,
        CASE WHEN active.cloud_partitioned THEN to_regclass('cloud_bet_records')
             ELSE to_regclass('cloud_bet_records_partitioned') END AS cloud_parent,
        CASE WHEN active.ledger_partitioned THEN to_regclass('wallet_ledger')
             ELSE to_regclass('wallet_ledger_partitioned') END AS ledger_parent
    FROM active
)
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
        'betOrders', active.bet_partitioned,
        'cloudBetRecords', active.cloud_partitioned,
        'walletLedger', active.ledger_partitioned
    ),
    'partitionCounts', jsonb_build_object(
        'betOrders', (SELECT count(*) FROM pg_inherits WHERE inhparent = parents.bet_parent),
        'cloudBetRecords', (SELECT count(*) FROM pg_inherits WHERE inhparent = parents.cloud_parent),
        'walletLedger', (SELECT count(*) FROM pg_inherits WHERE inhparent = parents.ledger_parent)
    )
)
FROM core_partition_migration_state s
CROSS JOIN active
CROSS JOIN parents
WHERE s.id = 1`).Scan(&payload)
	if err != nil {
		apix.Internal(w)
		return
	}
	apix.OK(w, json.RawMessage(payload))
}
