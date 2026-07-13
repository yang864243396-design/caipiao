"""用例与结果数据模型。"""
from __future__ import annotations

from dataclasses import asdict, dataclass, field
from enum import Enum
from typing import Any


class CaseStatus(str, Enum):
    PENDING = "pending"
    SKIPPED = "skipped"
    CREATE_FAILED = "create_failed"
    BET_COUNT_MISMATCH = "bet_count_mismatch"
    START_FAILED = "start_failed"
    EARLY_STOP = "early_stop"
    COMPARE_FAILED = "compare_failed"
    PASSED = "passed"


@dataclass(frozen=True)
class CaseKey:
    lottery_code: str
    run_type_id: str
    play_type_id: str
    sub_play_id: str
    trigger_mode: str = "-"  # adv_trigger 时为 always_pos 等

    def as_str(self) -> str:
        return "|".join(
            (
                self.lottery_code,
                self.run_type_id,
                self.play_type_id,
                self.sub_play_id,
                self.trigger_mode or "-",
            )
        )

    @classmethod
    def parse(cls, s: str) -> CaseKey:
        parts = s.split("|")
        if len(parts) != 5:
            raise ValueError(f"非法用例键: {s}")
        return cls(*parts)


@dataclass
class CaseSpec:
    key: CaseKey
    lottery_label: str
    run_type_label: str
    play_type_label: str
    sub_play_label: str
    trigger_mode_label: str = "-"
    draw_interval: str = ""
    target_periods: int = 0
    skip_reason: str = ""


@dataclass
class BetRecord:
    lottery_label: str
    period: str
    content: str
    bet_count: int | None = None
    amount: float | None = None
    win_status: str = ""
    payout: float | None = None
    draw_numbers: str = ""
    multiplier: float | None = None


@dataclass
class CaseResult:
    key: CaseKey
    lottery_label: str = ""
    run_type_label: str = ""
    play_type_label: str = ""
    sub_play_label: str = ""
    trigger_mode_label: str = "-"
    scheme_name: str = ""
    definition_id: str = ""
    instance_id: str = ""
    status: CaseStatus = CaseStatus.PENDING
    create_ok: str = ""
    bet_count_ok: str = ""
    start_ok: str = ""
    target_periods: int = 0
    actual_periods: int = 0
    stop_reason: str = ""
    record_compare: str = ""
    failure_detail: str = ""

    def to_progress_dict(self) -> dict[str, Any]:
        d = asdict(self)
        d["key"] = self.key.as_str()
        d["status"] = self.status.value
        return d

    def to_excel_row(self) -> list[Any]:
        return [
            self.lottery_label,
            self.run_type_label,
            self.play_type_label,
            self.sub_play_label,
            self.trigger_mode_label,
            self.scheme_name,
            self.create_ok,
            self.bet_count_ok,
            self.start_ok,
            self.target_periods,
            self.actual_periods,
            self.stop_reason,
            self.record_compare,
            self.failure_detail,
        ]


@dataclass
class ProgressState:
    version: int = 1
    cases: dict[str, dict[str, Any]] = field(default_factory=dict)
    active_instance_ids: list[str] = field(default_factory=list)
    report_path: str = ""
