"""选号策略（按玩法画像）。"""
from __future__ import annotations

from dataclasses import dataclass, field
from math import comb

from browser.platform_app import PlatformApp
from play_profile import (
    SSC_POS_LABELS,
    PickMode,
    danshi_sample,
    infer_pick_mode,
    prefer_pool_clicks,
    prefer_pool_values,
    ren_pick_count,
)


@dataclass
class PickPlan:
    """描述在本平台与第三方应点选的同一组号码/选项。"""

    mode: PickMode = PickMode.POSITION
    inputs: list[str] = field(default_factory=list)
    button_labels: list[str] = field(default_factory=list)
    line_picks: list[list[str]] = field(default_factory=list)
    position_labels: list[str] = field(default_factory=list)
    danshi_text: str = ""
    notes: str = ""


def estimate_bet_count(
    *,
    mode: PickMode,
    sub_play_label: str,
    button_labels: list[str],
    line_picks: list[list[str]],
) -> int:
    """UI 无「注数」时的兜底估算（与常见 SSC 规则对齐）。"""
    sub = sub_play_label or ""
    labels = [x for x in button_labels if x]
    if "组六" in sub:
        n = len(set(labels))
        return comb(n, 3) if n >= 3 else 0
    if "组三" in sub:
        n = len(set(labels))
        return n * (n - 1) if n >= 2 else 0
    if mode == PickMode.POSITION and line_picks:
        prod = 1
        for row in line_picks:
            prod *= max(len(row), 1)
        return prod
    if mode == PickMode.RENXUAN and line_picks:
        filled = sum(1 for row in line_picks if row)
        need = ren_pick_count(sub)
        if filled >= need:
            return comb(filled, need) if filled >= need else 0
    if labels:
        return max(1, len(labels))
    return 0


def _read_bet_count(
    app: PlatformApp,
    *,
    mode: PickMode,
    sub_play_label: str,
    button_labels: list[str],
    line_picks: list[list[str]],
) -> int:
    try:
        n = app.read_platform_bet_count()
        if n > 0:
            return n
    except Exception:
        pass
    return estimate_bet_count(
        mode=mode,
        sub_play_label=sub_play_label,
        button_labels=button_labels,
        line_picks=line_picks,
    )


