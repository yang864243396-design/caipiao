"""玩法选号画像：本端与第三方共用同一套模式推断。"""
from __future__ import annotations

from enum import Enum


class PickMode(str, Enum):
    """选号交互模式。"""

    POSITION = "position"  # 按位复式 / 定位胆 / 组合
    RENXUAN = "renxuan"  # 任选直选复式：只填 n 个位
    POOL = "pool"  # 单号池：和值、跨度、包胆、组选、不定位等
    DANSHI = "danshi"  # 单式 / 混合组选（textarea）
    ATTR = "attr"  # 文字属性：龙虎、大小单双、特殊号等


SSC_POS_LABELS = ("万", "千", "百", "十", "个")

# 第三方页签：本端名 → 候选名（含自身）
PLAY_TAB_ALIASES: dict[str, tuple[str, ...]] = {
    "一星": ("一星", "定位胆", "五星定位胆"),
    "前三码": ("前三码", "前三"),
    "中三码": ("中三码", "中三"),
    "后三码": ("后三码", "后三"),
    "前二码": ("前二码", "前二"),
    "后二码": ("后二码", "后二"),
    "前中后三": ("前中后三", "前中后"),
    "前后三": ("前后三",),
    "前后二": ("前后二",),
    "前后四": ("前后四",),
    "四星": ("四星",),
    "五星": ("五星",),
    "不定位": ("不定位",),
    "任选": ("任选",),
    "龙虎": ("龙虎",),
    "大小单双": ("大小单双", "趣味"),
}


def play_tab_candidates(play_type_label: str) -> list[str]:
    label = (play_type_label or "").strip()
    if not label:
        return []
    out: list[str] = []
    for c in PLAY_TAB_ALIASES.get(label, (label,)):
        if c and c not in out:
            out.append(c)
    if label.endswith("码"):
        short = label[:-1]
        if short and short not in out:
            out.append(short)
    elif label and f"{label}码" not in out:
        out.append(f"{label}码")
    if label not in out:
        out.insert(0, label)
    return out


_PREFIXES = (
    "前中后三",
    "前后三",
    "前后二",
    "前后四",
    "前三",
    "中三",
    "后三",
    "前二",
    "后二",
    "五星",
    "四星",
    "任选四",
    "任选三",
    "任选二",
    "任四",
    "任三",
    "任二",
)


def sub_play_candidates(play_type_label: str, sub_play_label: str) -> list[str]:
    """第三方子玩法文案候选（常省略大类前缀）。"""
    sub = (sub_play_label or "").strip()
    play = (play_type_label or "").strip()
    if not sub:
        return []
    out: list[str] = [sub]
    for p in (play, play.replace("码", ""), *_PREFIXES):
        p = (p or "").strip()
        if p and sub.startswith(p) and len(sub) > len(p):
            rest = sub[len(p) :].lstrip("-_ ")
            if rest and rest not in out:
                out.append(rest)
    for a, b in (
        ("直选复式", "复式"),
        ("直选单式", "单式"),
        ("直选跨度", "跨度"),
        ("直选和值", "和值"),
        ("直选组合", "组合"),
        ("组选和值", "组选和值"),
        ("组选复式", "组选"),
        ("组选包胆", "包胆"),
        ("和值尾数", "和值尾数"),
        ("混合组选", "混合组选"),
        ("混合组选", "混合"),
        ("组三", "组三"),
        ("组六", "组六"),
        ("特殊号", "特殊号"),
    ):
        if a in sub and b not in out:
            out.append(b)
        if sub.endswith(a) and a not in out:
            out.append(a)
        # 完整前缀名也放入候选（V6 常见「前三组六」「前三直选跨度」）
        if a in sub or sub.endswith(a):
            for pref in (play, play.replace("码", "")):
                pref = (pref or "").strip()
                if pref and not sub.startswith(pref):
                    full = pref + a
                    if full not in out:
                        out.append(full)
    # 短歧义词放最后，避免点到「和值单双」「混合」等无关项
    ambiguous = {"和值", "混合", "复式", "单式", "组选", "包胆", "跨度", "组合"}
    primary = [x for x in out if x not in ambiguous]
    secondary = [x for x in out if x in ambiguous]
    return primary + secondary


def ren_pick_count(sub_play_label: str) -> int:
    s = sub_play_label or ""
    if any(x in s for x in ("任选四", "任四", "ren4")):
        return 4
    if any(x in s for x in ("任选三", "任三", "ren3")):
        return 3
    if any(x in s for x in ("任选二", "任二", "ren2")):
        return 2
    return 2


def infer_pick_mode(play_type_label: str, sub_play_label: str) -> PickMode:
    play = (play_type_label or "").strip()
    sub = (sub_play_label or "").strip()
    text = f"{play} {sub}"

    if "单式" in sub or "混合组选" in sub:
        return PickMode.DANSHI

    if play == "龙虎" or "龙虎" in sub:
        return PickMode.ATTR
    if "大小单双" in text:
        return PickMode.ATTR
    if "特殊号" in sub:
        return PickMode.ATTR

    pool_keys = (
        "和值",
        "跨度",
        "包胆",
        "和值尾数",
        "组三",
        "组六",
        "组选",
        "不定位",
        "趣味",
    )
    if any(k in sub for k in pool_keys) or play == "不定位":
        if "直选复式" in sub or (sub.endswith("复式") and "组选" not in sub and "直选" in sub):
            return PickMode.POSITION
        return PickMode.POOL

    if play == "任选":
        if "单式" in sub or "混合" in sub:
            return PickMode.DANSHI
        if "直选复式" in sub or ("直选" in sub and "复式" in sub):
            return PickMode.RENXUAN
        return PickMode.POOL

    if "复式" in sub or "组合" in sub or "定位" in sub or play == "一星":
        return PickMode.POSITION

    return PickMode.POSITION


def danshi_sample(play_type_label: str, sub_play_label: str) -> str:
    """生成可出注的单式样例。"""
    text = f"{play_type_label} {sub_play_label}"
    if any(x in text for x in ("前二", "后二", "任二", "任选二", "二星")):
        return "01,23"
    if any(x in text for x in ("四星", "任四", "任选四", "前后四")):
        return "0123,4567"
    if "五星" in text:
        return "01234,56789"
    if "混合" in text:
        return "012,345,678"
    return "012,345"


def prefer_pool_clicks(sub_play_label: str) -> int:
    sub = sub_play_label or ""
    if "组六" in sub:
        return 3
    if "组三" in sub:
        return 2
    if "二码不定位" in sub or "组选2" in sub:
        return 2
    if "包胆" in sub:
        return 1
    if "跨度" in sub or "和值" in sub or "尾数" in sub:
        return 1
    return 3


def prefer_pool_values(play_type_label: str, sub_play_label: str) -> list[str] | None:
    """优先点选的号池/属性值（避免和值点 0、属性乱点）。"""
    play = play_type_label or ""
    sub = sub_play_label or ""
    if play == "龙虎" or "龙虎" in sub:
        return ["龙", "虎"]
    if "特殊号" in sub:
        return ["对子", "顺子", "豹子"]
    if "大小单双" in f"{play} {sub}":
        return ["大", "小", "单", "双"]
    if "和值尾数" in sub or ( "尾数" in sub and "和值" in sub):
        return ["3", "5", "7"]
    if "跨度" in sub:
        return ["3", "5", "2"]
    if "和值" in sub:
        return ["12", "13", "10", "9", "8", "11", "14"]
    if "包胆" in sub:
        return ["5", "3", "1"]
    return None
