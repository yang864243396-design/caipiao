-- +goose Up
-- 波场秒彩哈希玩法 g017/390（尾数大小）：按五位开奖号码最后一位判定，0-4 小、5-9 大。
-- 该规则用于 WS 开奖后的本地策略推进；资金结算仍以第三方派彩为准。
-- 历史正式记录回放存在 3 条本地判定与存量第三方状态不一致，保留在 source_meta
-- 供管理员策略对账接口暴露，不能再因规则缺失让整条策略链静默失效。
INSERT INTO play_rule_specs (
    template_code, type_id, sub_id, lottery_code, rule_version,
    evaluator_key, evaluator_version, evaluation_spec, sample_cases,
    source_meta, strategy_enabled, published_at, updated_at
) VALUES (
    'fast_ssc_std', 'g017', '390', NULL, 1,
    'ssc.attribute', 1,
    '{"mode":"attribute","numberMin":0,"numberMax":9,"segmentStart":0,"segmentLen":5,"betMode":"daxiao","catalogSubId":"390"}'::jsonb,
    '[{"balls":["5","5","6","5","3"],"content":"小","hit":true},{"balls":["5","5","6","5","3"],"content":"大","hit":false}]'::jsonb,
    '{"migration":"00150","semantic":"final_digit_0_4_small_5_9_big","historicalReplay":{"matched":114,"total":117,"providerMismatchPeriods":["85309977","85310043","85310053"]}}'::jsonb,
    TRUE, now(), now()
)
ON CONFLICT (template_code, type_id, sub_id, lottery_code)
DO UPDATE SET
    rule_version = EXCLUDED.rule_version,
    evaluator_key = EXCLUDED.evaluator_key,
    evaluator_version = EXCLUDED.evaluator_version,
    evaluation_spec = EXCLUDED.evaluation_spec,
    sample_cases = EXCLUDED.sample_cases,
    source_meta = EXCLUDED.source_meta,
    strategy_enabled = TRUE,
    published_at = now(),
    updated_at = now();

INSERT INTO play_rule_spec_revisions (
    rule_spec_id, template_code, type_id, sub_id, lottery_code,
    revision, status, evaluator_key, evaluator_version, evaluation_spec,
    sample_cases, source_meta, actor, change_reason, verified_at, published_at
)
SELECT p.id, p.template_code, p.type_id, p.sub_id, p.lottery_code,
       1, 'published', p.evaluator_key, p.evaluator_version, p.evaluation_spec,
       p.sample_cases, p.source_meta, 'migration:00150',
       'publish declared hash-tail semantics and expose provider mismatches', now(), now()
FROM play_rule_specs p
WHERE p.template_code = 'fast_ssc_std'
  AND p.type_id = 'g017'
  AND p.sub_id = '390'
  AND p.lottery_code IS NULL
ON CONFLICT (template_code, type_id, sub_id, lottery_code, revision)
DO UPDATE SET
    rule_spec_id = EXCLUDED.rule_spec_id,
    status = 'published',
    evaluator_key = EXCLUDED.evaluator_key,
    evaluator_version = EXCLUDED.evaluator_version,
    evaluation_spec = EXCLUDED.evaluation_spec,
    sample_cases = EXCLUDED.sample_cases,
    source_meta = EXCLUDED.source_meta,
    actor = EXCLUDED.actor,
    change_reason = EXCLUDED.change_reason,
    verified_at = EXCLUDED.verified_at,
    published_at = EXCLUDED.published_at;

-- +goose Down
DELETE FROM play_rule_spec_revisions
WHERE template_code = 'fast_ssc_std'
  AND type_id = 'g017'
  AND sub_id = '390'
  AND lottery_code IS NULL
  AND revision = 1
  AND actor = 'migration:00150';

DELETE FROM play_rule_specs
WHERE template_code = 'fast_ssc_std'
  AND type_id = 'g017'
  AND sub_id = '390'
  AND lottery_code IS NULL
  AND source_meta ->> 'migration' = '00150';
