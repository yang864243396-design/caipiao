"""常量与路径（对齐 README v1.1）。"""
from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parent
REPORTS_DIR = ROOT / "reports"
PROGRESS_PATH = ROOT / "progress.json"
ENV_PATH = ROOT / ".env"
V6_STORAGE_STATE = ROOT / "v6_storage_state.json"

MAX_PARALLEL = 1  # 建案+双浏览器镜像选号串行，避免 picker/页签超时
SCHEME_FUNDS = 100_000
STOP_LOSS = 100_000
TAKE_PROFIT = 100_000
MULT_COEFF = 1
BET_UNIT_LABEL = "1元"
SHARE_STATUS = "public"
RUN_MODE_FORMAL = True  # 正式运行

# 目标期数（快慢彩统一 10 期，缩短全矩阵与冒烟耗时）
TARGET_PERIODS_FAST = 10  # draw_interval ≤ 5m
TARGET_PERIODS_SLOW = 10  # draw_interval > 5m
FAST_INTERVAL_MAX_SECONDS = 5 * 60

# 等待系数
PERIOD_WAIT_INTERVALS = 2  # 单期最多等 2×间隔
CASE_WAIT_FACTOR = 1.5  # 整案最多 目标期数×间隔×1.5

AMOUNT_PRECISION = 0.01

EXCLUDED_RUN_TYPES = frozenset({"builtin_plan"})
ADV_TRIGGER_RUN_TYPE = "adv_trigger_bet"
ADV_TRIGGER_MODES = (
    ("always_pos", "一直正投"),
    ("always_neg", "一直反投"),
    ("alt_pos_first", "前正后反"),
    ("alt_neg_first", "前反后正"),
)

RUN_TYPE_LABELS = {
    "fixed_rotate": "定码轮换",
    "adv_fixed_rotate": "高级定码轮换",
    "adv_trigger_bet": "高级开某投某",
    "hot_cold_warm": "冷热温出号",
    "random_draw": "随机出号",
    "builtin_plan": "内置计画",
    "fixed_number": "固定号码",
}

EXCEL_HEADERS = [
    "彩种",
    "运行类型",
    "玩法",
    "子玩法",
    "触发模式",
    "方案最终名称",
    "创建结果",
    "注数对比",
    "开启结果",
    "目标期数",
    "实际期数",
    "终止原因",
    "记录对比结果",
    "失败详情",
]
