#!/usr/bin/env python3
"""P0 seed CSV generator — run from repo: python backend/docs/seeds/generate_p0_seeds.py"""
from __future__ import annotations
import csv
import json
from pathlib import Path

OUT = Path(__file__).resolve().parent

BALL = {
    "ssc_std": 5,
    "lhc_std": 7,
    "syxw_std": 5,
    "pk10_std": 10,
    "k3_std": 3,
    "pc28_std": 3,
}

CATALOG = [
    (1, "ffc", "波场1分彩", "tron_ffc_1m", "ssc_std", "1m"),
    (2, "ffc", "波场3分彩", "tron_ffc_3m", "ssc_std", "3m"),
    (3, "ffc", "波场5分彩", "tron_ffc_5m", "ssc_std", "5m"),
    (4, "ffc", "哈希1分彩", "hash_ffc_1m", "ssc_std", "1m"),
    (5, "ffc", "哈希3分彩", "hash_ffc_3m", "ssc_std", "3m"),
    (6, "ffc", "哈希5分彩", "hash_ffc_5m", "ssc_std", "5m"),
    (7, "ffc", "以太坊1分彩", "eth_ffc_1m", "ssc_std", "1m"),
    (8, "ffc", "以太坊3分彩", "eth_ffc_3m", "ssc_std", "3m"),
    (9, "ffc", "以太坊5分彩", "eth_ffc_5m", "ssc_std", "5m"),
    (10, "ffc", "币安1分彩", "bnb_ffc_1m", "ssc_std", "1m"),
    (11, "ffc", "币安3分彩", "bnb_ffc_3m", "ssc_std", "3m"),
    (12, "ffc", "币安5分彩", "bnb_ffc_5m", "ssc_std", "5m"),
    (13, "ffc", "新以太坊分分彩", "eth_ffc_new", "ssc_std", ""),
    (14, "jisu", "波场极速彩", "tron_jisu", "ssc_std", "jisu"),
    (15, "jisu", "哈希极速彩", "hash_jisu", "ssc_std", "jisu"),
    (16, "jisu", "以太坊极速彩", "eth_jisu", "ssc_std", "jisu"),
    (17, "lhc", "波场1分六合彩", "tron_lhc_1m", "lhc_std", "1m"),
    (18, "lhc", "波场3分六合彩", "tron_lhc_3m", "lhc_std", "3m"),
    (19, "lhc", "波场5分六合彩", "tron_lhc_5m", "lhc_std", "5m"),
    (21, "syxw", "波场11选5", "tron_syxw", "syxw_std", ""),
    (22, "syxw", "波场3分11选5", "tron_syxw_3m", "syxw_std", "3m"),
    (23, "syxw", "波场5分11选5", "tron_syxw_5m", "syxw_std", "5m"),
    (24, "syxw", "以太坊11选5", "eth_syxw", "syxw_std", ""),
    (25, "syxw", "以太坊3分11选5", "eth_syxw_3m", "syxw_std", "3m"),
    (26, "syxw", "以太坊5分11选5", "eth_syxw_5m", "syxw_std", "5m"),
    (27, "syxw", "币安11选5", "bnb_syxw", "syxw_std", ""),
    (28, "syxw", "币安3分11选5", "bnb_syxw_3m", "syxw_std", "3m"),
    (29, "syxw", "币安5分11选5", "bnb_syxw_5m", "syxw_std", "5m"),
    (30, "pk10", "以太极速赛车", "eth_pk10_jisu", "pk10_std", "jisu"),
    (31, "pk10", "以太5分赛车", "eth_pk10_5m", "pk10_std", "5m"),
    (32, "pk10", "币安极速飞艇", "bnb_pk10_jisu", "pk10_std", "jisu"),
    (33, "pk10", "币安5分飞艇", "bnb_pk10_5m", "pk10_std", "5m"),
    (34, "pk10", "波场极速赛车", "tron_pk10_jisu", "pk10_std", "jisu"),
    (36, "k3", "以太坊快三", "eth_k3", "k3_std", ""),
    (37, "k3", "以太坊3分快三", "eth_k3_3m", "k3_std", "3m"),
    (38, "k3", "以太坊5分快三", "eth_k3_5m", "k3_std", "5m"),
    (39, "k3", "波场极速快三", "tron_k3_jisu", "k3_std", "jisu"),
    (40, "k3", "波场1分快三", "tron_k3_1m", "k3_std", "1m"),
    (41, "k3", "波场3分快三", "tron_k3_3m", "k3_std", "3m"),
    (42, "k3", "波场5分快三", "tron_k3_5m", "k3_std", "5m"),
    (43, "k3", "币安1分快三", "bnb_k3_1m", "k3_std", "1m"),
    (44, "k3", "币安3分快三", "bnb_k3_3m", "k3_std", "3m"),
    (45, "k3", "币安5分快三", "bnb_k3_5m", "k3_std", "5m"),
]

