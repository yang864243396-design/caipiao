"""单测 mirror_picks 和值。"""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from browser.v6_app import V6App
from credentials import load_credentials
from play_profile import PickMode


def main() -> int:
    c = load_credentials()
    with sync_playwright() as p:
        b = p.chromium.launch(headless=False)
        ctx, v6 = V6App.launch(b, c.v6_url)
        try:
            v6.login(c.v6_user, c.v6_pass)
            v6.open_lottery_by_platform_label("波场一分彩")
            print("before mirror", flush=True)
            v6.mirror_picks(
                ["12"],
                play_type_label="前三码",
                sub_play_label="前三直选和值",
                mode=PickMode.POOL,
            )
            n = v6.read_preview_bet_count()
            print("bet", n, flush=True)
            return 0 if n > 1 else 1
        finally:
            ctx.close()
            b.close()


if __name__ == "__main__":
    raise SystemExit(main())
