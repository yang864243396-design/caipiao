# Core Online Table Partition Design

## Goal

Convert `bet_orders`, `cloud_bet_records`, and `wallet_ledger` into
monthly range-partitioned online tables without changing application SQL,
losing global business-key uniqueness, or removing the ability to roll back.

## Constraints

- Existing SQL continues to use the three current table names.
- Global uniqueness remains authoritative for ids and business numbers.
- `cloud_bet_records(id)` references are redirected to a stable identity table.
- Wallet rows remain append-only and financial validation is mandatory.
- Backfill runs in bounded batches while forward sync triggers cover concurrent writes.
- Cutover requires a short application stop because existing prepared statements retain
  relation OIDs across a table-name swap.
- A reverse-sync rollback window is kept after cutover.

## Architecture

Each business table receives a normal identity registry and a partitioned mirror.
The registry owns global uniqueness that PostgreSQL cannot enforce across date
partitions. The mirror owns composite primary keys that contain the partition key.

During migration, triggers copy inserts and updates from the active unpartitioned
table into the mirror. Batch functions backfill older rows idempotently. A validation
function compares row counts, missing ids, business-key mismatches, and financial
totals.

Cutover locks all three active tables, performs a final backfill and validation,
rewires the two cloud-record foreign keys to the registry, swaps table names, and
installs reverse-sync triggers. Rollback performs the inverse swap after validating
that reverse synchronization is complete.

## Partition Layout

- Range key: `placed_at` for bet and cloud records, `created_at` for wallet ledger.
- Monthly partitions are created by an idempotent helper.
- A default partition prevents a write outage when a future partition was not
  pre-created.
- Startup maintenance creates the current month plus twelve future months.

## Operational States

`core_partition_migration_state.phase` is one of:

- `mirroring`: old tables active, forward triggers enabled.
- `validated`: a full validation succeeded, ready for maintenance cutover.
- `cutover`: partitioned tables active, reverse triggers enabled.
- `rollback_ready`: post-cutover validation succeeded.

Cutover and rollback functions reject invalid states or failed validation.

## Verification

- Static migration contract tests prove the required registries, partition parents,
  sync functions, validation, cutover, and rollback functions exist.
- Database integration tests exercise forward sync, idempotent backfill, uniqueness,
  partition placement, append-only wallet behavior, cutover, reverse sync, and rollback
  inside a transaction.
- Production execution records before/after counts and amounts and restarts the backend
  after the relation swap.

