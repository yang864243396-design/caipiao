-- +goose Up
-- +goose StatementBegin
-- 连码是父玩法；二全中、二中特、特串、三中二、三全中均为其子玩法，不得提升为同级类型。
UPDATE sub_plays
SET type_id = 'g003',
    label = COALESCE(segment_rule->>'guajiFullName', label),
    sort_order = sub_id::integer - 278
WHERE template_code = 'lhc_std'
  AND type_id IN ('erquanzhong', 'erzhongte')
  AND sub_id IN ('279', '280', '281', '282', '283', '284', '285', '286', '287', '288', '289', '290');

WITH targets(sub_id, team, full_label) AS (
  VALUES
    ('279','二全中','二全中复式'), ('280','二全中','二全中拖头'),
    ('281','二全中','二全中生肖对碰'), ('282','二全中','二全中尾数对碰'),
    ('283','二全中','二全中生尾对碰'), ('284','二全中','二全中任意对碰'),
    ('285','二中特','二中特复式'), ('286','二中特','二中特拖头'),
    ('287','二中特','二中特生肖对碰'), ('288','二中特','二中特尾数对碰'),
    ('289','二中特','二中特生尾对碰'), ('290','二中特','二中特任意对碰')
)
UPDATE scheme_definitions d
SET config = jsonb_set(jsonb_set(jsonb_set(jsonb_set(jsonb_set(
  d.config, '{playTypeId}', to_jsonb('g003'::text), true), '{typeId}', to_jsonb('g003'::text), true),
  '{playTypeLabel}', to_jsonb('连码'::text), true), '{playMethodLabel}', to_jsonb(t.full_label), true),
  '{guajiGroup}', to_jsonb('连码'::text), true)
FROM targets t
WHERE d.config->>'playTemplate' = 'lhc_std'
  AND COALESCE(d.config->>'catalogSubId', d.config->>'subPlayId', d.config->>'subId') = t.sub_id
  AND COALESCE(d.config->>'playTypeId', d.config->>'typeId') IN ('erquanzhong', 'erzhongte');

WITH targets(sub_id, team, full_label) AS (
  VALUES
    ('279','二全中','二全中复式'), ('280','二全中','二全中拖头'),
    ('281','二全中','二全中生肖对碰'), ('282','二全中','二全中尾数对碰'),
    ('283','二全中','二全中生尾对碰'), ('284','二全中','二全中任意对碰'),
    ('285','二中特','二中特复式'), ('286','二中特','二中特拖头'),
    ('287','二中特','二中特生肖对碰'), ('288','二中特','二中特尾数对碰'),
    ('289','二中特','二中特生尾对碰'), ('290','二中特','二中特任意对碰')
)
UPDATE scheme_share_snapshots s
SET config = jsonb_set(jsonb_set(jsonb_set(jsonb_set(jsonb_set(
  s.config, '{playTypeId}', to_jsonb('g003'::text), true), '{typeId}', to_jsonb('g003'::text), true),
  '{playTypeLabel}', to_jsonb('连码'::text), true), '{playMethodLabel}', to_jsonb(t.full_label), true),
  '{guajiGroup}', to_jsonb('连码'::text), true),
    play_method = t.full_label
FROM targets t
WHERE s.config->>'playTemplate' = 'lhc_std'
  AND COALESCE(s.config->>'catalogSubId', s.config->>'subPlayId', s.config->>'subId') = t.sub_id
  AND COALESCE(s.config->>'playTypeId', s.config->>'typeId') IN ('erquanzhong', 'erzhongte');

DELETE FROM play_types
WHERE template_code = 'lhc_std'
  AND type_id IN ('erquanzhong', 'erzhongte');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 目录层级由 rules/v2 同步器维护；不恢复错误的并列层级。
-- +goose StatementEnd
