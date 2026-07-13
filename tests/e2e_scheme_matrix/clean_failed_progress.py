"""清理 progress：仅保留已完整通过的用例，便于修复后 --resume 重跑失败项。"""
from __future__ import annotations

import json
from pathlib import Path

from config import PROGRESS_PATH
from progress_store import is_passed, load_progress, save_progress


def main() -> int:
    state = load_progress()
    before = len(state.cases)
    kept = {k: v for k, v in state.cases.items() if is_passed(state, k)}
    state.cases = kept
    state.active_instance_ids = []
    save_progress(state)
    print(f"progress cleaned: {before} -> {len(kept)} (kept passed only)")
    print(json.dumps(list(kept.keys()), ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
