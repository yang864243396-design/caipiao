"""探针：抓 V6 登录 API 响应。"""
from __future__ import annotations

import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from credentials import load_credentials


def main() -> int:
    creds = load_credentials()
    responses: list[str] = []

    def on_response(resp) -> None:
        url = resp.url
        if any(k in url.lower() for k in ("login", "auth", "signin", "user", "member", "token")):
            try:
                body = resp.text()[:500]
            except Exception:
                body = "<no body>"
            responses.append(f"{resp.status} {url}\n{body}\n---")

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        ctx = browser.new_context(viewport={"width": 1400, "height": 900}, locale="zh-CN")
        page = ctx.new_page()
        page.on("response", on_response)
        page.goto(creds.v6_url, wait_until="domcontentloaded", timeout=90_000)
        page.wait_for_timeout(1500)
        for name in ("确定", "确认"):
            btn = page.get_by_role("button", name=name)
            if btn.count():
                try:
                    btn.first.click(timeout=1500)
                except Exception:
                    pass
        page.evaluate(
            """() => {
              const el = [...document.querySelectorAll('div,span,a,button')]
                .find(e => (e.textContent||'').trim() === '登录' && e.getBoundingClientRect().y < 100);
              if (el) el.click();
            }"""
        )
        page.wait_for_timeout(800)
        page.locator('input[name="username"], input[type="text"]').first.fill(creds.v6_user)
        page.locator('input[name="password"], input[type="password"]').first.fill(creds.v6_pass)
        page.locator("button").filter(has_text="登录").last.click()
        page.wait_for_timeout(8000)
        out = Path("reports/v6_login_api.txt")
        out.write_text("\n".join(responses) or "(no matching responses)", encoding="utf-8")
        print(f"wrote {out} n={len(responses)}", flush=True)
        for r in responses[:12]:
            print(r[:300], flush=True)
        # toast / message
        msgs = page.evaluate(
            """() => [...document.querySelectorAll('div,span')]
              .map(e => (e.innerText||'').trim())
              .filter(t => t && t.length < 40 && /错误|失败|密码|账号|验证|成功|无效/.test(t))
              .slice(0, 20)"""
        )
        print("msgs", msgs, flush=True)
        ctx.close()
        browser.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
