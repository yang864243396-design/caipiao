-- +goose Up
-- 同一玩法类型下短名重复（如任选下两个「直选复式」）时，改用 guajiFullName 入库消歧
-- 例：任二直选复式 / 任三直选复式 / 任选四直选复式
-- +goose StatementBegin

UPDATE sub_plays sp
SET label = TRIM(sp.segment_rule->>'guajiFullName')
WHERE COALESCE(TRIM(sp.segment_rule->>'guajiFullName'), '') <> ''
  AND EXISTS (
    SELECT 1
    FROM sub_plays sp2
    WHERE sp2.template_code = sp.template_code
      AND sp2.type_id = sp.type_id
      AND sp2.label = sp.label
      AND sp2.sub_id <> sp.sub_id
  );

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 短名回退需重新 rules-sync；此处不自动 Down
-- +goose StatementEnd
