-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ensure_core_online_partitions(
    start_month DATE DEFAULT date_trunc('month', now())::date,
    months_ahead INTEGER DEFAULT 12
)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    month_start DATE;
    month_end DATE;
    suffix TEXT;
    created_count INTEGER := 0;
    month_offset INTEGER;
    table_index INTEGER;
    parent_names TEXT[];
    partition_prefixes TEXT[] := ARRAY[
        'bet_orders_partitioned',
        'cloud_bet_records_partitioned',
        'wallet_ledger_partitioned'
    ];
    partition_name TEXT;
BEGIN
    IF start_month <> date_trunc('month', start_month)::date THEN
        RAISE EXCEPTION 'start_month must be the first day of a month';
    END IF;
    IF months_ahead < 0 OR months_ahead > 240 THEN
        RAISE EXCEPTION 'months_ahead must be between 0 and 240';
    END IF;

    parent_names := ARRAY[
        CASE
            WHEN EXISTS (
                SELECT 1 FROM pg_class
                WHERE oid = to_regclass('bet_orders') AND relkind = 'p'
            ) THEN 'bet_orders'
            ELSE 'bet_orders_partitioned'
        END,
        CASE
            WHEN EXISTS (
                SELECT 1 FROM pg_class
                WHERE oid = to_regclass('cloud_bet_records') AND relkind = 'p'
            ) THEN 'cloud_bet_records'
            ELSE 'cloud_bet_records_partitioned'
        END,
        CASE
            WHEN EXISTS (
                SELECT 1 FROM pg_class
                WHERE oid = to_regclass('wallet_ledger') AND relkind = 'p'
            ) THEN 'wallet_ledger'
            ELSE 'wallet_ledger_partitioned'
        END
    ];

    FOR table_index IN 1..array_length(parent_names, 1) LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_class
            WHERE oid = to_regclass(parent_names[table_index]) AND relkind = 'p'
        ) THEN
            RAISE EXCEPTION 'partition parent % is missing', parent_names[table_index];
        END IF;
    END LOOP;

    FOR month_offset IN 0..months_ahead LOOP
        month_start := (start_month + make_interval(months => month_offset))::date;
        month_end := (month_start + interval '1 month')::date;
        suffix := to_char(month_start, 'YYYYMM');

        FOR table_index IN 1..array_length(parent_names, 1) LOOP
            partition_name := partition_prefixes[table_index] || '_' || suffix;
            IF to_regclass(partition_name) IS NULL THEN
                EXECUTE format(
                    'CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                    partition_name,
                    parent_names[table_index],
                    month_start,
                    month_end
                );
                created_count := created_count + 1;
            END IF;
        END LOOP;
    END LOOP;
    RETURN created_count;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ensure_core_online_partitions(
    start_month DATE DEFAULT date_trunc('month', now())::date,
    months_ahead INTEGER DEFAULT 12
)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    month_start DATE;
    month_end DATE;
    suffix TEXT;
    created_count INTEGER := 0;
    i INTEGER;
    table_name TEXT;
    partition_name TEXT;
BEGIN
    IF start_month <> date_trunc('month', start_month)::date THEN
        RAISE EXCEPTION 'start_month must be the first day of a month';
    END IF;
    IF months_ahead < 0 OR months_ahead > 240 THEN
        RAISE EXCEPTION 'months_ahead must be between 0 and 240';
    END IF;

    FOR i IN 0..months_ahead LOOP
        month_start := (start_month + make_interval(months => i))::date;
        month_end := (month_start + interval '1 month')::date;
        suffix := to_char(month_start, 'YYYYMM');
        FOREACH table_name IN ARRAY ARRAY[
            'bet_orders_partitioned',
            'cloud_bet_records_partitioned',
            'wallet_ledger_partitioned'
        ] LOOP
            partition_name := table_name || '_' || suffix;
            IF to_regclass(partition_name) IS NULL THEN
                EXECUTE format(
                    'CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                    partition_name, table_name, month_start, month_end
                );
                created_count := created_count + 1;
            END IF;
        END LOOP;
    END LOOP;
    RETURN created_count;
END;
$$;
-- +goose StatementEnd
