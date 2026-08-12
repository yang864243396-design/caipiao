-- +goose Up
-- +goose StatementBegin
-- 00141 在部分既有库中已有版本记录但未包含拆分内容；本迁移幂等补齐实际目录与历史配置。
INSERT INTO play_types (template_code, type_id, label, sort_order, panel_type, enabled)
VALUES
  ('lhc_std', 'erquanzhong', '二全中', 2, NULL, true),
  ('lhc_std', 'erzhongte', '二中特', 3, NULL, true)
ON CONFLICT (template_code, type_id) DO UPDATE
SET label = EXCLUDED.label, sort_order = EXCLUDED.sort_order, enabled = true;

UPDATE sub_plays
SET type_id = CASE
      WHEN sub_id IN ('279', '280', '281', '282', '283', '284') THEN 'erquanzhong'
      WHEN sub_id IN ('285', '286', '287', '288', '289', '290') THEN 'erzhongte'
    END,
    label = CASE
      WHEN sub_id IN ('279', '285') THEN '复式'
      WHEN sub_id IN ('280', '286') THEN '拖头'
      WHEN sub_id IN ('281', '287') THEN '生肖对碰'
      WHEN sub_id IN ('282', '288') THEN '尾数对碰'
      WHEN sub_id IN ('283', '289') THEN '生尾对碰'
      WHEN sub_id IN ('284', '290') THEN '任意对碰'
    END,
    sort_order = CASE
      WHEN sub_id IN ('279', '285') THEN 1
      WHEN sub_id IN ('280', '286') THEN 2
      WHEN sub_id IN ('281', '287') THEN 3
      WHEN sub_id IN ('282', '288') THEN 4
      WHEN sub_id IN ('283', '289') THEN 5
      WHEN sub_id IN ('284', '290') THEN 6
    END
WHERE template_code = 'lhc_std'
  AND type_id = 'g003'
  AND sub_id IN ('279', '280', '281', '282', '283', '284', '285', '286', '287', '288', '289', '290');

WITH targets(sub_id, type_id, type_label, sub_label) AS (
  VALUES
    ('279','erquanzhong','二全中','复式'), ('280','erquanzhong','二全中','拖头'),
    ('281','erquanzhong','二全中','生肖对碰'), ('282','erquanzhong','二全中','尾数对碰'),
    ('283','erquanzhong','二全中','生尾对碰'), ('284','erquanzhong','二全中','任意对碰'),
    ('285','erzhongte','二中特','复式'), ('286','erzhongte','二中特','拖头'),
    ('287','erzhongte','二中特','生肖对碰'), ('288','erzhongte','二中特','尾数对碰'),
    ('289','erzhongte','二中特','生尾对碰'), ('290','erzhongte','二中特','任意对碰')
)
UPDATE scheme_definitions d
SET config = jsonb_set(jsonb_set(jsonb_set(jsonb_set(jsonb_set(jsonb_set(
  d.config, '{playTypeId}', to_jsonb(t.type_id), true), '{typeId}', to_jsonb(t.type_id), true),
  '{playTypeLabel}', to_jsonb(t.type_label), true), '{playMethodLabel}', to_jsonb(t.sub_label), true),
  '{guajiGroup}', to_jsonb(t.type_label), true), '{guajiTeam}', to_jsonb(t.type_label), true)
FROM targets t
WHERE d.config->>'playTemplate' = 'lhc_std'
  AND COALESCE(d.config->>'catalogSubId', d.config->>'subPlayId', d.config->>'subId') = t.sub_id
  AND COALESCE(d.config->>'playTypeId', d.config->>'typeId') = 'g003';

WITH targets(sub_id, type_id, type_label, sub_label) AS (
  VALUES
    ('279','erquanzhong','二全中','复式'), ('280','erquanzhong','二全中','拖头'),
    ('281','erquanzhong','二全中','生肖对碰'), ('282','erquanzhong','二全中','尾数对碰'),
    ('283','erquanzhong','二全中','生尾对碰'), ('284','erquanzhong','二全中','任意对碰'),
    ('285','erzhongte','二中特','复式'), ('286','erzhongte','二中特','拖头'),
    ('287','erzhongte','二中特','生肖对碰'), ('288','erzhongte','二中特','尾数对碰'),
    ('289','erzhongte','二中特','生尾对碰'), ('290','erzhongte','二中特','任意对碰')
)
UPDATE scheme_share_snapshots s
SET config = jsonb_set(jsonb_set(jsonb_set(jsonb_set(jsonb_set(jsonb_set(
  s.config, '{playTypeId}', to_jsonb(t.type_id), true), '{typeId}', to_jsonb(t.type_id), true),
  '{playTypeLabel}', to_jsonb(t.type_label), true), '{playMethodLabel}', to_jsonb(t.sub_label), true),
  '{guajiGroup}', to_jsonb(t.type_label), true), '{guajiTeam}', to_jsonb(t.type_label), true)
FROM targets t
WHERE s.config->>'playTemplate' = 'lhc_std'
  AND COALESCE(s.config->>'catalogSubId', s.config->>'subPlayId', s.config->>'subId') = t.sub_id
  AND COALESCE(s.config->>'playTypeId', s.config->>'typeId') = 'g003';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE sub_plays
SET type_id = 'g003', label = COALESCE(segment_rule->>'guajiFullName', label)
WHERE template_code = 'lhc_std'
  AND type_id IN ('erquanzhong', 'erzhongte')
  AND sub_id IN ('279', '280', '281', '282', '283', '284', '285', '286', '287', '288', '289', '290');
-- +goose StatementEnd
