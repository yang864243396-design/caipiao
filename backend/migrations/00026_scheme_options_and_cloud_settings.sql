-- +goose Up
-- +goose StatementBegin
CREATE TABLE lottery_scheme_option_sets (
    lottery_code VARCHAR(32)  PRIMARY KEY,
    run_types    JSONB        NOT NULL DEFAULT '[]'::jsonb,
    play_types   JSONB        NOT NULL DEFAULT '[]'::jsonb,
    sub_plays    JSONB        NOT NULL DEFAULT '[]'::jsonb,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE lottery_scheme_option_sets IS '彩种方案选项种子（运行/玩法/子玩法，只读）';
COMMENT ON COLUMN lottery_scheme_option_sets.lottery_code IS '彩种 code；_default 为兜底';
COMMENT ON COLUMN lottery_scheme_option_sets.run_types IS '运行类型选项 [{value,label}]';
COMMENT ON COLUMN lottery_scheme_option_sets.play_types IS '玩法类型选项 [{value,label}]';
COMMENT ON COLUMN lottery_scheme_option_sets.sub_plays IS '子玩法选项 [{value,label}]';

CREATE TABLE member_cloud_settings (
    member_id          BIGINT       PRIMARY KEY REFERENCES members(id) ON DELETE CASCADE,
    total_stop_loss    NUMERIC(18,2) NOT NULL DEFAULT 0,
    total_take_profit  NUMERIC(18,2) NOT NULL DEFAULT 0,
    plan_multiplier    NUMERIC(10,4) NOT NULL DEFAULT 1,
    break_period_stop  BOOLEAN      NOT NULL DEFAULT false,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE member_cloud_settings IS '会员云端全局规则（总止损止盈、断期停投等）';
COMMENT ON COLUMN member_cloud_settings.total_stop_loss IS '总止损（元）';
COMMENT ON COLUMN member_cloud_settings.total_take_profit IS '总止盈（元）';
COMMENT ON COLUMN member_cloud_settings.plan_multiplier IS '方案倍数系数';
COMMENT ON COLUMN member_cloud_settings.break_period_stop IS '断期停投开关';

INSERT INTO lottery_scheme_option_sets (lottery_code, run_types, play_types, sub_plays) VALUES
(
    '_default',
    '[
        {"value":"fixed_rotate","label":"定码轮换"},
        {"value":"adv_fixed_rotate","label":"高级定码轮换"},
        {"value":"random_draw","label":"随机出号"},
        {"value":"batch_fixed","label":"批量定码"},
        {"value":"dynamic_chase","label":"动态追号"},
        {"value":"plan_follow","label":"计划跟投"}
    ]'::jsonb,
    '[
        {"value":"hou4","label":"后四"},
        {"value":"qian3","label":"前三"},
        {"value":"zhong3","label":"中三"},
        {"value":"dingwei","label":"定位胆"}
    ]'::jsonb,
    '[
        {"value":"zhixuan_fs","label":"直选复式"},
        {"value":"zhixuan_ds","label":"直选单式"},
        {"value":"zuxuan_fs","label":"组选复式"}
    ]'::jsonb
),
(
    'tencent_ffc',
    '[
        {"value":"fixed_rotate","label":"定码轮换"},
        {"value":"adv_fixed_rotate","label":"高级定码轮换"},
        {"value":"random_draw","label":"随机出号"},
        {"value":"batch_fixed","label":"批量定码"},
        {"value":"dynamic_chase","label":"动态追号"},
        {"value":"plan_follow","label":"计划跟投"}
    ]'::jsonb,
    '[
        {"value":"hou4","label":"后四"},
        {"value":"qian3","label":"前三"},
        {"value":"zhong3","label":"中三"},
        {"value":"dingwei","label":"定位胆"}
    ]'::jsonb,
    '[
        {"value":"zhixuan_fs","label":"直选复式"},
        {"value":"zhixuan_ds","label":"直选单式"},
        {"value":"zuxuan_fs","label":"组选复式"}
    ]'::jsonb
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS member_cloud_settings;
DROP TABLE IF EXISTS lottery_scheme_option_sets;
-- +goose StatementEnd
