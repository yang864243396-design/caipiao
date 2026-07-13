"""跑期等待：按真实开奖节奏轮询本端投注记录。"""
from __future__ import annotations

import time
from dataclasses import dataclass
from typing import Callable

from api_client import PlatformApiClient
from config import CASE_WAIT_FACTOR, PERIOD_WAIT_INTERVALS
from models import BetRecord


@dataclass
class RunWaitResult:
    actual_periods: int
    stop_reason: str  # reached_target | early_stop | timeout | aborted
    status_reason: str = ""
    records: list[BetRecord] | None = None


def wait_for_periods(
    api: PlatformApiClient,
    *,
    instance_id: str,
    scheme_id: str,
    target_periods: int,
    interval_seconds: int,
    should_abort: Callable[[], bool] | None = None,
    poll_seconds: float | None = None,
) -> RunWaitResult:
    """
    轮询投注记录直至达到目标期数、实例非 running、超时或中止。
    scheme_id 用于 bet-records 路径（通常等于 instance id）。
    """
    if target_periods <= 0:
        return RunWaitResult(0, "early_stop", "目标期数为0")

    interval = max(30, int(interval_seconds or 60))
    poll = poll_seconds if poll_seconds is not None else min(30.0, interval / 2)
    deadline = time.time() + target_periods * interval * CASE_WAIT_FACTOR
    # 单期出现宽限：用于日志，整体仍受 deadline 约束
    _ = PERIOD_WAIT_INTERVALS

    last_n = 0
    while True:
        if should_abort and should_abort():
            recs = _safe_records(api, scheme_id)
            return RunWaitResult(len(recs), "aborted", "用户中断", recs)

        inst = api.get_instance(instance_id) or {}
        status = str(inst.get("status") or "")
        status_code = str(inst.get("statusReason") or "")
        status_label = str(inst.get("statusLabel") or "")
        # 展示优先中文 statusLabel（含「投注失败-…」）；判定仍看 code
        status_reason = status_label or status_code

        recs = _safe_records(api, scheme_id)
        # 只统计已有期号的记录
        settled = [r for r in recs if r.period and r.period != "—"]
        n = len(settled)
        if n != last_n:
            print(f"[wait] instance={instance_id} records={n}/{target_periods} status={status} reason={status_reason}")
            last_n = n

        if n >= target_periods:
            return RunWaitResult(n, "reached_target", status_reason, settled[:target_periods])

        reason_l = f"{status_code} {status_label}".lower()
        if any(
            x in reason_l
            for x in (
                "bet_failed",
                "auth",
                "error",
                "fail",
                "投注失败",
                "止损",
                "止盈",
                "资金",
                "stop_loss",
                "take_profit",
                "total_stop",
            )
        ):
            return RunWaitResult(n, "early_stop", status_reason or status, settled)

        if status and status not in ("running", "pending"):
            # paused / stopped / error 等
            return RunWaitResult(
                n,
                "early_stop",
                status_reason or status,
                settled,
            )

        if time.time() >= deadline:
            return RunWaitResult(n, "timeout", status_reason or "整案等待超时", settled)

        time.sleep(poll)


def _safe_records(api: PlatformApiClient, scheme_id: str) -> list[BetRecord]:
    try:
        return api.fetch_scheme_bet_records(scheme_id, mode="real", days=7, limit=100)
    except Exception as e:
        print(f"[wait] 拉记录失败: {e}")
        return []
