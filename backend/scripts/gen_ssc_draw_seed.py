#!/usr/bin/env python3
"""Generate 00077 SSC draw seed SQL (17 lotteries x 5 issues)."""

import csv
from pathlib import Path

root = Path(__file__).resolve().parents[1]

ssc_codes = []
with open(root / "docs/seeds/lottery_catalog.csv", encoding="utf-8-sig") as f:
    for row in csv.DictReader(f):
        if row["play_template"] == "ssc_std":
            ssc_codes.append(row["code"])

draws = [
    ("20231103027", "027", '["1","6","3","3","7"]', 20),
    ("20231103028", "028", '["2","2","9","0","3"]', 16),
    ("20231103029", "029", '["4","5","5","1","8"]', 23),
    ("20231103030", "030", '["8","1","0","6","4"]', 19),
    ("20231103031", "031", '["3","9","2","7","5"]', 26),
]

times = [
    "2026-06-08 10:20:00+00",
    "2026-06-08 10:21:00+00",
    "2026-06-08 10:22:00+00",
    "2026-06-08 10:23:00+00",
    "2026-06-08 10:24:00+00",
]

lines = []
for code in ssc_codes:
    for i, (issue, short, balls, sv) in enumerate(draws):
        lines.append(
            f"    ('{code}', '{issue}', '{short}', '{balls}'::jsonb, {sv}, '{times[i]}')"
        )

codes_sql = ", ".join(f"'{c}'" for c in ssc_codes)
issues_sql = ", ".join(f"'{d[0]}'" for d in draws)

header = f"""-- +goose Up
-- P2：ssc_std 17 彩种历史开奖种子

INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at) VALUES
"""
footer = f"""
ON CONFLICT (lottery_code, issue_no) DO NOTHING;

-- +goose Down
DELETE FROM lottery_draws
WHERE lottery_code IN ({codes_sql})
AND issue_no IN ({issues_sql});
"""

print(header + ",\n".join(lines) + footer)
