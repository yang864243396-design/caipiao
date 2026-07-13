"""中止冒烟后：暂停实例并立刻对本端/第三方投注记录对账。"""
from __future__ import annotations

import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from api_client import PlatformApiClient
from browser.platform_app import PlatformApp
from browser.v6_app import V6App
from compare import compare_period_records
from credentials import load_credentials
from models import CaseKey, CaseResult, CaseStatus
from progress_store import load_progress, save_progress, untrack_instance, upsert_case
from report_excel import append_result, new_report_path
from v6_records import fetch_v6_bet_records

INSTANCE_ID = "inst-1-1783717606041"
LOTTERY = "波场一分彩"
SCHEME_NAME = "波场一分彩-定码轮换-前三码-前三直选复式_20260711050403"
KEY = CaseKey(
    lottery_code=LOTTERY,
    run_type_id="fixed_rotate",
    play_type_id="前三码",
    sub_play_id="前三直选复式",
    trigger_mode="-",
)


def main() -> int:
    creds = load_credentials()
    progress = load_progress()
    if not progress.report_path:
        progress.report_path = str(new_report_path())
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        p_ctx, platform = PlatformApp.launch(browser, creds.platform_url)
        v_ctx, v6 = V6App.launch(browser, creds.v6_url)
        api = PlatformApiClient(creds.platform_api_base)
        try:
            platform.login(creds.platform_user, creds.platform_pass)
            token = platform.read_access_token()
            if token:
                api.set_bearer(token)

            try:
                api.stop_instance(INSTANCE_ID)
                print("[stop] instance paused", INSTANCE_ID)
            except Exception as e:
                print("[stop] warn", e)

            platform_recs = api.fetch_scheme_bet_records(
                INSTANCE_ID, mode="real", days=7, limit=200
            )
            for r in platform_recs:
                r.lottery_label = LOTTERY
            print(f"[platform] records={len(platform_recs)}")
            for r in platform_recs[:8]:
                print(
                    " ",
                    r.period,
                    "amt=",
                    r.amount,
                    "mult=",
                    r.multiplier,
                    "win=",
                    r.win_status,
                    "content=",
                    (r.content or "")[:30],
                )

            v6.login(creds.v6_user, creds.v6_pass)
            v6_recs = fetch_v6_bet_records(
                v6, lottery_hint=LOTTERY, limit=100, game_id=19
            )
            print(f"[v6] records={len(v6_recs)}")
            for r in v6_recs[:8]:
                print(
                    " ",
                    r.period,
                    "amt=",
                    r.amount,
                    "content=",
                    (r.content or "")[:40],
                    "win=",
                    r.win_status,
                )

            p_periods = {r.period for r in platform_recs if r.period}
            v6_matched = [r for r in v6_recs if r.period in p_periods]
            ok, issues = compare_period_records(platform_recs, v6_matched)
            print(
                f"[compare] ok={ok} issues={len(issues)} "
                f"platform={len(platform_recs)} v6_matched={len(v6_matched)}"
            )
            for i in issues[:30]:
                print(
                    f"  - {i.period}/{i.field}: 平台={i.platform!r} 第三方={i.third_party!r}"
                )

            detail = "; ".join(
                f"{i.period}/{i.field}:平台={i.platform} 第三方={i.third_party}"
                for i in issues[:20]
            )
            if not platform_recs:
                detail = "本端无投注记录; " + detail
            if not v6_matched and platform_recs:
                detail = "第三方无匹配期号; " + detail

            passed = bool(ok and platform_recs)
            result = CaseResult(
                key=KEY,
                lottery_label=LOTTERY,
                run_type_label="定码轮换",
                play_type_label="前三码",
                sub_play_label="前三直选复式",
                trigger_mode_label="-",
                scheme_name=SCHEME_NAME,
                instance_id=INSTANCE_ID,
                status=CaseStatus.PASSED if passed else CaseStatus.COMPARE_FAILED,
                create_ok="ok",
                bet_count_ok="ok (aborted smoke)",
                start_ok="ok",
                target_periods=10,
                actual_periods=len(platform_recs),
                stop_reason="aborted",
                record_compare="passed" if passed else "failed",
                failure_detail="" if passed else detail.strip("; "),
            )
            upsert_case(progress, KEY.as_str(), result.to_progress_dict())
            untrack_instance(progress, INSTANCE_ID)
            save_progress(progress)
            append_result(Path(progress.report_path), result)
            print("[report]", progress.report_path)
            print(json.dumps(result.to_progress_dict(), ensure_ascii=False, indent=2))
            return 0 if passed else 1
        finally:
            api.close()
            p_ctx.close()
            v_ctx.close()
            browser.close()


if __name__ == "__main__":
    raise SystemExit(main())
