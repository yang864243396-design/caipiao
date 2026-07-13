"""探针：V6 登录卡在何处。"""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from credentials import load_credentials


def main() -> int:
    creds = load_credentials()
    out = Path(__file__).resolve().parent / "reports"
    out.mkdir(exist_ok=True)
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        ctx = browser.new_context(viewport={"width": 1400, "height": 900}, locale="zh-CN")
        page = ctx.new_page()
        page.goto(creds.v6_url, wait_until="domcontentloaded", timeout=90_000)
        page.wait_for_timeout(2000)
        page.screenshot(path=str(out / "v6_probe_home.png"), full_page=True)
        # dismiss 确定
        for name in ("确定", "确认", "我知道了"):
            btn = page.get_by_role("button", name=name)
            if btn.count():
                try:
                    btn.first.click(timeout=2000)
                except Exception:
                    pass
        page.wait_for_timeout(500)
        # open login
        page.evaluate(
            """() => {
              const el = [...document.querySelectorAll('div,span,a,button')]
                .find(e => (e.textContent||'').trim() === '登录' && e.getBoundingClientRect().y < 100);
              if (el) el.click();
            }"""
        )
        page.wait_for_timeout(1000)
        page.screenshot(path=str(out / "v6_probe_login_open.png"))
        # fill
        page.locator('input[name="username"], input[type="text"]').first.fill(creds.v6_user)
        page.locator('input[name="password"], input[type="password"]').first.fill(creds.v6_pass)
        page.locator("button").filter(has_text="登录").last.click()
        for i in range(30):
            page.wait_for_timeout(1000)
            body = page.locator("body").inner_text()
            print(f"t={i+1}s has_user={creds.v6_user in body} login_pending={'登录中' in body} "
                  f"captcha={'验证码' in body or 'captcha' in body.lower()} "
                  f"maintain={'维护' in body}", flush=True)
            if creds.v6_user in body and "登录中" not in body:
                page.screenshot(path=str(out / "v6_probe_ok.png"))
                print("LOGIN OK", flush=True)
                ctx.close()
                browser.close()
                return 0
            if i in (5, 15, 25):
                page.screenshot(path=str(out / f"v6_probe_t{i}.png"))
        page.screenshot(path=str(out / "v6_probe_fail.png"))
        # frames
        print("frames:", [f.url for f in page.frames][:10], flush=True)
        print("FAIL body head:", body[:300].replace("\n", " | "), flush=True)
        ctx.close()
        browser.close()
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
