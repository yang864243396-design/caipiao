"""本平台彩种名 ↔ v6hs1 菜单名归一化匹配。"""
from __future__ import annotations

import re

_CN_DIGITS = {
    "零": "0",
    "〇": "0",
    "一": "1",
    "二": "2",
    "两": "2",
    "三": "3",
    "四": "4",
    "五": "5",
    "六": "6",
    "七": "7",
    "八": "8",
    "九": "9",
    "十": "10",
}


def normalize_lottery_name(name: str) -> str:
    s = (name or "").strip().lower()
    s = re.sub(r"\s+", "", s)
    for a, b in _CN_DIGITS.items():
        s = s.replace(a, b)
    s = s.replace("分钟", "分")
    return s


def match_lottery_label(platform_label: str, candidates: list[str]) -> str | None:
    """在候选菜单名中找最佳匹配；失败返回 None。

    优先完全相等；其次要求归一化后互相包含，且按公共前缀长度打分，
    避免「波场1分彩」误匹配到仅含「1分彩」的短标签或其它品牌。
    """
    target = normalize_lottery_name(platform_label)
    if not target:
        return None
    # 过滤噪音
    cleaned = []
    for c in candidates:
        n = normalize_lottery_name(c)
        if not n or n in {"彩票", "官方", "首页", "娱乐", "推广", "福利"}:
            continue
        if "\n" in (c or ""):
            # 取较短行
            parts = [normalize_lottery_name(x) for x in c.splitlines() if x.strip()]
            n = min(parts, key=len) if parts else n
        cleaned.append((c, n))

    exact = [c for c, n in cleaned if n == target]
    if exact:
        return exact[0]

    scored: list[tuple[int, str]] = []
    for c, n in cleaned:
        if target == n:
            scored.append((1000, c))
            continue
        if target in n or n in target:
            # 公共前缀长度（波场/哈希/币安）
            prefix = 0
            for a, b in zip(target, n):
                if a != b:
                    break
                prefix += 1
            # 过短包含（如仅「1分彩」）降权
            score = prefix * 10 + min(len(n), len(target))
            if min(len(n), len(target)) < 4:
                score -= 50
            scored.append((score, c))
    if not scored:
        return None
    scored.sort(key=lambda x: -x[0])
    return scored[0][1]
