-- +goose Up
-- Conservative initial limits for the explicitly enabled tron_ffc_6s gray
-- rollout. Existing operator-managed values, including a disabled row, win.
INSERT INTO scheme_betting_capacity_limits (
    lottery_code,
    max_due_outbox,
    max_active_schemes,
    max_dispatch_per_second,
    enabled,
    max_account_dispatch_per_second,
    max_global_dispatch_per_second,
    updated_at
)
VALUES ('tron_ffc_6s', 1024, 256, 64, true, 16, 128, now())
ON CONFLICT (lottery_code) DO NOTHING;

-- +goose Down
DELETE FROM scheme_betting_capacity_limits
WHERE lottery_code = 'tron_ffc_6s'
  AND max_due_outbox = 1024
  AND max_active_schemes = 256
  AND max_dispatch_per_second = 64
  AND max_account_dispatch_per_second = 16
  AND max_global_dispatch_per_second = 128;
