"""compare_period_records 同期限消歧与派奖净额对齐。"""
from __future__ import annotations

from compare import compare_period_records
from models import BetRecord
from v6_records import _record_from_api


def test_pick_best_match_prefers_content() -> None:
    platform = [
        BetRecord(lottery_label="波场", period="101", content="12", amount=73.0, multiplier=1.0),
    ]
    v6 = [
        BetRecord(lottery_label="波场", period="101", content="012,345", amount=4.0, multiplier=2.0),
        BetRecord(lottery_label="波场", period="101", content="12", amount=73.0, multiplier=1.0),
    ]
    ok, issues = compare_period_records(platform, v6)
    assert ok, issues


def test_baodan_payout_uses_net() -> None:
    item = {
        "periods": "101",
        "game_name": "波场1分彩",
        "bet_amount": 54.0,
        "net_amount": 107.67,
        "payout_amount": 161.666,
        "settled": True,
        "note": "中奖",
        "bet_content": {"bet_content": {"bet_content": "5", "bets_nums": 54, "multiple": 1}},
    }
    rec = _record_from_api(item)
    assert rec is not None
    assert abs((rec.payout or 0) - 107.67) < 0.01

    platform = [
        BetRecord(
            lottery_label="波场",
            period="101",
            content="5",
            amount=54.0,
            payout=107.67,
            multiplier=1.0,
            win_status="中",
        ),
    ]
    ok, issues = compare_period_records(platform, [rec])
    assert ok, issues


def test_multiline_content_norm() -> None:
    platform = [
        BetRecord(lottery_label="波场", period="101", content="0,1,3\n0\n0", amount=9.0),
    ]
    v6 = [
        BetRecord(lottery_label="波场", period="101", content="013,0,0", amount=9.0),
    ]
    ok, issues = compare_period_records(platform, v6)
    assert ok, issues
