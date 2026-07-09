-- +goose Up
-- rules/v2 玩法类型模板 id：play_templates ↔ data[id].name

ALTER TABLE play_templates
    ADD COLUMN IF NOT EXISTS guaji_rules_type_id VARCHAR(8);

COMMENT ON COLUMN play_templates.guaji_rules_type_id IS '第三方 rules/v2 顶层键（玩法类型模板 id，如 1=时时彩）';

UPDATE play_templates SET guaji_rules_type_id = v.rid, label = v.lbl
FROM (VALUES
    ('ssc_std',  '1',  '时时彩'),
    ('syxw_std', '2',  '十一选五'),
    ('pk10_std', '3',  'PK10'),
    ('k3_std',   '4',  '快三'),
    ('pc28_std', '5',  'PC28'),
    ('lhc_std',  '8',  '六合彩')
) AS v(code, rid, lbl)
WHERE play_templates.code = v.code;

-- +goose Down
ALTER TABLE play_templates DROP COLUMN IF EXISTS guaji_rules_type_id;