TEMPLATES = [
    ("ssc_std", "时时彩类标准玩法"),
    ("lhc_std", "六合彩标准玩法"),
    ("syxw_std", "11选5标准玩法"),
    ("pk10_std", "PK10标准玩法"),
    ("k3_std", "快三标准玩法"),
    ("pc28_std", "PC28标准玩法"),
]

LH_PAIRS = [
    ("wanqian", "万千"),
    ("wanbai", "万百"),
    ("wanshi", "万十"),
    ("wange", "万个"),
    ("qianbai", "千百"),
    ("qianshi", "千十"),
    ("qiange", "千个"),
    ("baishi", "百十"),
    ("baige", "百个"),
    ("shige", "十个"),
]


def seg(**kw) -> str:
    return json.dumps(kw, ensure_ascii=False)


def write_csv(name: str, headers: list[str], rows: list[list]) -> None:
    path = OUT / name
    with path.open("w", newline="", encoding="utf-8-sig") as f:
        w = csv.writer(f)
        w.writerow(headers)
        w.writerows(rows)
    print(f"  {name}: {len(rows)} rows")


def sql_str(value: str) -> str:
    return "'" + value.replace("'", "''") + "'"


def sql_nullable(value: str) -> str:
    v = (value or "").strip()
    return "NULL" if v == "" else sql_str(v)


def sql_bool(value: str) -> str:
    return "true" if str(value).strip().lower() == "true" else "false"


SSC_THREE_MODES = [
    ("zhixuan_fs", "直选复式", "fushi"),
    ("zhixuan_ds", "直选单式", "danshi"),
    ("zhixuan_hz", "直选和值", "hezhi"),
    ("zhixuan_kd", "直选跨度", "kuadu"),
    ("zuhe", "组合", "zuhe"),
    ("zu3", "组三", "zu3"),
    ("zu6", "组六", "zu6"),
    ("zuxuan_hz", "组选和值", "hezhi"),
    ("zuxuan_bd", "组选包胆", "baodan"),
    ("hunhe_zx", "混合组选", "hunhe"),
    ("hz_weishu", "和值尾数", "weishu"),
    ("teshu", "特殊号", "teshu"),
]

SYXW_SEGMENT = seg(numberPoolMin=1, numberPoolMax=11, pickCount=5)


def outbound_play_code(template: str, type_id: str, sub_id: str) -> str:
    """§11-1 默认；同模板内 sub_id 可重复时带上 type_id。"""
    return f"{template}:{type_id}:{sub_id}"


def ssc_three(prefix: str, label_prefix: str) -> list[tuple[str, str, str]]:
    return [
        (f"{prefix}_{sid}", f"{label_prefix}{lbl}", mode)
        for sid, lbl, mode in SSC_THREE_MODES
    ]


def ssc_three_short(prefix: str) -> list[tuple[str, str, str]]:
    """前中后三 / 前后三：大类内菜单短名（§11-11）。"""
    return [(f"{prefix}_{sid}", lbl, mode) for sid, lbl, mode in SSC_THREE_MODES]


def ssc_two(prefix: str, label_prefix: str) -> list[tuple[str, str, str]]:
    subs = [
        ("zhixuan_fs", "直选复式", "fushi"),
        ("zhixuan_ds", "直选单式", "danshi"),
        ("zhixuan_hz", "直选和值", "hezhi"),
        ("zhixuan_kd", "直选跨度", "kuadu"),
        ("zuxuan_fs", "组选复式", "fushi"),
        ("zuxuan_ds", "组选单式", "danshi"),
        ("zuxuan_hz", "组选和值", "hezhi"),
        ("zuxuan_bd", "组选包胆", "baodan"),
    ]
    return [(f"{prefix}_{s}", f"{label_prefix}{l}", m) for s, l, m in subs]


