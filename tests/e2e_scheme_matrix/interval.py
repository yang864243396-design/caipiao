"""draw_interval → 目标期数 / 跳过。"""
from __future__ import annotations

import re
from dataclasses import dataclass

from config import (
    ADV_TRIGGER_RUN_TYPE,
    FAST_INTERVAL_MAX_SECONDS,
    TARGET_PERIODS_FAST,
    TARGET_PERIODS_SLOW,
)


@dataclass(frozen=True)
class IntervalDecision:
    raw: str
    seconds: int | None
    target_periods: int
    skip: bool
    skip_reason: str = ""


_UNIT = {
    "s": 1,
    "sec": 1,
    "secs": 1,
    "m": 60,
    "min": 60,
    "mins": 60,
    "h": 3600,
    "hr": 3600,
    "hour": 3600,
    "hours": 3600,
}


def parse_draw_interval(raw: str) -> IntervalDecision:
    text = (raw or "").strip().lower()
    if not text or text in {"jisu", "unknown", "null", "none"}:
        return IntervalDecision(
            raw=text,
            seconds=None,
            target_periods=0,
            skip=True,
            skip_reason=f"draw_interval 无法用于判定: {raw!r}",
        )
    m = re.fullmatch(r"(\d+)\s*([a-z]+)?", text)
    if not m:
        return IntervalDecision(
            raw=text,
            seconds=None,
            target_periods=0,
            skip=True,
            skip_reason=f"draw_interval 无法解析: {raw!r}",
        )
    n = int(m.group(1))
    unit = (m.group(2) or "m").lower()
    if unit not in _UNIT:
        return IntervalDecision(
            raw=text,
            seconds=None,
            target_periods=0,
            skip=True,
            skip_reason=f"draw_interval 未知单位: {raw!r}",
        )
    seconds = n * _UNIT[unit]
    if seconds <= FAST_INTERVAL_MAX_SECONDS:
        target = TARGET_PERIODS_FAST
    else:
        target = TARGET_PERIODS_SLOW
    return IntervalDecision(
        raw=text, seconds=seconds, target_periods=target, skip=False
    )


def target_periods_for_case(run_type_id: str, global_target: int) -> int:
    if run_type_id == ADV_TRIGGER_RUN_TYPE:
        return max(1, (global_target + 3) // 4)  # ceil(n/4)
    return global_target
