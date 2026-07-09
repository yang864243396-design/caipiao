-- +goose Up
-- 波场 3/6/15 秒彩：第三方 rules/v2 type 7「快速彩」，与 type 1「时时彩」rule_id 集合不同。
-- +goose StatementBegin
INSERT INTO play_templates (code, label, version, guaji_rules_type_id)
VALUES ('fast_ssc_std', '快速彩', 1, '7')
ON CONFLICT (code) DO UPDATE SET
  label = EXCLUDED.label,
  guaji_rules_type_id = EXCLUDED.guaji_rules_type_id;

UPDATE lottery_catalog
SET play_template = 'fast_ssc_std'
WHERE code IN ('tron_ffc_3s', 'tron_ffc_6s', 'tron_ffc_15s');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE lottery_catalog
SET play_template = 'ssc_std'
WHERE code IN ('tron_ffc_3s', 'tron_ffc_6s', 'tron_ffc_15s');
-- +goose StatementEnd
