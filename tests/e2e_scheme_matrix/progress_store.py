"""progress.json 读写与续跑过滤。"""
from __future__ import annotations

import json
from pathlib import Path
from typing import Iterable

from config import PROGRESS_PATH
from models import CaseStatus, ProgressState


def load_progress(path: Path = PROGRESS_PATH) -> ProgressState:
    if not path.exists():
        return ProgressState()
    raw = json.loads(path.read_text(encoding="utf-8"))
    return ProgressState(
        version=int(raw.get("version", 1)),
        cases=dict(raw.get("cases") or {}),
        active_instance_ids=list(raw.get("active_instance_ids") or []),
        report_path=str(raw.get("report_path") or ""),
    )


def save_progress(state: ProgressState, path: Path = PROGRESS_PATH) -> None:
    path.write_text(
        json.dumps(
            {
                "version": state.version,
                "cases": state.cases,
                "active_instance_ids": state.active_instance_ids,
                "report_path": state.report_path,
            },
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )


def is_passed(state: ProgressState, case_key: str) -> bool:
    row = state.cases.get(case_key) or {}
    return row.get("status") == CaseStatus.PASSED.value or row.get("record_compare") == "passed"


def filter_pending_keys(state: ProgressState, keys: Iterable[str], resume: bool) -> list[str]:
    out: list[str] = []
    for k in keys:
        if resume and is_passed(state, k):
            continue
        out.append(k)
    return out


def upsert_case(state: ProgressState, case_key: str, payload: dict) -> None:
    prev = state.cases.get(case_key) or {}
    prev.update(payload)
    prev["key"] = case_key
    state.cases[case_key] = prev


def track_instance(state: ProgressState, instance_id: str) -> None:
    if instance_id and instance_id not in state.active_instance_ids:
        state.active_instance_ids.append(instance_id)


def untrack_instance(state: ProgressState, instance_id: str) -> None:
    state.active_instance_ids = [i for i in state.active_instance_ids if i != instance_id]
