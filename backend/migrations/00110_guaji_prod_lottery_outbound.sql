-- +goose Up
-- 正式环境 www.v6hs1.com GET /api/games/new_lott 对齐 outbound_lottery_code（33 彩种保留在售）。
-- 13 个以太坊系列彩种 maintenance 下架，并暂停 running/pending 方案实例。
-- canonical 表见 catalogsync.V6hs1OutboundByCode。

-- 1) 正式 game_id
UPDATE lottery_catalog SET outbound_lottery_code = v.gid
FROM (VALUES
    ('hash_ffc_1m',     '19'),
    ('hash_ffc_3m',     '20'),
    ('hash_ffc_5m',     '21'),
    ('hash_jisu',       '23'),
    ('tron_jisu',       '24'),
    ('tron_ffc_1m',     '25'),
    ('tron_ffc_3m',     '26'),
    ('tron_ffc_5m',     '27'),
    ('bnb_ffc_1m',      '39'),
    ('bnb_ffc_3m',      '40'),
    ('bnb_ffc_5m',      '41'),
    ('tron_syxw',       '42'),
    ('tron_syxw_3m',    '43'),
    ('tron_syxw_5m',    '44'),
    ('bnb_syxw',        '48'),
    ('bnb_syxw_3m',     '49'),
    ('bnb_syxw_5m',     '50'),
    ('bnb_pk10_jisu',   '53'),
    ('bnb_pk10_5m',     '54'),
    ('tron_pk10_jisu',  '55'),
    ('tron_k3_jisu',    '59'),
    ('tron_k3_1m',      '60'),
    ('tron_k3_3m',      '61'),
    ('tron_k3_5m',      '62'),
    ('bnb_k3_1m',       '63'),
    ('bnb_k3_3m',       '64'),
    ('bnb_k3_5m',       '65'),
    ('tron_ffc_3s',     '73'),
    ('tron_ffc_6s',     '74'),
    ('tron_ffc_15s',    '75'),
    ('tron_lhc_1m',     '76'),
    ('tron_lhc_3m',     '77'),
    ('tron_lhc_5m',     '78')
) AS v(code, gid)
WHERE lottery_catalog.code = v.code;

-- 2) 以太坊系列 maintenance 下架
UPDATE lottery_catalog
SET sale_status = 'maintenance'::lottery_sale_status,
    on_sale = false,
    updated_at = now()
WHERE code IN (
    'eth_ffc_1m', 'eth_ffc_3m', 'eth_ffc_5m',
    'eth_ffc_new', 'eth_jisu',
    'eth_syxw', 'eth_syxw_3m', 'eth_syxw_5m',
    'eth_pk10_jisu', 'eth_pk10_5m',
    'eth_k3', 'eth_k3_3m', 'eth_k3_5m'
);

-- 3) 暂停绑定下架彩种的 running/pending 方案
UPDATE scheme_instances
SET status = 'paused',
    status_reason = 'maintenance',
    updated_at = now()
WHERE lottery_code IN (
    'eth_ffc_1m', 'eth_ffc_3m', 'eth_ffc_5m',
    'eth_ffc_new', 'eth_jisu',
    'eth_syxw', 'eth_syxw_3m', 'eth_syxw_5m',
    'eth_pk10_jisu', 'eth_pk10_5m',
    'eth_k3', 'eth_k3_3m', 'eth_k3_5m'
)
AND status IN ('running', 'pending');

-- +goose Down
-- 回退 outbound 至 iyes.dev 实测（00107）；以太坊彩种恢复 on_sale（不恢复方案状态）。
UPDATE lottery_catalog SET outbound_lottery_code = v.gid
FROM (VALUES
    ('hash_ffc_1m', '21'), ('hash_ffc_3m', '22'), ('hash_ffc_5m', '23'),
    ('hash_jisu', '25'), ('tron_jisu', '26'),
    ('tron_ffc_1m', '27'), ('tron_ffc_3m', '28'), ('tron_ffc_5m', '29'),
    ('bnb_ffc_1m', '41'), ('bnb_ffc_3m', '42'), ('bnb_ffc_5m', '43'),
    ('tron_syxw', '44'), ('tron_syxw_3m', '45'), ('tron_syxw_5m', '46'),
    ('bnb_syxw', '50'), ('bnb_syxw_3m', '51'), ('bnb_syxw_5m', '52'),
    ('bnb_pk10_jisu', '55'), ('bnb_pk10_5m', '56'),
    ('tron_pk10_jisu', '57'),
    ('tron_k3_jisu', '61'), ('tron_k3_1m', '62'), ('tron_k3_3m', '63'), ('tron_k3_5m', '64'),
    ('bnb_k3_1m', '65'), ('bnb_k3_3m', '66'), ('bnb_k3_5m', '67'),
    ('tron_ffc_3s', '75'), ('tron_ffc_6s', '76'), ('tron_ffc_15s', '77'),
    ('tron_lhc_1m', '78'), ('tron_lhc_3m', '79'), ('tron_lhc_5m', '80')
) AS v(code, gid)
WHERE lottery_catalog.code = v.code;

UPDATE lottery_catalog
SET sale_status = 'on_sale'::lottery_sale_status,
    on_sale = true,
    updated_at = now()
WHERE code IN (
    'eth_ffc_1m', 'eth_ffc_3m', 'eth_ffc_5m',
    'eth_ffc_new', 'eth_jisu',
    'eth_syxw', 'eth_syxw_3m', 'eth_syxw_5m',
    'eth_pk10_jisu', 'eth_pk10_5m',
    'eth_k3', 'eth_k3_3m', 'eth_k3_5m'
);
