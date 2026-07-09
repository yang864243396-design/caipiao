-- +goose Up
-- +goose StatementBegin
ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS third_party_period VARCHAR(32);

COMMENT ON COLUMN cloud_bet_records.third_party_period IS '第三方 web_bets/lott 接单返回的 periods（real）';
-- +goose StatementEnd

-- +goose Down
ALTER TABLE cloud_bet_records DROP COLUMN IF EXISTS third_party_period;
