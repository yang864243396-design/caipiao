"""快速诊断：本平台登录 + 打开自创方案页。"""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from browser.platform_app import PlatformApp
from credentials import load_credentials


def main() -> int:
    creds = load_credentials()
    print(f"url={creds.platform_url}", flush=True)
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        ctx, app = PlatformApp.launch(browser, creds.platform_url)
        try:
            print("goto login…", flush=True)
            app.page.goto(f"{creds.platform_url}/login", wait_until="domcontentloaded", timeout=30_000)
            print(f"title={app.page.title()} url={app.page.url}", flush=True)
            print("login…", flush=True)
            app.login(creds.platform_user, creds.platform_pass)
            print(f"after login url={app.page.url}", flush=True)
            print("open custom scheme…", flush=True)
            app.open_custom_scheme_new()
            print(f"scheme page url={app.page.url}", flush=True)
            app.open_picker("lottery")
            labels = app.list_picker_option_labels()
            print(f"lotteries={len(labels)} sample={labels[:5]}", flush=True)
            app.close_picker()
            print("diag ok", flush=True)
            return 0
        except Exception as e:
            print(f"diag fail: {e}", flush=True)
            try:
                print(f"url={app.page.url}", flush=True)
            except Exception:
                pass
            return 1
        finally:
            ctx.close()
            browser.close()


if __name__ == "__main__":
    raise SystemExit(main())
