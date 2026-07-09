-- +goose Up
-- +goose StatementBegin
CREATE TABLE lottery_draws (
    id            BIGSERIAL PRIMARY KEY,
    lottery_code  VARCHAR(32)  NOT NULL,
    issue_no      VARCHAR(32)  NOT NULL,
    period_short  VARCHAR(16)  NOT NULL,
    balls         JSONB        NOT NULL,
    sum_value     INT          NOT NULL,
    drawn_at      TIMESTAMPTZ  NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_lottery_draws_issue UNIQUE (lottery_code, issue_no)
);

COMMENT ON TABLE lottery_draws IS '彩种历史开奖（玩法详情 · 历史开奖 Tab）';
COMMENT ON COLUMN lottery_draws.lottery_code IS '彩种 code';
COMMENT ON COLUMN lottery_draws.issue_no IS '官方期号';
COMMENT ON COLUMN lottery_draws.period_short IS '短期号展示，如 031';
COMMENT ON COLUMN lottery_draws.balls IS '开奖号码 JSON 数组';
COMMENT ON COLUMN lottery_draws.sum_value IS '号码总和';

CREATE INDEX idx_lottery_draws_lookup
    ON lottery_draws (lottery_code, drawn_at DESC, id DESC);

INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at) VALUES
('tencent_ffc', '20231103031', '031', '["3","9","2","7","5"]'::jsonb, 26, '2023-10-27 12:40:00+00'),
('tencent_ffc', '20231103030', '030', '["8","1","0","6","4"]'::jsonb, 19, '2023-10-27 12:35:00+00'),
('tencent_ffc', '20231103029', '029', '["4","5","5","1","8"]'::jsonb, 23, '2023-10-27 12:30:00+00'),
('tencent_ffc', '20231103028', '028', '["2","2","9","0","3"]'::jsonb, 16, '2023-10-27 12:25:00+00'),
('tencent_ffc', '20231103027', '027', '["1","6","3","3","7"]'::jsonb, 20, '2023-10-27 12:20:00+00');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lottery_draws;
-- +goose StatementEnd