def apply_platform_pick(
    app: PlatformApp,
    *,
    play_type_label: str = "",
    sub_play_label: str = "",
    prefer_bet_count: int = 3,
    run_type_id: str = "",
) -> PickPlan:
    rt = (run_type_id or "").strip()
    if rt == "adv_fixed_rotate":
        app.open_jushu_dialog()
        plan = apply_platform_pick(
            app,
            play_type_label=play_type_label,
            sub_play_label=sub_play_label,
            prefer_bet_count=prefer_bet_count,
            run_type_id="",
        )
        app.confirm_jushu_dialog()
        plan.notes = f"adv_jushu {plan.notes}"
        return plan

    if rt == "hot_cold_warm":
        line_picks = app.pick_hcw_digits(per_pos=1)
        flat = [d for row in line_picks for d in row]
        n = _read_bet_count(
            app,
            mode=PickMode.POSITION,
            sub_play_label=sub_play_label,
            button_labels=flat,
            line_picks=line_picks,
        )
        if n <= 0:
            raise RuntimeError("冷热温选号后注数仍为 0")
        return PickPlan(
            mode=PickMode.POSITION,
            button_labels=flat,
            line_picks=line_picks,
            notes=f"hcw actual={n} picks={flat}",
        )

    if rt == "random_draw":
        line_picks = app.ensure_random_draw_ready()
        flat = [d for row in line_picks for d in row]
        n = _read_bet_count(
            app,
            mode=PickMode.POSITION,
            sub_play_label=sub_play_label,
            button_labels=flat,
            line_picks=line_picks,
        )
        if n <= 0:
            n = 1
            for row in line_picks:
                n *= max(len(row), 1)
        return PickPlan(
            mode=PickMode.POSITION,
            button_labels=flat,
            line_picks=line_picks,
            notes=f"random_draw actual={n} picks={flat}",
        )

    if rt == "adv_trigger_bet":
        # 模式与全部随机由 runner.set_adv_trigger_mode 处理；此处取首行正投作镜像样本
        page = app.page
        first_pos = page.locator(".scf-trig-grid").nth(1).locator("input, .el-select").first
        sample = "0"
        try:
            if first_pos.count():
                sample = (first_pos.input_value() or first_pos.inner_text() or "0").strip() or "0"
        except Exception:
            sample = "0"
        # 触发玩法多为单码/属性；用 POOL/ATTR 镜像一个正投值
        mode = infer_pick_mode(play_type_label, sub_play_label)
        if mode == PickMode.POSITION:
            mode = PickMode.POOL
        return PickPlan(
            mode=mode,
            button_labels=[sample],
            line_picks=[],
            notes=f"adv_trigger sample={sample}",
        )

    mode = infer_pick_mode(play_type_label, sub_play_label)
    prefer = max(1, prefer_bet_count)
    if mode == PickMode.POOL:
        prefer = prefer_pool_clicks(sub_play_label)

    if mode == PickMode.DANSHI:
        sample = danshi_sample(play_type_label, sub_play_label)
        app.fill_scheme_danshi(sample)
        n = _read_bet_count(
            app,
            mode=mode,
            sub_play_label=sub_play_label,
            button_labels=[p.strip() for p in sample.replace("，", ",").split(",") if p.strip()],
            line_picks=[],
        )
        if n <= 0:
            raise RuntimeError(f"单式填入后注数仍为 0 sample={sample!r}")
        return PickPlan(
            mode=mode,
            danshi_text=sample,
            button_labels=[p.strip() for p in sample.replace("，", ",").split(",") if p.strip()],
            notes=f"danshi actual={n}",
        )

    panel = app.detect_content_panel()
    # 直选复式/组合：勿被「仅有 chip、暂无 row」误判为号池
    if mode == PickMode.POSITION and panel == "pool":
        panel = app.wait_content_panel(prefer="position")
        if panel == "pool" and (
            "复式" in sub_play_label or "组合" in sub_play_label or "定位" in sub_play_label
        ):
            panel = "position"

    if panel == "danshi" and mode != PickMode.DANSHI:
        sample = danshi_sample(play_type_label, sub_play_label)
        app.fill_scheme_danshi(sample)
        n = _read_bet_count(
            app,
            mode=PickMode.DANSHI,
            sub_play_label=sub_play_label,
            button_labels=[p.strip() for p in sample.replace("，", ",").split(",") if p.strip()],
            line_picks=[],
        )
        if n <= 0:
            raise RuntimeError(f"探测为单式但注数仍为 0 sample={sample!r}")
        return PickPlan(
            mode=PickMode.DANSHI,
            danshi_text=sample,
            button_labels=[p.strip() for p in sample.replace("，", ",").split(",") if p.strip()],
            notes=f"danshi(detected) actual={n}",
        )

    if mode == PickMode.RENXUAN:
        n_pos = ren_pick_count(sub_play_label)
        pos_labels = list(SSC_POS_LABELS[:n_pos])
        digits = [str(i) for i in range(n_pos)]
        line_picks = app.pick_renxuan_positions(pos_labels, digits)
        flat = [d for row in line_picks for d in row]
        bet_n = _read_bet_count(
            app,
            mode=mode,
            sub_play_label=sub_play_label,
            button_labels=flat,
            line_picks=line_picks,
        )
        # 若校验仍按五星满位要求导致 0，则五位各点 1 码（注数=C(5,n)）
        if bet_n <= 0:
            pos_labels = list(SSC_POS_LABELS)
            digits = [str(i % 10) for i in range(5)]
            line_picks = app.pick_renxuan_positions(pos_labels, digits)
            flat = [d for row in line_picks for d in row]
            bet_n = _read_bet_count(
                app,
                mode=mode,
                sub_play_label=sub_play_label,
                button_labels=flat,
                line_picks=line_picks,
            )
        if bet_n <= 0:
            raise RuntimeError(f"任选选号后注数仍为 0 positions={pos_labels}")
        return PickPlan(
            mode=PickMode.RENXUAN,
            button_labels=flat,
            line_picks=line_picks,
            position_labels=pos_labels,
            notes=f"renxuan n={len(pos_labels)} actual={bet_n}",
        )

    if mode in (PickMode.POOL, PickMode.ATTR) or (
        panel == "pool" and mode not in (PickMode.POSITION, PickMode.RENXUAN)
    ):
        app.clear_scheme_chips()
        prefer_vals = prefer_pool_values(play_type_label, sub_play_label)
        if prefer_vals:
            # 和值/属性：只取所需个数，禁止回退到号池首位 0
            need = 1 if mode == PickMode.ATTR or prefer == 1 else prefer
            picked = app.pick_pool_chips_by_labels(prefer_vals[:need], allow_fallback_first=False)
            if not picked and mode == PickMode.ATTR:
                picked = app.pick_pool_chips_by_labels(prefer_vals, allow_fallback_first=True)
            if not picked:
                picked = app.pick_pool_mid_numeric(count=need)
        else:
            picked = app.pick_pool_chips(min_clicks=prefer)
        n = _read_bet_count(
            app,
            mode=PickMode.ATTR if mode == PickMode.ATTR else PickMode.POOL,
            sub_play_label=sub_play_label,
            button_labels=picked,
            line_picks=[],
        )
        if n <= 0:
            picked = app.pick_pool_mid_numeric(count=max(prefer, 1))
            n = _read_bet_count(
                app,
                mode=PickMode.POOL,
                sub_play_label=sub_play_label,
                button_labels=picked,
                line_picks=[],
            )
        if n <= 0:
            raise RuntimeError("号池选号后注数仍为 0")
        return PickPlan(
            mode=PickMode.ATTR if mode == PickMode.ATTR else PickMode.POOL,
            button_labels=picked,
            line_picks=[],
            notes=f"pool prefer≈{prefer} actual={n} picks={picked}",
        )

    # POSITION：先等分行面板
    if mode == PickMode.POSITION:
        app.wait_content_panel(prefer="position")

    # POSITION
    picked, line_picks = app.pick_scheme_content_chips(min_clicks=prefer)
    n = _read_bet_count(
        app,
        mode=PickMode.POSITION,
        sub_play_label=sub_play_label,
        button_labels=picked,
        line_picks=line_picks,
    )
    if n <= 0:
        picked = app.pick_pool_chips(min_clicks=prefer)
        n = _read_bet_count(
            app,
            mode=PickMode.POOL,
            sub_play_label=sub_play_label,
            button_labels=picked,
            line_picks=[],
        )
        if n > 0:
            return PickPlan(
                mode=PickMode.POOL,
                button_labels=picked,
                notes=f"fallback-pool actual={n}",
            )
        raise RuntimeError("选号后注数仍为 0，请检查玩法面板是否为按位选号")

    guard = 0
    while n < prefer and guard < 30:
        if not app.click_one_more_chip():
            break
        n = _read_bet_count(
            app,
            mode=PickMode.POSITION,
            sub_play_label=sub_play_label,
            button_labels=picked,
            line_picks=line_picks,
        )
        guard += 1

    if n <= 0:
        raise RuntimeError("选号调整后注数仍为 0")

    return PickPlan(
        mode=PickMode.POSITION,
        button_labels=picked,
        line_picks=line_picks,
        notes=f"position prefer≈{prefer} actual={n}",
    )
