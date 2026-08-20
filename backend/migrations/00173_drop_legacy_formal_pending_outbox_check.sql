-- +goose Up
-- Earlier deployments can retain an auto-renamed copy of the Phase 1 guard:
-- mode = 'shadow' OR state <> 'pending'.  Formal gray/production commands
-- are intentionally created pending, so remove every legacy copy by meaning
-- rather than by its unstable generated constraint name (for example check1).
DO $migration$
DECLARE
    constraint_name TEXT;
BEGIN
    FOR constraint_name IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid = 'scheme_bet_outbox'::regclass
          AND contype = 'c'
          AND pg_get_constraintdef(oid) ~ 'state.*<>.*pending'
    LOOP
        EXECUTE format('ALTER TABLE scheme_bet_outbox DROP CONSTRAINT %I', constraint_name);
    END LOOP;
END
$migration$;

-- +goose Down
-- Restoring the legacy restriction would invalidate active formal pending rows.
