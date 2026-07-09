-- +goose Up
-- hash.iyes.dev 实测 new_lott id 与文档 §8 序号全面不一致（53 个已对接彩种均需修正）。
-- 00087 按文档序号 1–47 写入；00088 误将 tron_ffc_1m 设为 29（实为波场五分彩）。
-- 以下按 GET /api/games/new_lott 逐条对齐（2026-06-21）；canonical 表见 catalogsync.IyesDevOutboundByCode。
UPDATE lottery_catalog SET outbound_lottery_code = v.gid
FROM (VALUES
    ('hash_ffc_1m',     '21'),
    ('hash_ffc_3m',     '22'),
    ('hash_ffc_5m',     '23'),
    ('hash_jisu',       '25'),
    ('tron_jisu',       '26'),
    ('tron_ffc_1m',     '27'),
    ('tron_ffc_3m',     '28'),
    ('tron_ffc_5m',     '29'),
    ('eth_jisu',        '37'),
    ('eth_ffc_1m',      '38'),
    ('eth_ffc_3m',      '39'),
    ('eth_ffc_5m',      '40'),
    ('bnb_ffc_1m',      '41'),
    ('bnb_ffc_3m',      '42'),
    ('bnb_ffc_5m',      '43'),
    ('tron_syxw',       '44'),
    ('tron_syxw_3m',    '45'),
    ('tron_syxw_5m',    '46'),
    ('eth_syxw',        '47'),
    ('eth_syxw_3m',     '48'),
    ('eth_syxw_5m',     '49'),
    ('bnb_syxw',        '50'),
    ('bnb_syxw_3m',     '51'),
    ('bnb_syxw_5m',     '52'),
    ('eth_pk10_jisu',   '53'),
    ('eth_pk10_5m',     '54'),
    ('bnb_pk10_jisu',   '55'),
    ('bnb_pk10_5m',     '56'),
    ('tron_pk10_jisu',  '57'),
    ('eth_k3',          '58'),
    ('eth_k3_3m',       '59'),
    ('eth_k3_5m',       '60'),
    ('tron_k3_jisu',    '61'),
    ('tron_k3_1m',      '62'),
    ('tron_k3_3m',      '63'),
    ('tron_k3_5m',      '64'),
    ('bnb_k3_1m',       '65'),
    ('bnb_k3_3m',       '66'),
    ('bnb_k3_5m',       '67'),
    ('eth_ffc_new',     '68'),
    ('taiwan_ssc_5m',   '69'),
    ('taiwan_pk10',     '70'),
    ('taiwan_pc28',     '71'),
    ('tron_ffc_3s',     '75'),
    ('tron_ffc_6s',     '76'),
    ('tron_ffc_15s',    '77'),
    ('tron_lhc_1m',     '78'),
    ('tron_lhc_3m',     '79'),
    ('tron_lhc_5m',     '80'),
    ('tron_lhc',        '81')
) AS v(code, gid)
WHERE lottery_catalog.code = v.code;

-- +goose Down
-- 回退至 00087 文档序号 + 00088 对 tron_ffc_1m 的修改（不推荐用于 iyes.dev）
UPDATE lottery_catalog SET outbound_lottery_code = v.gid
FROM (VALUES
    ('tron_ffc_1m',  '29'),
    ('tron_ffc_3m',  '2'),
    ('tron_ffc_5m',  '3'),
    ('hash_ffc_1m',  '4'),
    ('hash_ffc_3m',  '5'),
    ('hash_ffc_5m',  '6')
) AS v(code, gid)
WHERE lottery_catalog.code = v.code;
