-- name: ResolvePublishedPlayRuleSpec :one
SELECT id,
       template_code,
       type_id,
       sub_id,
       lottery_code,
       rule_version,
       evaluator_key,
       evaluator_version,
       evaluation_spec,
       sample_cases,
       source_meta,
       strategy_enabled,
       published_at,
       updated_at
FROM play_rule_specs
WHERE template_code = sqlc.arg(template_code)
  AND type_id = sqlc.arg(type_id)
  AND sub_id = sqlc.arg(sub_id)
  AND (lottery_code = sqlc.arg(lottery_code) OR lottery_code IS NULL)
ORDER BY CASE WHEN lottery_code = sqlc.arg(lottery_code) THEN 0 ELSE 1 END
LIMIT 1;

-- name: ListEnabledPlayRuleSpecs :many
SELECT id,
       template_code,
       type_id,
       sub_id,
       lottery_code,
       rule_version,
       evaluator_key,
       evaluator_version,
       evaluation_spec,
       sample_cases,
       source_meta,
       strategy_enabled,
       published_at,
       updated_at
FROM play_rule_specs
WHERE strategy_enabled
ORDER BY template_code, type_id, sub_id, lottery_code NULLS FIRST;

-- name: ListPlayRuleImportCandidates :many
SELECT sp.template_code,
       sp.type_id,
       sp.sub_id,
       COALESCE(sp.segment_rule ->> 'guajiRuleId', sp.outbound_play_code, sp.sub_id) AS rule_id,
       COALESCE(sp.segment_rule ->> 'guajiFullName', sp.label) AS full_name
FROM sub_plays sp
WHERE sp.enabled
ORDER BY sp.template_code, sp.type_id, sp.sub_id;

-- name: NextPlayRuleSpecRevision :one
SELECT COALESCE(MAX(revision), 0)::int + 1 AS next_revision
FROM play_rule_spec_revisions
WHERE template_code = sqlc.arg(template_code)
  AND type_id = sqlc.arg(type_id)
  AND sub_id = sqlc.arg(sub_id)
  AND lottery_code IS NOT DISTINCT FROM sqlc.narg(lottery_code);

-- name: InsertPlayRuleSpecRevision :one
INSERT INTO play_rule_spec_revisions (
    rule_spec_id, template_code, type_id, sub_id, lottery_code,
    revision, status, evaluator_key, evaluator_version,
    evaluation_spec, sample_cases, source_meta, actor, change_reason
) VALUES (
    sqlc.narg(rule_spec_id), sqlc.arg(template_code), sqlc.arg(type_id), sqlc.arg(sub_id), sqlc.narg(lottery_code),
    sqlc.arg(revision), sqlc.arg(status), sqlc.arg(evaluator_key), sqlc.arg(evaluator_version),
    sqlc.arg(evaluation_spec), sqlc.arg(sample_cases), sqlc.arg(source_meta), sqlc.arg(actor), sqlc.arg(change_reason)
)
RETURNING id, status, revision, created_at;

-- name: MarkPlayRuleSpecRevisionVerified :execrows
UPDATE play_rule_spec_revisions
SET status = 'verified', verified_at = now()
WHERE id = sqlc.arg(id)
  AND status = 'draft';

-- name: GetPlayRuleSpecRevisionForPublish :one
SELECT id,
       rule_spec_id,
       template_code,
       type_id,
       sub_id,
       lottery_code,
       revision,
       status,
       evaluator_key,
       evaluator_version,
       evaluation_spec,
       sample_cases,
       source_meta,
       actor,
       change_reason,
       verified_at,
       published_at,
       created_at
FROM play_rule_spec_revisions
WHERE id = sqlc.arg(id)
FOR UPDATE;

-- name: UpsertPublishedPlayRuleSpec :one
INSERT INTO play_rule_specs (
    template_code, type_id, sub_id, lottery_code, rule_version,
    evaluator_key, evaluator_version, evaluation_spec, sample_cases,
    source_meta, strategy_enabled, published_at, updated_at
) VALUES (
    sqlc.arg(template_code), sqlc.arg(type_id), sqlc.arg(sub_id), sqlc.narg(lottery_code), sqlc.arg(rule_version),
    sqlc.arg(evaluator_key), sqlc.arg(evaluator_version), sqlc.arg(evaluation_spec), sqlc.arg(sample_cases),
    sqlc.arg(source_meta), sqlc.arg(strategy_enabled), now(), now()
)
ON CONFLICT (template_code, type_id, sub_id, lottery_code)
DO UPDATE SET
    rule_version = EXCLUDED.rule_version,
    evaluator_key = EXCLUDED.evaluator_key,
    evaluator_version = EXCLUDED.evaluator_version,
    evaluation_spec = EXCLUDED.evaluation_spec,
    sample_cases = EXCLUDED.sample_cases,
    source_meta = EXCLUDED.source_meta,
    strategy_enabled = EXCLUDED.strategy_enabled,
    published_at = now(),
    updated_at = now()
RETURNING id, rule_version, strategy_enabled, updated_at;

-- name: MarkPlayRuleSpecRevisionPublished :execrows
UPDATE play_rule_spec_revisions
SET status = 'published', published_at = now()
WHERE id = sqlc.arg(id)
  AND status = 'verified';
