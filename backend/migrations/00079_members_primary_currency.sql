-- +goose Up
-- +goose StatementBegin
-- T1b：会员主币种（USDT / TRX / CNY，默认 CNY）；切换主币种停全部挂机方案

ALTER TABLE members
    ADD COLUMN IF NOT EXISTS primary_currency VARCHAR(8) NOT NULL DEFAULT 'CNY';

ALTER TABLE members
    ADD CONSTRAINT chk_members_primary_currency
    CHECK (primary_currency IN ('USDT', 'TRX', 'CNY'));

COMMENT ON COLUMN members.primary_currency IS '主币种：USDT/TRX/CNY；方案运行与顶栏余额按此币种（默认 CNY）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_primary_currency;
ALTER TABLE members DROP COLUMN IF EXISTS primary_currency;
-- +goose StatementEnd
