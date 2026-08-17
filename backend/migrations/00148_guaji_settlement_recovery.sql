-- +goose Up
CREATE TABLE IF NOT EXISTS guaji_settlement_recovery (
    guaji_account_id BIGINT NOT NULL REFERENCES member_guaji_accounts(id) ON DELETE CASCADE,
    third_party_bet_id TEXT NOT NULL,
    next_page INTEGER NOT NULL DEFAULT 4 CHECK (next_page >= 4),
    last_error TEXT NOT NULL DEFAULT '',
    last_attempt_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guaji_account_id, third_party_bet_id)
);

-- +goose Down
DROP TABLE IF EXISTS guaji_settlement_recovery;