def build_ssc() -> tuple[list, list]:
    types: list[tuple[str, str, int]] = []
    subs: list[tuple[str, str, str, str, int, str]] = []
    sort_t = 0

    def add_type(tid: str, label: str):
        nonlocal sort_t
        sort_t += 1
        types.append((tid, label, sort_t))

    def add_subs(tid: str, items: list[tuple[str, str, str]], start: int = 1):
        for i, (sid, label, bet_mode) in enumerate(items, start):
            subs.append(("ssc_std", tid, sid, label, start + i - 1, bet_mode))

    add_type("dingwei", "定位胆")
    dw = [
        ("dingwei_wan", "一星定位胆 · 万位", "dingwei"),
        ("dingwei_qian", "一星定位胆 · 千位", "dingwei"),
        ("dingwei_bai", "一星定位胆 · 百位", "dingwei"),
        ("dingwei_shi", "一星定位胆 · 十位", "dingwei"),
        ("dingwei_ge", "一星定位胆 · 个位", "dingwei"),
    ]
    add_subs("dingwei", dw)

    for tid, lp in [("qian3", "前三"), ("zhong3", "中三"), ("hou3", "后三")]:
        add_type(tid, lp)
        add_subs(tid, [(s, l, m) for s, l, m in ssc_three(tid, lp)])

    for tid, lp in [("qian2", "前二"), ("hou2", "后二")]:
        add_type(tid, lp)
        add_subs(tid, [(s, l, m) for s, l, m in ssc_two(tid, lp)])

    add_type("longhu", "龙虎")
    lh: list[tuple[str, str, str]] = []
    for code, cn in LH_PAIRS:
        lh.append((f"lh_{code}_he", f"{cn}龙虎和", "longhuhe"))
        lh.append((f"lh_{code}_dou", f"{cn}龙虎斗", "longhu"))
    add_subs("longhu", lh)

    add_type("renxuan", "任选")
    rx: list[tuple[str, str, str]] = []
    rx += [
        ("ren2_zhixuan_fs", "任二直选复式", "fushi"),
        ("ren2_zhixuan_ds", "任二直选单式", "danshi"),
        ("ren2_zhixuan_hz", "任二直选和值", "hezhi"),
        ("ren2_zuxuan_fs", "任二组选复式", "fushi"),
        ("ren2_zuxuan_ds", "任二组选单式", "danshi"),
        ("ren2_zuxuan_hz", "任二组选和值", "hezhi"),
        ("ren3_zhixuan_fs", "任三直选复式", "fushi"),
        ("ren3_zhixuan_ds", "任三直选单式", "danshi"),
        ("ren3_zhixuan_hz", "任三直选和值", "hezhi"),
        ("ren3_zu3_fs", "任三组三复式", "fushi"),
        ("ren3_zu3_ds", "任三组三单式", "danshi"),
        ("ren3_zu6_fs", "任三组六复式", "fushi"),
        ("ren3_zu6_ds", "任三组六单式", "danshi"),
        ("ren3_hunhe_zx", "任三混合组选", "hunhe"),
        ("ren3_zuxuan_hz", "任三组选和值", "hezhi"),
        ("ren4_zhixuan_fs", "任选四直选复式", "fushi"),
        ("ren4_zhixuan_ds", "任选四直选单式", "danshi"),
        ("ren4_zu24", "任选四组选24", "zu24"),
        ("ren4_zu12", "任选四组选12", "zu12"),
        ("ren4_zu6", "任选四组选6", "zu6"),
    ]
    add_subs("renxuan", rx)

    add_type("qianzhonghou3", "前中后三")
    add_subs("qianzhonghou3", ssc_three_short("qzh3"))

    add_type("qianhou3", "前后三")
    add_subs("qianhou3", ssc_three_short("qh3"))

    add_type("budingwei", "不定位")
    bw = [
        ("qian3_1ma", "前三一码不定位", "budingwei"),
        ("qian3_2ma", "前三二码不定位", "budingwei"),
        ("zhong3_1ma", "中三一码不定位", "budingwei"),
        ("zhong3_2ma", "中三二码不定位", "budingwei"),
        ("hou3_1ma", "后三一码不定位", "budingwei"),
        ("hou3_2ma", "后三二码不定位", "budingwei"),
        ("qian4_1ma", "前四一码不定位", "budingwei"),
        ("qian4_2ma", "前四二码不定位", "budingwei"),
        ("hou4_1ma", "后四一码不定位", "budingwei"),
        ("hou4_2ma", "后四二码不定位", "budingwei"),
        ("wuxing_1ma", "五星一码不定位", "budingwei"),
        ("wuxing_2ma", "五星二码不定位", "budingwei"),
        ("wuxing_3ma", "五星三码不定位", "budingwei"),
    ]
    add_subs("budingwei", bw)

    add_type("combo24", "前后二/前后四")
    c24: list[tuple[str, str, str]] = []
    for s, l, m in ssc_two("qh2", "前后二"):
        c24.append((s, l, m))
    for sid, lbl, mode in [
        ("qh4_zhixuan_fs", "前后四直选复式", "fushi"),
        ("qh4_zhixuan_ds", "前后四直选单式", "danshi"),
        ("qh4_zhixuan_zh", "前后四直选组合", "zuhe"),
        ("qh4_zu24", "前后四组选24", "zu24"),
        ("qh4_zu12", "前后四组选12", "zu12"),
        ("qh4_zu6", "前后四组选6", "zu6"),
        ("qh4_zu4", "前后四组选4", "zu4"),
    ]:
        c24.append((sid, lbl, mode))
    add_subs("combo24", c24)

    add_type("sixing", "四星")
    sx = [
        ("sixing_zhixuan_fs", "四星直选复式", "fushi"),
        ("sixing_zhixuan_ds", "四星直选单式", "danshi"),
        ("sixing_zhixuan_zh", "四星直选组合", "zuhe"),
        ("sixing_zu24", "四星组选24", "zu24"),
        ("sixing_zu12", "四星组选12", "zu12"),
        ("sixing_zu6", "四星组选6", "zu6"),
        ("sixing_zu4", "四星组选4", "zu4"),
    ]
    add_subs("sixing", sx)

    add_type("wuxing", "五星")
    wx = [
        ("wuxing_zhixuan_fs", "五星直选复式", "fushi"),
        ("wuxing_zhixuan_ds", "五星直选单式", "danshi"),
        ("wuxing_zhixuan_zh", "五星直选组合", "zuhe"),
        ("wuxing_zu120", "五星组选120", "zu120"),
        ("wuxing_zu60", "五星组选60", "zu60"),
        ("wuxing_zu30", "五星组选30", "zu30"),
        ("wuxing_zu20", "五星组选20", "zu20"),
        ("wuxing_zu10", "五星组选10", "zu10"),
        ("wuxing_zu5", "五星组选5", "zu5"),
        ("wuxing_yifan", "五星一帆风顺", "teshu"),
        ("wuxing_haoshi", "五星好事成双", "teshu"),
        ("wuxing_sanxing", "五星三星报喜", "teshu"),
        ("wuxing_siji", "五星四季发财", "teshu"),
    ]
    add_subs("wuxing", wx)

    add_type("dxds", "大小单双")
    dx = [
        ("hou2_dxds", "后二大小单双", "dxds"),
        ("hou3_dxds", "后三大小单双", "dxds"),
        ("qian2_dxds", "前二大小单双", "dxds"),
        ("qian3_dxds", "前三大小单双", "dxds"),
        ("wuxing_hz_ds", "五星和值单双", "danshuang"),
        ("wuxing_hz_dx", "五星和值大小", "daxiao"),
    ]
    add_subs("dxds", dx)

    assert len(subs) == 175, f"ssc sub_plays expected 175, got {len(subs)}"
    assert len(types) == 15, f"ssc play_types expected 15, got {len(types)}"
    return types, subs


