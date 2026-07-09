-- +goose Up
-- +goose StatementBegin
-- 删除第三方 game_id 69/70/71/81 四个不再对接的彩种及关联数据。
DO $$
DECLARE
    codes text[] := ARRAY['taiwan_ssc_5m', 'taiwan_pk10', 'taiwan_pc28', 'tron_lhc'];
BEGIN
    DELETE FROM wallet_ledger wl
    USING bet_orders b
    WHERE wl.order_ref = b.order_no AND b.lottery_code = ANY(codes);

    DELETE FROM wallet_ledger wl
    USING chase_orders c
    WHERE wl.order_ref = c.chase_no AND c.lottery_code = ANY(codes);

    DELETE FROM bet_orders WHERE lottery_code = ANY(codes);
    DELETE FROM chase_orders WHERE lottery_code = ANY(codes);

    DELETE FROM cloud_bet_records cbr
    USING scheme_instances si
    WHERE cbr.scheme_id = si.id AND si.lottery_code = ANY(codes);

    DELETE FROM member_scheme_favorites msf
    USING scheme_share_snapshots ss
    WHERE msf.snapshot_id = ss.id AND ss.lottery_code = ANY(codes);

    DELETE FROM scheme_definitions WHERE lottery_code = ANY(codes);
    DELETE FROM copy_hall_rank_slots WHERE lottery_code = ANY(codes);
    DELETE FROM scheme_share_snapshots WHERE lottery_code = ANY(codes);
    DELETE FROM scheme_templates WHERE lottery_code = ANY(codes);
    DELETE FROM lottery_draws WHERE lottery_code = ANY(codes);
    DELETE FROM lottery_scheme_option_sets WHERE lottery_code = ANY(codes);

    DELETE FROM admin_audit_logs a
    WHERE EXISTS (
        SELECT 1 FROM unnest(codes) AS c(code)
        WHERE a.action ILIKE '%' || c.code || '%'
    );

    DELETE FROM lottery_catalog WHERE code = ANY(codes);
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 已物理删除的彩种不做回滚恢复。
-- +goose StatementEnd
