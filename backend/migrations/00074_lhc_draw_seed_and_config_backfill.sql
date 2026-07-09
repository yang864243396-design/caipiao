-- +goose Up
-- P3：LHC 历史开奖种子 + 方案/分享快照 config 补齐 catalog 玩法字段

INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at) VALUES
    ('tron_lhc_1m', '20260608027', '027', '["08","19","26","33","41","02","49"]'::jsonb, 178, '2026-06-08 09:55:00+00'),
    ('tron_lhc_1m', '20260608028', '028', '["05","14","22","31","38","11","46"]'::jsonb, 167, '2026-06-08 09:56:00+00'),
    ('tron_lhc_1m', '20260608029', '029', '["01","16","24","35","42","09","48"]'::jsonb, 175, '2026-06-08 09:57:00+00'),
    ('tron_lhc_1m', '20260608030', '030', '["07","18","27","34","40","13","45"]'::jsonb, 184, '2026-06-08 09:58:00+00'),
    ('tron_lhc_1m', '20260608031', '031', '["03","12","25","33","41","07","49"]'::jsonb, 170, '2026-06-08 09:59:00+00'),
    ('tron_lhc_3m', '20260608027', '027', '["06","15","23","32","39","04","47"]'::jsonb, 166, '2026-06-08 09:50:00+00'),
    ('tron_lhc_3m', '20260608028', '028', '["02","17","21","36","43","10","44"]'::jsonb, 173, '2026-06-08 09:53:00+00'),
    ('tron_lhc_3m', '20260608029', '029', '["09","20","28","37","41","05","49"]'::jsonb, 189, '2026-06-08 09:56:00+00'),
    ('tron_lhc_3m', '20260608030', '030', '["11","14","26","30","42","08","48"]'::jsonb, 179, '2026-06-08 09:59:00+00'),
    ('tron_lhc_3m', '20260608031', '031', '["03","12","25","33","41","07","49"]'::jsonb, 170, '2026-06-08 10:02:00+00'),
    ('tron_lhc_5m', '20260608027', '027', '["04","13","25","29","40","06","45"]'::jsonb, 162, '2026-06-08 09:45:00+00'),
    ('tron_lhc_5m', '20260608028', '028', '["10","19","22","34","38","12","46"]'::jsonb, 181, '2026-06-08 09:50:00+00'),
    ('tron_lhc_5m', '20260608029', '029', '["01","15","27","31","44","08","49"]'::jsonb, 175, '2026-06-08 09:55:00+00'),
    ('tron_lhc_5m', '20260608030', '030', '["05","18","24","35","39","11","47"]'::jsonb, 179, '2026-06-08 10:00:00+00'),
    ('tron_lhc_5m', '20260608031', '031', '["03","12","25","33","41","07","49"]'::jsonb, 170, '2026-06-08 10:05:00+00'),
    ('tron_lhc', '20260608027', '027', '["07","16","23","32","40","02","48"]'::jsonb, 168, '2026-06-08 09:40:00+00'),
    ('tron_lhc', '20260608028', '028', '["09","14","21","36","43","06","44"]'::jsonb, 173, '2026-06-08 09:45:00+00'),
    ('tron_lhc', '20260608029', '029', '["02","11","28","34","42","13","49"]'::jsonb, 179, '2026-06-08 09:50:00+00'),
    ('tron_lhc', '20260608030', '030', '["08","17","26","33","38","05","46"]'::jsonb, 173, '2026-06-08 09:55:00+00'),
    ('tron_lhc', '20260608031', '031', '["03","12","25","33","41","07","49"]'::jsonb, 170, '2026-06-08 10:00:00+00')
ON CONFLICT (lottery_code, issue_no) DO NOTHING;

UPDATE scheme_definitions sd
SET config =
    sd.config
    || jsonb_build_object('playTemplate', lc.play_template)
    || jsonb_build_object(
        'typeId', COALESCE(NULLIF(sd.config->>'typeId', ''), sd.config->>'playTypeId'),
        'subId', COALESCE(NULLIF(sd.config->>'subId', ''), sd.config->>'subPlayId'),
        'playTypeId', COALESCE(NULLIF(sd.config->>'playTypeId', ''), sd.config->>'typeId'),
        'subPlayId', COALESCE(NULLIF(sd.config->>'subPlayId', ''), sd.config->>'subId')
    )
    || CASE
        WHEN sp.bet_mode IS NOT NULL AND sp.bet_mode <> '' THEN jsonb_build_object('betMode', sp.bet_mode)
        ELSE '{}'::jsonb
    END
FROM lottery_catalog lc, sub_plays sp
WHERE sd.lottery_code = lc.code
  AND sp.template_code = lc.play_template
  AND sp.type_id = COALESCE(NULLIF(sd.config->>'typeId', ''), sd.config->>'playTypeId')
  AND sp.sub_id = COALESCE(NULLIF(sd.config->>'subId', ''), sd.config->>'subPlayId')
  AND (sd.config->>'playTemplate' IS NULL OR sd.config->>'playTemplate' = '')
  AND COALESCE(NULLIF(sd.config->>'playTypeId', ''), sd.config->>'typeId', '') <> ''
  AND COALESCE(NULLIF(sd.config->>'subPlayId', ''), sd.config->>'subId', '') <> '';

UPDATE scheme_share_snapshots ss
SET config =
    ss.config
    || jsonb_build_object('playTemplate', lc.play_template)
    || jsonb_build_object(
        'typeId', COALESCE(NULLIF(ss.config->>'typeId', ''), ss.config->>'playTypeId'),
        'subId', COALESCE(NULLIF(ss.config->>'subId', ''), ss.config->>'subPlayId'),
        'playTypeId', COALESCE(NULLIF(ss.config->>'playTypeId', ''), ss.config->>'typeId'),
        'subPlayId', COALESCE(NULLIF(ss.config->>'subPlayId', ''), ss.config->>'subId')
    )
    || CASE
        WHEN sp.bet_mode IS NOT NULL AND sp.bet_mode <> '' THEN jsonb_build_object('betMode', sp.bet_mode)
        ELSE '{}'::jsonb
    END
FROM lottery_catalog lc, sub_plays sp
WHERE ss.lottery_code = lc.code
  AND sp.template_code = lc.play_template
  AND sp.type_id = COALESCE(NULLIF(ss.config->>'typeId', ''), ss.config->>'playTypeId')
  AND sp.sub_id = COALESCE(NULLIF(ss.config->>'subId', ''), ss.config->>'subPlayId')
  AND (ss.config->>'playTemplate' IS NULL OR ss.config->>'playTemplate' = '')
  AND COALESCE(NULLIF(ss.config->>'playTypeId', ''), ss.config->>'typeId', '') <> ''
  AND COALESCE(NULLIF(ss.config->>'subPlayId', ''), ss.config->>'subId', '') <> '';

-- +goose Down
DELETE FROM lottery_draws
WHERE lottery_code IN ('tron_lhc_1m', 'tron_lhc_3m', 'tron_lhc_5m', 'tron_lhc')
  AND issue_no BETWEEN '20260608027' AND '20260608031';