def build_syxw() -> tuple[list, list]:
    types = [
        ("dingwei", "定位胆", 1),
        ("qian3", "前三", 2),
        ("qian2", "前二", 3),
        ("renxuan_fs", "复式任选", 4),
        ("renxuan_ds", "单式任选", 5),
    ]
    subs: list[tuple[str, str, str, str, int, str]] = []
    for sid, lbl in [
        ("dingwei_wan", "万位"),
        ("dingwei_qian", "千位"),
        ("dingwei_bai", "百位"),
        ("dingwei_shi", "十位"),
        ("dingwei_ge", "个位"),
    ]:
        subs.append(("syxw_std", "dingwei", sid, f"定位胆 · {lbl}", len(subs) + 1, "dingwei"))
    for sid, lbl, m in [
        ("qian3_zhixuan_fs", "前三直选复式", "fushi"),
        ("qian3_zhixuan_ds", "前三直选单式", "danshi"),
        ("qian3_zuxuan_fs", "前三组选复式", "fushi"),
        ("qian3_zuxuan_ds", "前三组选单式", "danshi"),
        ("qian3_budingwei", "前三不定位", "budingwei"),
    ]:
        subs.append(("syxw_std", "qian3", sid, lbl, len(subs) + 1, m))
    for sid, lbl, m in [
        ("qian2_zhixuan_fs", "前二直选复式", "fushi"),
        ("qian2_zhixuan_ds", "前二直选单式", "danshi"),
        ("qian2_zuxuan_fs", "前二组选复式", "fushi"),
        ("qian2_zuxuan_ds", "前二组选单式", "danshi"),
    ]:
        subs.append(("syxw_std", "qian2", sid, lbl, len(subs) + 1, m))
    rx_labels = [
        ("1z1", "一中一"),
        ("2z2", "二中二"),
        ("3z3", "三中三"),
        ("4z4", "四中四"),
        ("5z5", "五中五"),
        ("6z5", "六中五"),
        ("7z5", "七中五"),
        ("8z5", "八中五"),
    ]
    for code, cn in rx_labels:
        subs.append(
            ("syxw_std", "renxuan_fs", f"rx_{code}", f"复式任选{cn}", len(subs) + 1, "fushi")
        )
        subs.append(
            ("syxw_std", "renxuan_ds", f"rx_{code}_ds", f"单式任选{cn}", len(subs) + 1, "danshi")
        )
    assert len(subs) == 30
    return types, subs


