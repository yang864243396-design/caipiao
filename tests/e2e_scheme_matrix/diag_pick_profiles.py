"""快速验证：任选 / 和值 / 龙虎 / 特殊号 / 混合组选。"""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from bet_pick import apply_platform_pick
from browser.platform_app import PlatformApp
from browser.v6_app import V6App
from compare import compare_bet_counts
from config import V6_STORAGE_STATE
from credentials import load_credentials
from play_profile import infer_pick_mode

CASES = [
    ("前三码", "前三直选和值"),
    ("前三码", "前三混合组选"),
    ("前三码", "前三特殊号"),
    ("任选", "任二直选复式"),
    ("龙虎", "万千龙虎斗"),
]


def run_one(platform: PlatformApp, v6: V6App, play: str, sub: str) -> tuple[bool, str]:
    mode = infer_pick_mode(play, sub)
    name = platform.build_scheme_name("波场一分彩", "定码轮换", play, sub)
    # 每案前确认仍在登录态
    if "/login" in (platform.page.url or ""):
        raise RuntimeError("平台会话已失效，需重新登录")
    platform.open_custom_scheme_new()
    platform.fill_scheme_name(name)
    platform.open_and_confirm_picker("lottery", "波场一分彩")
    platform.open_and_confirm_picker("runType", "定码轮换")
    platform.open_and_confirm_picker("playType", play)
    platform.open_and_confirm_picker("subPlay", sub)
    platform.click_next()
    platform.fill_basic_config()
    if platform.page.locator("#scf-name").count():
        platform.page.locator("#scf-name").fill(name)
    platform.set_simple_bet_multiplier()
    plan = apply_platform_pick(platform, play_type_label=play, sub_play_label=sub)
    pn = platform.read_platform_bet_count()
    v6.open_lottery_by_platform_label("波场一分彩")
    v6.mirror_picks(
        plan.button_labels,
        line_picks=plan.line_picks,
        play_type_label=play,
        sub_play_label=sub,
        mode=plan.mode,
        danshi_text=plan.danshi_text,
        position_labels=plan.position_labels,
    )
    vn = v6.read_preview_bet_count()
    ok, msg = compare_bet_counts(pn, vn)
    return ok, f"mode={mode.value}->{plan.mode.value} {msg} notes={plan.notes}"


def main() -> int:
    creds = load_credentials()
    failed = 0
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        p_ctx, platform = PlatformApp.launch(browser, creds.platform_url)
        state = V6_STORAGE_STATE if V6_STORAGE_STATE.is_file() else None
        v_ctx, v6 = V6App.launch(browser, creds.v6_url, storage_state=state)
        try:
            print("[diag] platform login…", flush=True)
            platform.login(creds.platform_user, creds.platform_pass)
            print("[diag] guaji auth…", flush=True)
            platform.ensure_guaji_auth(creds.platform_api_base)
            print("[diag] v6 login…", flush=True)
            v6.login(creds.v6_user, creds.v6_pass)
            print("[diag] both logged in", flush=True)
            for play, sub in CASES:
                key = f"{play}/{sub}"
                print(f"[diag] start {key}", flush=True)
                try:
                    ok, msg = run_one(platform, v6, play, sub)
                    print(f"[{'ok' if ok else 'FAIL'}] {key}: {msg}", flush=True)
                    if not ok:
                        failed += 1
                except Exception as e:
                    failed += 1
                    print(f"[FAIL] {key}: {e}", flush=True)
        finally:
            p_ctx.close()
            v_ctx.close()
            browser.close()
    print(f"done failed={failed}/{len(CASES)}")
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
