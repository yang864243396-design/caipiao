"""诊断：打开波场1分彩并探测选号/注数 DOM。"""
from __future__ import annotations

import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from browser.v6_app import V6App
from credentials import load_credentials
from name_match import match_lottery_label, normalize_lottery_name


def main() -> int:
    creds = load_credentials()
    out = Path(__file__).resolve().parent / "reports"
    out.mkdir(exist_ok=True)
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        ctx, app = V6App.launch(browser, creds.v6_url)
        page = app.page
        try:
            app.login(creds.v6_user, creds.v6_pass)
            labels = app.open_lottery_menu()
            print("menu_n", len(labels))
            print("menu_sample", labels[:40])
            target = "波场一分彩"
            matched = match_lottery_label(target, labels)
            print("matched", matched, "norm_target", normalize_lottery_name(target))
            # 强制点「波场1分彩」或「波场一分彩」
            want = matched or "波场1分彩"
            loc = page.get_by_text(want, exact=True)
            print("want_count", loc.count())
            for i in range(min(loc.count(), 5)):
                el = loc.nth(i)
                print("el", i, el.evaluate("e => ({tag:e.tagName, text:e.innerText, cls:e.className})"))
            # JS click first visible-ish
            clicked = False
            for i in range(min(loc.count(), 8)):
                try:
                    loc.nth(i).evaluate(
                        """(node) => {
                          const c = node.closest('a,button,[role=button],li,div') || node;
                          c.scrollIntoView({block:'center'});
                          c.click();
                        }"""
                    )
                    clicked = True
                    print("clicked idx", i)
                    break
                except Exception as e:
                    print("click fail", i, e)
            if not clicked:
                raise RuntimeError("click failed")
            page.wait_for_timeout(3000)
            app.dismiss_dialogs()
            print("url", page.url)
            page.screenshot(path=str(out / "v6_bc.png"), full_page=True)
            body = page.locator("body").inner_text()
            (out / "v6_bc.txt").write_text(body[:12000], encoding="utf-8")
            print("has 前三", "前三" in body, "直选", "直选" in body, "复式", "复式" in body)
            print("注", re.findall(r".{0,10}注.{0,15}", body)[:30])
            # try click 前三 / 直选复式
            for name in ("前三码", "前三", "三星", "直选复式", "复式"):
                t = page.get_by_text(name, exact=False)
                print("find", name, t.count())
            return 0
        finally:
            ctx.close()
            browser.close()


if __name__ == "__main__":
    raise SystemExit(main())