def build_pk10() -> tuple[list, list]:
    types = [
        ("dingwei", "定位胆", 1),
        ("longhu", "龙虎斗", 2),
        ("qian1", "前一", 3),
        ("qian2", "前二", 4),
        ("qian3", "前三", 5),
        ("qian4", "前四", 6),
        ("qian5", "前五", 7),
        ("daxiao", "大小", 8),
        ("danshuang", "单双", 9),
        ("hezhi", "和值", 10),
        ("dxds_combo", "大小单双", 11),
    ]
    subs: list[tuple[str, str, str, str, int, str]] = []
    for sid, lbl in [
        ("dingwei_wan", "万位"),
        ("dingwei_qian", "千位"),
        ("dingwei_bai", "百位"),
        ("dingwei_shi", "十位"),
        ("dingwei_ge", "个位"),
    ]:
        subs.append(("pk10_std", "dingwei", sid, f"定位胆 · {lbl}", len(subs) + 1, "dingwei"))
    for sid, lbl in [
        ("lh_1v10", "1-Vs-10"),
        ("lh_2v9", "2-Vs-9"),
        ("lh_3v8", "3-Vs-8"),
        ("lh_4v7", "4-Vs-7"),
        ("lh_5v6", "5-Vs-6"),
    ]:
        subs.append(("pk10_std", "longhu", sid, lbl, len(subs) + 1, "longhu"))
    subs.append(("pk10_std", "qian1", "qian1_zhixuan_fs", "前一直选复式", len(subs) + 1, "fushi"))
    for tid, n in [("qian2", "二"), ("qian3", "三"), ("qian4", "四"), ("qian5", "五")]:
        subs.append(("pk10_std", tid, f"{tid}_zhixuan_fs", f"前{n}直选复式", len(subs) + 1, "fushi"))
        subs.append(("pk10_std", tid, f"{tid}_zhixuan_ds", f"前{n}直选单式", len(subs) + 1, "danshi"))
    rank = ["gj", "yj", "jj", "ds4", "ds5"]
    rank_cn = ["冠军", "亚军", "季军", "第四名", "第五名"]
    for s, cn in zip(rank, rank_cn):
        subs.append(("pk10_std", "daxiao", f"dx_{s}", f"{cn}大小", len(subs) + 1, "daxiao"))
        subs.append(("pk10_std", "danshuang", f"ds_{s}", f"{cn}单双", len(subs) + 1, "danshuang"))
    for sid, lbl in [
        ("hz_guanya", "冠亚和值"),
        ("hz_shouwei", "首尾和值"),
        ("hz_qian3", "前三和值"),
        ("hz_hou3", "后三和值"),
    ]:
        subs.append(("pk10_std", "hezhi", sid, lbl, len(subs) + 1, "hezhi"))
    for sid, lbl in [
        ("dxds_guanya", "冠亚和值大小单双"),
        ("dxds_qian3", "前三大小单双"),
        ("dxds_hou3", "后三大小单双"),
    ]:
        subs.append(("pk10_std", "dxds_combo", sid, lbl, len(subs) + 1, "dxds"))
    assert len(subs) == 36
    return types, subs


