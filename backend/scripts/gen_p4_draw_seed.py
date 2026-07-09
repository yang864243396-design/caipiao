#!/usr/bin/env python3
"""Generate 00076 P4 draw seed SQL (22 lotteries x 5 issues)."""

syxw = [
    ("20231103027", "027", '["01","04","06","08","11"]', 30),
    ("20231103028", "028", '["02","05","07","09","10"]', 33),
    ("20231103029", "029", '["01","03","06","08","11"]', 29),
    ("20231103030", "030", '["02","04","07","09","10"]', 32),
    ("20231103031", "031", '["01","05","06","08","11"]', 31),
]
pk10 = [
    ("20231103027", "027", '["3","7","1","9","5","2","8","4","6","10"]', 55),
    ("20231103028", "028", '["8","2","5","1","10","4","7","3","6","9"]', 55),
    ("20231103029", "029", '["6","4","9","2","7","1","10","3","5","8"]', 55),
    ("20231103030", "030", '["10","1","4","8","3","6","9","2","7","5"]', 55),
    ("20231103031", "031", '["5","9","2","7","1","10","4","8","3","6"]', 55),
]
k3 = [
    ("20231103027", "027", '["2","4","6"]', 12),
    ("20231103028", "028", '["1","3","5"]', 9),
    ("20231103029", "029", '["2","2","5"]', 9),
    ("20231103030", "030", '["3","4","6"]', 13),
    ("20231103031", "031", '["1","1","6"]', 8),
]

groups = {
    "syxw": [
        "tron_syxw_3m",
        "tron_syxw_5m",
        "eth_syxw",
        "eth_syxw_3m",
        "eth_syxw_5m",
        "bnb_syxw",
        "bnb_syxw_3m",
        "bnb_syxw_5m",
    ],
    "pk10": [
        "eth_pk10_5m",
        "bnb_pk10_jisu",
        "bnb_pk10_5m",
        "tron_pk10_jisu",
        "taiwan_pk10",
    ],
    "k3": [
        "eth_k3_3m",
        "eth_k3_5m",
        "tron_k3_jisu",
        "tron_k3_1m",
        "tron_k3_3m",
        "tron_k3_5m",
        "bnb_k3_1m",
        "bnb_k3_3m",
        "bnb_k3_5m",
    ],
}

rows = []
for code in groups["syxw"]:
    for issue, short, balls, sv in syxw:
        rows.append((code, issue, short, balls, sv))
for code in groups["pk10"]:
    for issue, short, balls, sv in pk10:
        rows.append((code, issue, short, balls, sv))
for code in groups["k3"]:
    for issue, short, balls, sv in k3:
        rows.append((code, issue, short, balls, sv))

times = [
    "2026-06-08 10:10:00+00",
    "2026-06-08 10:11:00+00",
    "2026-06-08 10:12:00+00",
    "2026-06-08 10:13:00+00",
    "2026-06-08 10:14:00+00",
]

lines = []
for i, (code, issue, short, balls, sv) in enumerate(rows):
    t = times[i % 5]
    lines.append(
        f"    ('{code}', '{issue}', '{short}', '{balls}'::jsonb, {sv}, '{t}')"
    )

header = """-- +goose Up
-- P4：其余 22 个彩种开奖种子（00075 已覆盖 4 个代表彩种）

INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at) VALUES
"""
footer = """
ON CONFLICT (lottery_code, issue_no) DO NOTHING;

-- +goose Down
DELETE FROM lottery_draws
WHERE lottery_code IN (
    'tron_syxw_3m', 'tron_syxw_5m', 'eth_syxw', 'eth_syxw_3m', 'eth_syxw_5m',
    'bnb_syxw', 'bnb_syxw_3m', 'bnb_syxw_5m',
    'eth_pk10_5m', 'bnb_pk10_jisu', 'bnb_pk10_5m', 'tron_pk10_jisu', 'taiwan_pk10',
    'eth_k3_3m', 'eth_k3_5m', 'tron_k3_jisu', 'tron_k3_1m', 'tron_k3_3m', 'tron_k3_5m',
    'bnb_k3_1m', 'bnb_k3_3m', 'bnb_k3_5m'
)
AND issue_no IN ('20231103027', '20231103028', '20231103029', '20231103030', '20231103031');
"""

out = header + ",\n".join(lines) + footer
print(out)
print(f"-- total rows: {len(rows)}", file=__import__("sys").stderr)
