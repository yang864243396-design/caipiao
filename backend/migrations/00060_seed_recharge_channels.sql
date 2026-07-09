-- +goose Up
-- +goose StatementBegin
INSERT INTO recharge_channels (
    id, channel_group, label, icon, recommended, fee_rate, min_amount, max_amount,
    show_activities, bind_reminder, chain_hint, sort_order, enabled
) VALUES
    ('usdt-trc20', 'crypto', 'USDT（TRC20）', 'currency_bitcoin', true, 0.02, 100, 50000, false, NULL, '请确认网络为 TRC20，错链转账无法找回。', 10, true),
    ('usdt-bsc', 'crypto', 'USDT（BSC）', 'currency_bitcoin', false, 0.02, 100, 50000, false, NULL, '请确认网络为 BSC（BEP20），错链转账无法找回。', 20, true),
    ('usdt-ton', 'crypto', 'USDT（TON）', 'currency_bitcoin', false, 0.02, 100, 50000, false, NULL, '请确认网络为 TON，错链转账无法找回。', 30, true),
    ('usdt-sol', 'crypto', 'USDT（SOL）', 'currency_bitcoin', false, 0.02, 100, 50000, false, NULL, '请确认网络为 Solana，错链转账无法找回。', 40, true),
    ('douyin-1', 'social', '抖音支付 1', 'music_video', true, 0.02, 100, 5000, true, NULL, NULL, 50, true),
    ('douyin-2', 'social', '抖音支付 2', 'music_video', true, 0.02, 100, 5000, true, NULL, NULL, 60, true),
    ('goubao-1', 'wallet', '购宝钱包 1', 'account_balance_wallet', false, 0.02, 100, 499, true, '充值前需先至银行卡设置绑定购宝钱包。', NULL, 70, true),
    ('mpay-1', 'wallet', 'MPay1', 'smartphone', false, 0.02, 100, 5000, true, NULL, NULL, 80, true),
    ('vip-1', 'wallet', 'VIPPAY1', 'payments', false, 0.02, 100, 5000, true, NULL, NULL, 90, true),
    ('eb-1', 'wallet', 'EBpay1', 'credit_card', false, 0.02, 100, 5000, true, NULL, NULL, 100, true),
    ('alipay-1', 'alipay', '支付宝 1', 'account_balance', true, 0.02, 500, 50000, true, NULL, NULL, 110, true),
    ('alipay-2', 'alipay', '支付宝 2', 'account_balance', true, 0.02, 100, 499, true, NULL, NULL, 120, true),
    ('alipay-3', 'alipay', '支付宝 3', 'account_balance', false, 0.02, 500, 50000, true, NULL, NULL, 130, true),
    ('union-1', 'bank', '银联扫码 1', 'qr_code_scanner', false, 0.02, 500, 50000, true, NULL, NULL, 140, true)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM recharge_channels WHERE id IN (
    'usdt-trc20', 'usdt-bsc', 'usdt-ton', 'usdt-sol',
    'douyin-1', 'douyin-2', 'goubao-1', 'mpay-1', 'vip-1', 'eb-1',
    'alipay-1', 'alipay-2', 'alipay-3', 'union-1'
);
-- +goose StatementEnd