def build_k3() -> tuple[list, list]:
    types = [
        ("hezhi", "和值", 1),
        ("tonghao", "同号", 2),
        ("butonghao", "不同号", 3),
        ("lianhao_qita", "连号与其它", 4),
    ]
    items = [
        ("hezhi", "k3_hezhi", "快三和值", "hezhi"),
        ("tonghao", "ertong_dan", "二同号单选", "danshi"),
        ("tonghao", "ertong_fu", "二同号复选", "fushi"),
        ("tonghao", "santong", "三同号", "tonghao"),
        ("butonghao", "2butong", "二不同号", "butong"),
        ("butonghao", "biaozhun", "标准选号", "fushi"),
        ("butonghao", "shoudong", "手动输入", "shoudong"),
        ("lianhao_qita", "sanlian", "三连号", "lianhao"),
        ("lianhao_qita", "dantiao", "单挑一骰", "dantiao"),
    ]
    subs = [
        ("k3_std", tid, sid, lbl, i + 1, m) for i, (tid, sid, lbl, m) in enumerate(items)
    ]
    assert len(subs) == 9
    return types, subs


LHC_DP6 = [
    ("fushi", "复式", "fushi"),
    ("tuotou", "拖头", "tuotou"),
    ("sx_dp", "生肖对碰", "sx_dp"),
    ("ws_dp", "尾数对碰", "ws_dp"),
    ("sw_dp", "生尾对碰", "sw_dp"),
    ("renyi_dp", "任意对碰", "renyi_dp"),
]


def build_lhc() -> tuple[list, list]:
    types: list[tuple[str, str, int]] = []
    subs: list[tuple[str, str, str, str, int, str]] = []
    sort_t = 0

    def add_type(tid: str, label: str):
        nonlocal sort_t
        sort_t += 1
        types.append((tid, label, sort_t))

    def add_subs(tid: str, items: list[tuple[str, str, str]]):
        for sid, label, bet_mode in items:
            subs.append(("lhc_std", tid, sid, label, len(subs) + 1, bet_mode))

    add_type("tema", "特码")
    tema: list[tuple[str, str, str]] = [("tema_a", "特码A", "tema")]
    for i in range(1, 7):
        tema.append((f"zheng{i}_te", f"正{i}特", "zhengte"))
    add_subs("tema", tema)

    for tid, tlabel in [
        ("erquanzhong", "二全中"),
        ("erzhongte", "二中特"),
        ("techuan", "特串"),
    ]:
        add_type(tid, tlabel)
        add_subs(tid, [(sid, lbl, m) for sid, lbl, m in LHC_DP6])

    for tid, tlabel in [("sanzhonger", "三中二"), ("sanquanzhong", "三全中")]:
        add_type(tid, tlabel)
        add_subs(
            tid,
            [(sid, lbl, m) for sid, lbl, m in LHC_DP6[:2]],
        )

    add_type("shengxiao", "生肖")
    sx: list[tuple[str, str, str]] = [
        ("texiao", "特肖", "texiao"),
        ("zongxiao", "总肖", "zongxiao"),
    ]
    for n in range(2, 7):
        sx.append((f"{n}xiao", f"{n}肖", "xiao"))
    sx += [
        ("1xiao", "一肖", "xiao"),
        ("1xiao_bz", "一肖不中", "xiao_bz"),
    ]
    for n in range(2, 6):
        sx.append((f"{n}xiao_z", f"{n}肖中", "xiao_z"))
    for n in range(2, 6):
        sx.append((f"{n}xiao_bz", f"{n}肖不中", "xiao_bz"))
    add_subs("shengxiao", sx)

    add_type("weishu", "尾数")
    ws: list[tuple[str, str, str]] = [
        ("weishu", "尾数", "weishu"),
        ("weishu_bz", "尾数不中", "weishu_bz"),
    ]
    for n in range(2, 5):
        ws.append((f"{n}wei_z", f"{n}尾中", "wei_z"))
    for n in range(2, 5):
        ws.append((f"{n}wei_bz", f"{n}尾不中", "wei_bz"))
    add_subs("weishu", ws)

    add_type("buzhong_xuanyi", "不中/选一")
    bx: list[tuple[str, str, str]] = []
    for n in range(5, 13):
        bx.append((f"{n}bz", f"{n}不中", "buzhong"))
    bx.append(("15bz", "15不中", "buzhong"))
    for n in range(5, 11):
        bx.append((f"{n}x1", f"{n}选中一", "xuanyi"))
    add_subs("buzhong_xuanyi", bx)

    add_type("guoguan", "过关")
    add_subs("guoguan", [("guoguan", "过关", "guoguan")])

    add_type("tematouwei", "特码头尾")
    add_subs("tematouwei", [("tematouwei", "特码头尾", "tematouwei")])

    add_type("wuxingjiaye", "五行家野")
    add_subs(
        "wuxingjiaye",
        [("wuxing", "五行", "wuxing"), ("jiaye", "家野", "jiaye")],
    )

    add_type("bose", "波色")
    add_subs(
        "bose",
        [
            ("bose", "波色", "bose"),
            ("banbo", "半波", "banbo"),
            ("banbanbo", "半半波", "banbanbo"),
        ],
    )

    add_type("qima", "七码")
    add_subs("qima", [("qima", "七码", "qima")])

    add_type("renzhong", "任中")
    add_subs(
        "renzhong",
        [(f"{n}l_rz", f"{n}粒任中", "renzhong") for n in range(1, 6)],
    )

    assert len(subs) == 82, f"lhc sub_plays expected 82, got {len(subs)}"
    assert len(types) == 15, f"lhc play_types expected 15, got {len(types)}"
    return types, subs


