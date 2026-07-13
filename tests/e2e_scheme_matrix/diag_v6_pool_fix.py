"""仅第三方：验证和值号池 / 混合组选 / 特殊号镜像选号。"""
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from browser.v6_app import V6App
from config import V6_STORAGE_STATE
from credentials import load_credentials
from play_profile import PickMode, danshi_sample, infer_pick_mode, prefer_pool_values

CASES = [
    ("前三码", "前三直选和值", ["12"], PickMode.POOL),
    ("前三码", "前三混合组选", None, PickMode.DANSHI),
    ("前三码", "前三特殊号", ["对子"], PickMode.ATTR),
]


def main() -> int:
    creds = load_credentials()
    failed = 0
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        state = V6_STORAGE_STATE if V6_STORAGE_STATE.is_file() else None
        if not state:
            print(
                f"[warn] 无 {V6_STORAGE_STATE.name}，将尝试自动登录；"
                "若站点开了 captcha，请先 python save_v6_session.py",
                flush=True,
            )
        ctx, v6 = V6App.launch(browser, creds.v6_url, storage_state=state)
        try:
            print("[diag] v6 login…", flush=True)
            v6.login(creds.v6_user, creds.v6_pass)
            print("[diag] open lottery…", flush=True)
            matched = v6.open_lottery_by_platform_label("波场一分彩")
            print(f"[diag] matched={matched}", flush=True)
            for play, sub, picks, mode in CASES:
                key = f"{play}/{sub}"
                print(f"[diag] start {key}", flush=True)
                try:
                    inferred = infer_pick_mode(play, sub)
                    labels = picks
                    danshi = ""
                    if mode == PickMode.DANSHI or inferred == PickMode.DANSHI:
                        danshi = danshi_sample(play, sub)
                        labels = [x.strip() for x in danshi.split(",") if x.strip()]
                        mode = PickMode.DANSHI
                    elif labels is None:
                        labels = (prefer_pool_values(play, sub) or ["12"])[:1]
                    v6.mirror_picks(
                        labels,
                        play_type_label=play,
                        sub_play_label=sub,
                        mode=mode,
                        danshi_text=danshi,
                    )
                    n = v6.read_preview_bet_count()
                    ok = n > 1 if "和值" in sub else n >= 1
                    print(
                        f"[{'ok' if ok else 'FAIL'}] {key}: v6={n} mode={mode.value} picks={labels}",
                        flush=True,
                    )
                    if not ok:
                        failed += 1
                except Exception as e:
                    failed += 1
                    print(f"[FAIL] {key}: {e}", flush=True)
        finally:
            ctx.close()
            browser.close()
    print(f"done failed={failed}/{len(CASES)}", flush=True)
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
