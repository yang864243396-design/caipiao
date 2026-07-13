"""快速验证：基础配置填充、高级定码局数、组三注数镜像。"""
from __future__ import annotations

import traceback

from playwright.sync_api import sync_playwright

from bet_pick import apply_platform_pick
from browser.platform_app import PlatformApp
from browser.v6_app import V6App
from compare import compare_bet_counts
from config import V6_STORAGE_STATE
from credentials import load_credentials


def main() -> int:
    creds = load_credentials()
    fails = 0
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        try:
            # ---- 1) 冷热温：#scf-funds + hcw 选号 ----
            p_ctx, platform = PlatformApp.launch(browser, creds.platform_url)
            try:
                platform.login(creds.platform_user, creds.platform_pass)
                platform.ensure_guaji_auth(creds.platform_api_base)
                name = platform.build_scheme_name("波场一分彩", "冷热温出号", "前三码", "前三直选复式")
                platform.open_custom_scheme_new()
                platform.fill_scheme_name(name)
                platform.open_and_confirm_picker("lottery", "波场一分彩")
                platform.open_and_confirm_picker("runType", "冷热温出号")
                platform.open_and_confirm_picker("playType", "前三码")
                platform.open_and_confirm_picker("subPlay", "前三直选复式")
                platform.click_next()
                platform.fill_basic_config()
                platform.set_simple_bet_multiplier()
                plan = apply_platform_pick(
                    platform,
                    play_type_label="前三码",
                    sub_play_label="前三直选复式",
                    run_type_id="hot_cold_warm",
                )
                print("[ok] hot_cold_warm funds+pick", plan.notes)
            except Exception:
                fails += 1
                print("[fail] hot_cold_warm")
                traceback.print_exc()
            finally:
                p_ctx.close()

            # ---- 2) 高级定码轮换：添加局数 ----
            p_ctx, platform = PlatformApp.launch(browser, creds.platform_url)
            try:
                platform.login(creds.platform_user, creds.platform_pass)
                name = platform.build_scheme_name("波场一分彩", "高级定码轮换", "前三码", "前三直选复式")
                platform.open_custom_scheme_new()
                platform.fill_scheme_name(name)
                platform.open_and_confirm_picker("lottery", "波场一分彩")
                platform.open_and_confirm_picker("runType", "高级定码轮换")
                platform.open_and_confirm_picker("playType", "前三码")
                platform.open_and_confirm_picker("subPlay", "前三直选复式")
                platform.click_next()
                platform.fill_basic_config()
                platform.set_simple_bet_multiplier()
                plan = apply_platform_pick(
                    platform,
                    play_type_label="前三码",
                    sub_play_label="前三直选复式",
                    run_type_id="adv_fixed_rotate",
                )
                print("[ok] adv_fixed_rotate jushu", plan.notes, "labels", plan.button_labels)
            except Exception:
                fails += 1
                print("[fail] adv_fixed_rotate")
                traceback.print_exc()
            finally:
                p_ctx.close()

            # ---- 3) 组三注数镜像 ----
            p_ctx, platform = PlatformApp.launch(browser, creds.platform_url)
            v_state = V6_STORAGE_STATE if V6_STORAGE_STATE.is_file() else None
            v_ctx, v6 = V6App.launch(browser, creds.v6_url, storage_state=v_state)
            try:
                platform.login(creds.platform_user, creds.platform_pass)
                v6.login(creds.v6_user, creds.v6_pass)
                name = platform.build_scheme_name("波场一分彩", "定码轮换", "前三码", "前三组三")
                platform.open_custom_scheme_new()
                platform.fill_scheme_name(name)
                platform.open_and_confirm_picker("lottery", "波场一分彩")
                platform.open_and_confirm_picker("runType", "定码轮换")
                platform.open_and_confirm_picker("playType", "前三码")
                platform.open_and_confirm_picker("subPlay", "前三组三")
                platform.click_next()
                platform.fill_basic_config()
                platform.set_simple_bet_multiplier()
                plan = apply_platform_pick(
                    platform,
                    play_type_label="前三码",
                    sub_play_label="前三组三",
                    run_type_id="fixed_rotate",
                )
                platform_n = platform.read_platform_bet_count()
                v6.open_lottery_by_platform_label("波场一分彩")
                v6.mirror_picks(
                    plan.button_labels,
                    line_picks=plan.line_picks,
                    play_type_label="前三码",
                    sub_play_label="前三组三",
                    mode=plan.mode,
                )
                v6_n = v6.read_preview_bet_count()
                ok, msg = compare_bet_counts(platform_n, v6_n)
                print(f"[{'ok' if ok else 'fail'}] 组三 platform={platform_n} v6={v6_n} {msg}")
                if not ok:
                    fails += 1
            except Exception:
                fails += 1
                print("[fail] 组三")
                traceback.print_exc()
            finally:
                p_ctx.close()
                v_ctx.close()
        finally:
            browser.close()
    print(f"done fails={fails}")
    return 1 if fails else 0


if __name__ == "__main__":
    raise SystemExit(main())
