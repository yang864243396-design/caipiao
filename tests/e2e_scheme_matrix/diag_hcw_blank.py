"""诊断冷热温配置页空白。"""
from __future__ import annotations

from playwright.sync_api import sync_playwright

from browser.platform_app import PlatformApp
from credentials import load_credentials


def main() -> int:
    creds = load_credentials()
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        ctx, app = PlatformApp.launch(browser, creds.platform_url)
        errors: list[str] = []
        app.page.on("pageerror", lambda e: errors.append(f"pageerror:{e}"))
        app.page.on(
            "console",
            lambda m: errors.append(f"console:{m.type}:{m.text}")
            if m.type in ("error", "warning")
            else None,
        )
        app.login(creds.platform_user, creds.platform_pass)
        name = app.build_scheme_name("波场一分彩", "冷热温出号", "前三码", "前三直选复式")
        app.open_custom_scheme_new()
        app.fill_scheme_name(name)
        app.open_and_confirm_picker("lottery", "波场一分彩")
        app.open_and_confirm_picker("runType", "冷热温出号")
        app.open_and_confirm_picker("playType", "前三码")
        app.open_and_confirm_picker("subPlay", "前三直选复式")
        app.click_next()
        app.page.wait_for_timeout(3000)
        print("url", app.page.url)
        print("errors:")
        for e in errors[-30:]:
            print(" ", e[:400])
        html = app.page.evaluate("() => document.querySelector('#app')?.innerHTML?.slice(0, 800) || ''")
        print("app html:", html)
        print("body text len:", len(app.page.locator("body").inner_text() or ""))
        # compare fixed_rotate
        app.open_custom_scheme_new()
        name2 = app.build_scheme_name("波场一分彩", "定码轮换", "前三码", "前三直选复式")
        app.fill_scheme_name(name2)
        app.open_and_confirm_picker("lottery", "波场一分彩")
        app.open_and_confirm_picker("runType", "定码轮换")
        app.open_and_confirm_picker("playType", "前三码")
        app.open_and_confirm_picker("subPlay", "前三直选复式")
        app.click_next()
        app.page.wait_for_timeout(1500)
        print("fixed has funds", app.page.locator("#scf-funds").count())
        ctx.close()
        browser.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
