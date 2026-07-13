"""play_profile 单元测试。"""
from __future__ import annotations

from play_profile import (
    PickMode,
    danshi_sample,
    infer_pick_mode,
    play_tab_candidates,
    prefer_pool_values,
    ren_pick_count,
    sub_play_candidates,
)


def test_infer_modes() -> None:
    assert infer_pick_mode("前三码", "前三直选复式") == PickMode.POSITION
    assert infer_pick_mode("前三码", "前三直选单式") == PickMode.DANSHI
    assert infer_pick_mode("前三码", "前三直选和值") == PickMode.POOL
    assert infer_pick_mode("前三码", "前三混合组选") == PickMode.DANSHI
    assert infer_pick_mode("前三码", "前三特殊号") == PickMode.ATTR
    assert infer_pick_mode("龙虎", "万千") == PickMode.ATTR
    assert infer_pick_mode("任选", "任二直选复式") == PickMode.RENXUAN
    assert infer_pick_mode("任选", "任选四组选6") == PickMode.POOL
    assert ren_pick_count("任二直选复式") == 2
    assert ren_pick_count("任三直选复式") == 3


def test_aliases_and_prefs() -> None:
    assert "前三" in play_tab_candidates("前三码")
    assert "直选复式" in sub_play_candidates("前三码", "前三直选复式")
    assert prefer_pool_values("前三码", "前三直选和值")[0] == "12"
    assert "龙" in prefer_pool_values("龙虎", "万千")
    assert "012,345,678" in danshi_sample("前三码", "前三混合组选")


if __name__ == "__main__":
    test_infer_modes()
    test_aliases_and_prefs()
    print("ok")
