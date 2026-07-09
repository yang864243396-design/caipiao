#!/usr/bin/env python3
import csv
import re
from pathlib import Path

root = Path(__file__).resolve().parents[1]
migs = [
    root / "migrations/00074_lhc_draw_seed_and_config_backfill.sql",
    root / "migrations/00075_p4_panel_type_and_draw_seed.sql",
    root / "migrations/00076_p4_remaining_draw_seed.sql",
]
sql = "".join(p.read_text(encoding="utf-8") for p in migs)
seed_codes = set(re.findall(r"\('([a-z0-9_]+)', '20231103", sql))

with open(root / "docs/seeds/lottery_catalog.csv", encoding="utf-8-sig") as f:
    rows = list(csv.DictReader(f))

by_tpl = {}
for r in rows:
    by_tpl.setdefault(r["play_template"], []).append(r["code"])

for tpl, codes in sorted(by_tpl.items()):
    missing = [c for c in codes if c not in seed_codes]
    print(f"{tpl}: {len(codes)} lotteries, seeds {len(codes)-len(missing)}/{len(codes)}")
    if missing:
        print("  missing:", ", ".join(missing))
