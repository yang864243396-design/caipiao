"""验证 v6 前三码直选复式镜像选号与已选注数。"""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from browser.v6_app import V6App
from credentials import load_credentials


def main() -> int:
    creds = load_credentials()
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        ctx, app = V6App.launch(browser, creds.v6_url)
        try:
            app.login(creds.v6_user, creds.v6_pass)
            matched = app.open_lottery_by_platform_label("波场一分彩")
            print("matched", matched)
            app.mirror_picks(
                ["0", "1", "2"],
                line_picks=[["0", "1", "3"], ["0"], ["0"]],
                play_type_label="前三码",
                sub_play_label="前三直选复式",
            )
            n = app.read_preview_bet_count()
            print("v6_bet_count", n)
            assert n > 0, "v6 bet count must > 0"
            print("v6 pick ok")
            return 0
        except Exception as e:
            print("fail", e)
            try:
                app.page.screenshot(
                    path=str(Path(__file__).resolve().parent / "reports" / "v6_fail.png")
                )
            except Exception:
                pass
            return 1
        finally:
            ctx.close()
            browser.close()


if __name__ == "__main__":
    raise SystemExit(main())
