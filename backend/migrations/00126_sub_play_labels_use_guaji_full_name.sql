-- +goose Up
-- 子玩法 label 一律使用 guajiFullName（无则保留原 label）
-- +goose StatementBegin

UPDATE sub_plays
SET label = TRIM(segment_rule->>'guajiFullName')
WHERE COALESCE(TRIM(segment_rule->>'guajiFullName'), '') <> ''
  AND label IS DISTINCT FROM TRIM(segment_rule->>'guajiFullName');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 短名回退需重新 rules-sync；此处不自动 Down
-- +goose StatementEnd
