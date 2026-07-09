-- +goose Up
-- +goose StatementBegin
-- 运行类型差异化引擎（docs/run-types-implementation-plan.md v8）

-- 1) 实例出号运行时状态（与倍率轮次游标 round_index 解耦）
ALTER TABLE scheme_instances
    ADD COLUMN pick_index     INT          NOT NULL DEFAULT 0,
    ADD COLUMN current_pick   TEXT         NOT NULL DEFAULT '',
    ADD COLUMN last_direction VARCHAR(8)   NOT NULL DEFAULT '';

COMMENT ON COLUMN scheme_instances.pick_index IS '出号游标：定码轮换组游标 / 高级定码轮换当前局数（0=未初始化）';
COMMENT ON COLUMN scheme_instances.current_pick IS '当前号码池（冷热温/随机跨期保号）';
COMMENT ON COLUMN scheme_instances.last_direction IS '开某投某上一局投向：pos 正投 / neg 反投';

UPDATE scheme_instances SET pick_index = round_index;

-- 2) 投注明细记录实际下注号码（随机/触发/冷热温/计画可审计）
ALTER TABLE cloud_bet_records
    ADD COLUMN bet_content TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN cloud_bet_records.bet_content IS '实际下注号码内容（多行按位，与方案内容同格式）';

-- 3) 运行类型枚举定版为 7 种（全部行统一）
UPDATE lottery_scheme_option_sets
SET run_types = '[
        {"value":"fixed_rotate","label":"定码轮换"},
        {"value":"adv_fixed_rotate","label":"高级定码轮换"},
        {"value":"adv_trigger_bet","label":"高级开某投某"},
        {"value":"hot_cold_warm","label":"冷热温出号"},
        {"value":"random_draw","label":"随机出号"},
        {"value":"builtin_plan","label":"内置计画"},
        {"value":"fixed_number","label":"固定号码"}
    ]'::jsonb,
    updated_at = now();

-- 4) 存量废弃运行类型映射（Q9=B：全部映射到高级定码轮换）
UPDATE scheme_definitions
SET config = jsonb_set(config, '{runTypeId}', '"adv_fixed_rotate"'::jsonb),
    updated_at = now()
WHERE config->>'runTypeId' IN ('batch_fixed', 'dynamic_chase', 'plan_follow');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cloud_bet_records DROP COLUMN IF EXISTS bet_content;
ALTER TABLE scheme_instances
    DROP COLUMN IF EXISTS last_direction,
    DROP COLUMN IF EXISTS current_pick,
    DROP COLUMN IF EXISTS pick_index;
-- +goose StatementEnd
