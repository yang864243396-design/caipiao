-- name: DeleteWalletLedgerForBetOrders :exec
DELETE FROM wallet_ledger wl
USING bet_orders b
WHERE wl.order_ref = b.order_no
  AND b.lottery_code = ANY($1::text[]);

-- name: DeleteWalletLedgerForChaseOrders :exec
DELETE FROM wallet_ledger wl
USING chase_orders c
WHERE wl.order_ref = c.chase_no
  AND c.lottery_code = ANY($1::text[]);

-- name: DeleteBetOrdersByLotteryCodes :exec
DELETE FROM bet_orders
WHERE lottery_code = ANY($1::text[]);

-- name: DeleteChaseOrdersByLotteryCodes :exec
DELETE FROM chase_orders
WHERE lottery_code = ANY($1::text[]);

-- name: DeleteCloudBetRecordsForLotteryCodes :exec
DELETE FROM cloud_bet_records cbr
USING scheme_instances si
WHERE cbr.scheme_id = si.id
  AND si.lottery_code = ANY($1::text[]);

-- name: DeleteSchemeDefinitionsByLotteryCodes :exec
DELETE FROM scheme_definitions
WHERE lottery_code = ANY($1::text[]);

-- name: DeleteCopyHallRankSlotsByLotteryCodes :exec
DELETE FROM copy_hall_rank_slots
WHERE lottery_code = ANY($1::text[]);

-- name: DeleteSchemeShareSnapshotsByLotteryCodes :exec
DELETE FROM scheme_share_snapshots
WHERE lottery_code = ANY($1::text[]);

-- name: DeleteSchemeTemplatesByLotteryCodes :exec
DELETE FROM scheme_templates
WHERE lottery_code = ANY($1::text[]);

-- name: DeleteLotteryDrawsByLotteryCodes :exec
DELETE FROM lottery_draws
WHERE lottery_code = ANY($1::text[]);

-- name: DeleteLotterySchemeOptionSetsByLotteryCodes :exec
DELETE FROM lottery_scheme_option_sets
WHERE lottery_code = ANY($1::text[]);

-- name: DeleteAdminAuditLogsForLegacyLotteries :exec
DELETE FROM admin_audit_logs a
WHERE EXISTS (
    SELECT 1 FROM unnest($1::text[]) AS c(code)
    WHERE a.action ILIKE '%' || c.code || '%'
);

-- name: DeleteLotteryCatalogByCodes :exec
DELETE FROM lottery_catalog
WHERE code = ANY($1::text[]);
