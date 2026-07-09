-- +goose Up
-- +goose StatementBegin
-- T1：第三方挂机授权账号（member_guaji_accounts）

CREATE TABLE IF NOT EXISTS member_guaji_accounts (
    id                  BIGSERIAL PRIMARY KEY,
    member_id           BIGINT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    guaji_username      VARCHAR(64) NOT NULL,
    password_enc        TEXT NOT NULL,
    mfa_material_enc    TEXT,
    access_token_enc    TEXT,
    refresh_token_enc   TEXT,
    token_expires_at    TIMESTAMPTZ,
    is_active           BOOLEAN NOT NULL DEFAULT false,
    bound_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_sync_at        TIMESTAMPTZ,
    last_token_error    TEXT,
    last_bet_at         TIMESTAMPTZ,
    reauth_fail_count   INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT member_guaji_accounts_username_unique UNIQUE (guaji_username)
);

CREATE UNIQUE INDEX IF NOT EXISTS member_guaji_accounts_one_active_per_member
    ON member_guaji_accounts (member_id)
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS member_guaji_accounts_member_id_idx
    ON member_guaji_accounts (member_id);

COMMENT ON TABLE member_guaji_accounts IS '第三方 Hash 挂机授权账号；guaji_username 全局唯一';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS member_guaji_accounts;
-- +goose StatementEnd