def build_pc28() -> tuple[list, list]:
    types = [("pc28_20", "2.0", 1), ("pc28_28", "2.8", 2)]
    subs = []
    for tid, line in [("pc28_20", "2.0"), ("pc28_28", "2.8")]:
        for sid, lbl, m in [
            ("hezhi", "和值", "hezhi"),
            ("dxds", "大小单双", "dxds"),
            ("teshu", "特殊号", "teshu"),
            ("longhubao", "龙虎豹", "longhubao"),
        ]:
            subs.append(("pc28_std", tid, sid, f"{line} · {lbl}", len(subs) + 1, m))
    assert len(subs) == 8
    return types, subs


def main() -> None:
    print("Generating P0 seeds ->", OUT)
    cat_rows = [
        [
            code,
            name,
            cat,
            tpl,
            BALL[tpl],
            interval or "",
            sort,
            "true",
            code,
        ]
        for sort, cat, name, code, tpl, interval in CATALOG
    ]
    write_csv(
        "lottery_catalog.csv",
        [
            "code",
            "display_name",
            "category_code",
            "play_template",
            "ball_count",
            "draw_interval",
            "sort_order",
            "on_sale",
            "outbound_lottery_code",
        ],
        cat_rows,
    )
    write_csv(
        "play_templates.csv",
        ["code", "label", "version"],
        [[c, l, 1] for c, l in TEMPLATES],
    )

    all_types: list[list] = []
    all_subs: list[list] = []
    builders = [
        ("ssc_std", build_ssc),
        ("lhc_std", build_lhc),
        ("syxw_std", build_syxw),
        ("pk10_std", build_pk10),
        ("k3_std", build_k3),
        ("pc28_std", build_pc28),
    ]
    ssc_mapping: list[list] = []
    for tpl, fn in builders:
        types, subs = fn()
        for tid, label, so in types:
            all_types.append([tpl, tid, label, so, "", "true"])
        for row in subs:
            tpl, tid, sid, label, so, bet_mode = row
            seg_rule = SYXW_SEGMENT if tpl == "syxw_std" else "{}"
            outbound = outbound_play_code(tpl, tid, sid)
            all_subs.append(
                [tpl, tid, sid, label, so, bet_mode, seg_rule, outbound, "true"]
            )
            if tpl == "ssc_std":
                ssc_mapping.append(
                    [tpl, tid, sid, label, outbound, bet_mode, so]
                )

    write_csv(
        "play_types.csv",
        ["template_code", "type_id", "label", "sort_order", "panel_type", "enabled"],
        all_types,
    )
    write_csv(
        "sub_plays.csv",
        [
            "template_code",
            "type_id",
            "sub_id",
            "label",
            "sort_order",
            "bet_mode",
            "segment_rule",
            "outbound_play_code",
            "enabled",
        ],
        all_subs,
    )
    write_csv(
        "ssc_175_play_mapping.csv",
        [
            "template_code",
            "type_id",
            "sub_id",
            "label",
            "outbound_play_code",
            "bet_mode",
            "sort_order",
        ],
        ssc_mapping,
    )

    platform_mapping = [
        [tpl, tid, sid, label, outbound, bet_mode, so]
        for tpl, tid, sid, label, so, bet_mode, seg_rule, outbound, enabled in all_subs
    ]
    write_csv(
        "platform_340_play_mapping.csv",
        [
            "template_code",
            "type_id",
            "sub_id",
            "label",
            "outbound_play_code",
            "bet_mode",
            "sort_order",
        ],
        platform_mapping,
    )

    write_migration_seed_sql(cat_rows, all_types, all_subs)

    assert len(all_types) == 52, f"play_types expected 52, got {len(all_types)}"
    assert len(all_subs) == 340, f"sub_plays expected 340, got {len(all_subs)}"
    assert len(ssc_mapping) == 175, f"ssc mapping expected 175, got {len(ssc_mapping)}"
    print(f"  Total play_types: {len(all_types)}, sub_plays: {len(all_subs)}")
    print(f"  ssc_175_play_mapping.csv: {len(ssc_mapping)} rows")
    print(f"  platform_340_play_mapping.csv: {len(platform_mapping)} rows")


