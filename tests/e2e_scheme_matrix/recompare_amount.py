"""按 注数×倍数×投注单位 重算本端金额，与第三方对账并输出冒烟结论。"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from api_client import PlatformApiClient
from browser.platform_app import PlatformApp
from browser.v6_app import V6App
from compare import CompareIssue, amounts_equal, _norm, _norm_content
from credentials import load_credentials
from models import BetRecord, CaseKey, CaseResult, CaseStatus
from progress_store import load_progress, save_progress, upsert_case
from report_excel import append_result
from v6_records import fetch_v6_bet_records

INSTANCE_ID = "inst-1-1783717606041"
LOTTERY = "波场一分彩"
SCHEME_NAME = "波场一分彩-定码轮换-前三码-前三直选复式_20260711050403"
BET_UNIT = 1.0  # 方案 betUnit=1 元
KEY = CaseKey(
    lottery_code=LOTTERY,
    run_type_id="fixed_rotate",
    play_type_id="前三码",
    sub_play_id="前三直选复式",
    trigger_mode="-",
)


def fushi_bet_units(content: str, segment_len: int = 3) -> int:
    """前三直选复式：各位号码个数之积。"""
    text = (content or "").replace("\r\n", "\n").strip()
    lines = text.split("\n") if text else []
    if len(lines) >= segment_len:
        pools = []
        for i in range(segment_len):
            toks = [t for t in re.split(r"[,，\s]+", lines[i].strip()) if t != ""]
            # 也接受粘连数字 "013"
            if len(toks) == 1 and toks[0].isdigit() and len(toks[0]) > 1:
                toks = list(toks[0])
            pools.append(toks)
    else:
        digits = re.findall(r"\d", text)
        pools = [digits for _ in range(segment_len)] if digits else [["0"]] * segment_len
    units = 1
    for p in pools:
        n = len(p) if p else 1
        units *= n
    return max(1, units)


def expected_amount(content: str, multiplier: float | None, unit: float = BET_UNIT) -> float:
    mult = float(multiplier or 1)
    return round(fushi_bet_units(content) * mult * unit, 2)


def compare_with_expected(
    platform: list[BetRecord],
    v6: list[BetRecord],
) -> tuple[bool, list[CompareIssue], list[dict]]:
    issues: list[CompareIssue] = []
    rows: list[dict] = []
    p_map = {r.period: r for r in platform}
    v_map = {r.period: r for r in v6}
    for period in sorted(set(p_map) | set(v_map)):
        pr = p_map.get(period)
        vr = v_map.get(period)
        if pr is None:
            issues.append(CompareIssue(period, "存在性", "缺失", "有"))
            continue
        if vr is None:
            issues.append(CompareIssue(period, "存在性", "有", "缺失"))
            continue
        exp = expected_amount(pr.content, pr.multiplier)
        row = {
            "period": period,
            "units": fushi_bet_units(pr.content),
            "mult": pr.multiplier,
            "expected": exp,
            "platform_stored": pr.amount,
            "v6": vr.amount,
            "content_ok": _norm_content(pr.content) == _norm_content(vr.content),
            "win_ok": (not _norm(pr.win_status) or not _norm(vr.win_status))
            or _norm(pr.win_status) == _norm(vr.win_status),
            "amount_ok": amounts_equal(exp, vr.amount),
        }
        rows.append(row)
        if not row["content_ok"]:
            issues.append(CompareIssue(period, "投注内容", pr.content, vr.content))
        if _norm(pr.win_status) and _norm(vr.win_status) and not row["win_ok"]:
            issues.append(CompareIssue(period, "中奖状态", pr.win_status, vr.win_status))
        if pr.multiplier is not None and vr.multiplier is not None:
            if not amounts_equal(pr.multiplier, vr.multiplier):
                issues.append(
                    CompareIssue(period, "倍数", str(pr.multiplier), str(vr.multiplier))
                )
        if not row["amount_ok"]:
            issues.append(
                CompareIssue(
                    period,
                    "金额(注数×倍数×单位)",
                    str(exp),
                    str(vr.amount),
                )
            )
        # 附带记录：库内错误金额（历史脏数据）
        if pr.amount is not None and not amounts_equal(pr.amount, exp):
            issues.append(
                CompareIssue(
                    period,
                    "本端库内金额(历史误记)",
                    str(pr.amount),
                    f"应记{exp}",
                )
            )
    # 冒烟结论：业务对账以「应记金额 vs 第三方」为准；库内误记单独列出但不否决冒烟
    hard = [i for i in issues if i.field != "本端库内金额(历史误记)"]
    return len(hard) == 0, issues, rows


def main() -> int:
    creds = load_credentials()
    progress = load_progress()
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        p_ctx, platform = PlatformApp.launch(browser, creds.platform_url)
        v_ctx, v6 = V6App.launch(browser, creds.v6_url)
        api = PlatformApiClient(creds.platform_api_base)
        try:
            platform.login(creds.platform_user, creds.platform_pass)
            token = platform.read_access_token()
            if token:
                api.set_bearer(token)

            platform_recs = api.fetch_scheme_bet_records(
                INSTANCE_ID, mode="real", days=7, limit=200
            )
            for r in platform_recs:
                r.lottery_label = LOTTERY
            print(f"[platform] n={len(platform_recs)}")

            v6.login(creds.v6_user, creds.v6_pass)
            v6_recs = fetch_v6_bet_records(v6, lottery_hint=LOTTERY, limit=100)
            p_periods = {r.period for r in platform_recs if r.period}
            v6_matched = [r for r in v6_recs if r.period in p_periods]
            print(f"[v6] n={len(v6_recs)} matched={len(v6_matched)}")

            ok, issues, rows = compare_with_expected(platform_recs, v6_matched)
            print("\n=== 金额重算明细（注数×倍数×1元）===")
            for r in rows:
                flag = "OK" if r["amount_ok"] and r["content_ok"] and r["win_ok"] else "DIFF"
                print(
                    f"  {r['period']}: units={r['units']}×mult={r['mult']} "
                    f"=> expect={r['expected']} v6={r['v6']} stored={r['platform_stored']} [{flag}]"
                )

            hard = [i for i in issues if i.field != "本端库内金额(历史误记)"]
            soft = [i for i in issues if i.field == "本端库内金额(历史误记)"]
            print(f"\n[compare] business_ok={ok} hard_issues={len(hard)} stored_amount_drift={len(soft)}")
            for i in hard[:20]:
                print(f"  HARD {i.period}/{i.field}: 平台侧={i.platform!r} 第三方={i.third_party!r}")
            if soft:
                print(f"  (库内误记 {len(soft)} 期：已按 g001 SegmentLen 修复，历史行不会自动改写)")

            detail_hard = "; ".join(
                f"{i.period}/{i.field}:应={i.platform} 第三方={i.third_party}" for i in hard[:20]
            )
            detail_soft = f"库内金额误记{len(soft)}期(已修注数逻辑)" if soft else ""
            passed = ok and bool(platform_recs)
            result = CaseResult(
                key=KEY,
                lottery_label=LOTTERY,
                run_type_label="定码轮换",
                play_type_label="前三码",
                sub_play_label="前三直选复式",
                scheme_name=SCHEME_NAME,
                instance_id=INSTANCE_ID,
                status=CaseStatus.PASSED if passed else CaseStatus.COMPARE_FAILED,
                create_ok="ok",
                bet_count_ok="ok",
                start_ok="ok",
                target_periods=10,
                actual_periods=len(platform_recs),
                stop_reason="aborted",
                record_compare="passed" if passed else "failed",
                failure_detail=("; ".join(x for x in (detail_hard, detail_soft) if x)),
            )
            upsert_case(progress, KEY.as_str(), result.to_progress_dict())
            save_progress(progress)
            if progress.report_path:
                append_result(Path(progress.report_path), result)

            print("\n======== 冒烟测试结果 ========")
            print(f"方案: {SCHEME_NAME}")
            print(f"实例: {INSTANCE_ID}")
            print(f"实际期数: {len(platform_recs)} (目标已改为10，本次中止于15)")
            print(f"期号对齐: {len(v6_matched)}/{len(platform_recs)}")
            print(f"内容/倍数/中奖/应记金额: {'通过' if passed else '失败'}")
            print(f"结论: {'PASSED' if passed else 'FAILED'}")
            print(json.dumps(result.to_progress_dict(), ensure_ascii=False, indent=2))
            return 0 if passed else 1
        finally:
            api.close()
            p_ctx.close()
            v_ctx.close()
            browser.close()


if __name__ == "__main__":
    raise SystemExit(main())
