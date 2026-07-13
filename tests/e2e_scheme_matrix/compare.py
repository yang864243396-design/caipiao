"""注数与期记录对比。"""
from __future__ import annotations

from dataclasses import dataclass

from config import AMOUNT_PRECISION
from models import BetRecord


def amounts_equal(a: float | None, b: float | None) -> bool:
    if a is None and b is None:
        return True
    if a is None or b is None:
        return False
    return abs(float(a) - float(b)) <= AMOUNT_PRECISION + 1e-12


@dataclass
class CompareIssue:
    period: str
    field: str
    platform: str
    third_party: str


def compare_bet_counts(platform_n: int, v6_n: int) -> tuple[bool, str]:
    """注数必须双方都 >0，且相等；以第三方预览注数为准。"""
    if platform_n <= 0 or v6_n <= 0:
        return False, f"invalid bet count (must >0) platform={platform_n} v6={v6_n}"
    if platform_n == v6_n:
        return True, f"ok platform={platform_n} v6={v6_n}"
    return False, f"mismatch platform={platform_n} v6={v6_n} (以第三方为准)"


def compare_period_records(
    platform: list[BetRecord],
    v6: list[BetRecord],
) -> tuple[bool, list[CompareIssue]]:
    """配对键：同一彩种下按期号配对（彩种对齐由调用方保证）。

    同期限可能有多笔历史注单（矩阵连续跑多玩法）。按内容/金额/倍数打分取最佳匹配，
    避免「期号撞车」把和值配到单式。

    本端无记录时不拿第三方历史注单刷屏「平台=缺失」；只报一条根因。
    有本端记录时，仅对比本端出现的期号（第三方多出的历史单忽略）。
    """
    issues: list[CompareIssue] = []
    if not platform:
        if v6:
            issues.append(CompareIssue("-", "存在性", "本端无投注记录", f"第三方有{len(v6)}条"))
        else:
            issues.append(CompareIssue("-", "存在性", "本端无投注记录", "第三方亦无"))
        return False, issues

    v_by_period: dict[str, list[BetRecord]] = {}
    for r in v6:
        if not r.period:
            continue
        v_by_period.setdefault(r.period, []).append(r)

    used: set[int] = set()
    for pr in sorted((r for r in platform if r.period), key=lambda r: r.period):
        period = pr.period
        candidates = [c for c in v_by_period.get(period, []) if id(c) not in used]
        if not candidates:
            issues.append(CompareIssue(period, "存在性", "有", "缺失"))
            continue
        vr = _pick_best_match(pr, candidates)
        used.add(id(vr))
        if _norm_content(pr.content) != _norm_content(vr.content):
            issues.append(CompareIssue(period, "投注内容", pr.content, vr.content))
        if pr.bet_count is not None and vr.bet_count is not None and pr.bet_count != vr.bet_count:
            issues.append(
                CompareIssue(period, "注数", str(pr.bet_count), str(vr.bet_count))
            )
        if not amounts_equal(pr.amount, vr.amount):
            if pr.amount is not None and vr.amount is not None:
                issues.append(CompareIssue(period, "金额", str(pr.amount), str(vr.amount)))
        if _norm(pr.win_status) and _norm(vr.win_status):
            if _norm(pr.win_status) != _norm(vr.win_status):
                issues.append(
                    CompareIssue(period, "中奖状态", pr.win_status, vr.win_status)
                )
        if pr.payout is not None and vr.payout is not None:
            if not amounts_equal(pr.payout, vr.payout):
                issues.append(CompareIssue(period, "派奖", str(pr.payout), str(vr.payout)))
        if _norm(pr.draw_numbers) and _norm(vr.draw_numbers):
            if _norm(pr.draw_numbers) != _norm(vr.draw_numbers):
                issues.append(
                    CompareIssue(period, "开奖号码", pr.draw_numbers, vr.draw_numbers)
                )
        if pr.multiplier is not None and vr.multiplier is not None:
            if not amounts_equal(pr.multiplier, vr.multiplier):
                issues.append(
                    CompareIssue(period, "倍数", str(pr.multiplier), str(vr.multiplier))
                )
    return len(issues) == 0, issues


def _pick_best_match(pr: BetRecord, candidates: list[BetRecord]) -> BetRecord:
    """同期限多候选时，优先内容一致，其次金额/注数/倍数。"""

    def score(vr: BetRecord) -> tuple[int, float]:
        s = 0
        pc = _norm_content(pr.content)
        vc = _norm_content(vr.content)
        if pc and vc and pc == vc:
            s += 100
        elif pc and vc and (pc in vc or vc in pc):
            s += 40
        elif _content_compatible(pc, vc):
            s += 60
        if amounts_equal(pr.amount, vr.amount):
            s += 50
        if pr.bet_count is not None and vr.bet_count is not None and pr.bet_count == vr.bet_count:
            s += 20
        if amounts_equal(pr.multiplier, vr.multiplier):
            s += 10
        # 金额差越小越好（次级排序取负距离）
        dist = 0.0
        if pr.amount is not None and vr.amount is not None:
            dist = abs(float(pr.amount) - float(vr.amount))
        return (s, -dist)

    return max(candidates, key=score)


def _content_compatible(pc: str, vc: str) -> bool:
    """本端多行/逗号选号与第三方 wire 的宽松兼容（去逗号后相同）。"""
    if not pc or not vc:
        return False
    a = "".join(ch for ch in pc if ch.isdigit() or ch in "对子豹子顺子龙虎和大单小双")
    b = "".join(ch for ch in vc if ch.isdigit() or ch in "对子豹子顺子龙虎和大单小双")
    return bool(a) and a == b


def _norm(s: str) -> str:
    return "".join((s or "").split())


def _norm_content(s: str) -> str:
    """对齐本端多行选号与第三方逗号 wire。"""
    text = (s or "").replace("\r\n", "\n").strip()
    if "\n" in text:
        parts = []
        for line in text.split("\n"):
            digits = "".join(ch for ch in line if ch.isdigit())
            parts.append(digits)
        return ",".join(parts)
    return _norm(text)