def write_migration_seed_sql(
    cat_rows: list[list],
    all_types: list[list],
    all_subs: list[list],
) -> None:
    """生成 P1 goose seed：backend/migrations/00070_lottery_play_catalog_seed.sql"""
    mig_dir = OUT.parent.parent / "migrations"
    path = mig_dir / "00070_lottery_play_catalog_seed.sql"
    lines = [
        "-- +goose Up",
        "-- +goose StatementBegin",
        "-- P1 seed：47 彩种 + 52 play_types + 340 sub_plays（由 generate_p0_seeds.py 生成）",
        "",
        "INSERT INTO lottery_catalog (",
        "    code, display_name, category_code, play_template, ball_count,",
        "    draw_interval, sort_order, on_sale, sale_status, outbound_lottery_code",
        ") VALUES",
    ]
    cat_values = []
    for row in cat_rows:
        code, name, cat, tpl, ball, interval, sort, on_sale, outbound = row
        cat_values.append(
            f"    ({sql_str(code)}, {sql_str(name)}, {sql_str(cat)}, {sql_str(tpl)}, "
            f"{ball}, {sql_nullable(interval)}, {sort}, {sql_bool(on_sale)}, "
            f"'on_sale', {sql_str(outbound)})"
        )
    lines.append(",\n".join(cat_values))
    lines.append("ON CONFLICT (code) DO NOTHING;")
    lines.append("")
    lines.append(
        "INSERT INTO play_types (template_code, type_id, label, sort_order, panel_type, enabled) VALUES"
    )
    type_values = []
    for tpl, tid, label, so, panel, enabled in all_types:
        panel_sql = sql_nullable(panel)
        type_values.append(
            f"    ({sql_str(tpl)}, {sql_str(tid)}, {sql_str(label)}, {so}, {panel_sql}, {sql_bool(enabled)})"
        )
    lines.append(",\n".join(type_values))
    lines.append("ON CONFLICT (template_code, type_id) DO NOTHING;")
    lines.append("")
    lines.append(
        "INSERT INTO sub_plays (template_code, type_id, sub_id, label, sort_order, bet_mode, segment_rule, outbound_play_code, enabled) VALUES"
    )
    sub_values = []
    for tpl, tid, sid, label, so, bet_mode, seg_rule, outbound, enabled in all_subs:
        sub_values.append(
            f"    ({sql_str(tpl)}, {sql_str(tid)}, {sql_str(sid)}, {sql_str(label)}, {so}, "
            f"{sql_nullable(bet_mode)}, {sql_str(seg_rule)}::jsonb, {sql_str(outbound)}, {sql_bool(enabled)})"
        )
    lines.append(",\n".join(sub_values))
    lines.append("ON CONFLICT (template_code, type_id, sub_id) DO NOTHING;")
    lines.append("-- +goose StatementEnd")
    lines.append("")
    lines.append("-- +goose Down")
    lines.append("-- +goose StatementBegin")
    lines.append("DELETE FROM sub_plays;")
    lines.append("DELETE FROM play_types;")
    lines.append(
        "DELETE FROM lottery_catalog WHERE code NOT IN ("
        "'tencent_ffc', 'tencent_10', 'qiqu_tencent', 'us_ffc', "
        "'cq_ssc', 'xj_ssc', 'tj_ssc', 'fc_3d', 'pl3'"
        ");"
    )
    lines.append("-- +goose StatementEnd")
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"  {path.relative_to(OUT.parent.parent.parent)}: seed SQL written")


if __name__ == "__main__":
    main()
