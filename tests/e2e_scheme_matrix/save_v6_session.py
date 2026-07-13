"""手动登录 V6 并保存 storage_state（过阿里云验证码后复用）。

用法：
  .venv\\Scripts\\python.exe save_v6_session.py

浏览器打开后请在 180 秒内完成登录；成功后写入 v6_storage_state.json。
"""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from config import V6_STORAGE_STATE
from credentials import load_credentials


def main() -> int:
    creds = load_credentials()
    print(
        f"将打开 {creds.v6_url}，请手动登录账号 {creds.v6_user}（含验证码）。\n"
        f"登录成功后会话保存到 {V6_STORAGE_STATE}",
        flush=True,
    )
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        ctx = browser.new_context(viewport={"width": 1400, "height": 900}, locale="zh-CN")
        page = ctx.new_page()
        page.goto(creds.v6_url, wait_until="domcontentloaded", timeout=90_000)
        ok = False
        for i in range(180):
            page.wait_for_timeout(1000)
            body = ""
            try:
                body = page.locator("body").inner_text() or ""
            except Exception:
                pass
            if creds.v6_user in body and "账号登录" not in body[:80]:
                # 顶部仍可能有「登录」文案；以用户名出现且无密码框为准
                has_pwd = page.locator('input[type="password"]').count()
                visible_pwd = False
                if has_pwd:
                    try:
                        visible_pwd = page.locator('input[type="password"]').first.is_visible(timeout=300)
                    except Exception:
                        visible_pwd = False
                if not visible_pwd:
                    ok = True
                    print(f"[ok] 检测到已登录 t={i+1}s", flush=True)
                    break
            if i % 15 == 0:
                print(f"[wait] {i}s …请在浏览器完成登录/验证码", flush=True)
        if not ok:
            print("超时未检测到登录成功", flush=True)
            ctx.close()
            browser.close()
            return 1
        page.wait_for_timeout(1000)
        ctx.storage_state(path=str(V6_STORAGE_STATE))
        print(f"saved {V6_STORAGE_STATE}", flush=True)
        ctx.close()
        browser.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
