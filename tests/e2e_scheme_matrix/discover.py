"""UI 枚举彩种×运行类型×玩法×子玩法。"""
from __future__ import annotations

from browser.platform_app import PlatformApp
from config import ADV_TRIGGER_MODES, ADV_TRIGGER_RUN_TYPE, EXCLUDED_RUN_TYPES, RUN_TYPE_LABELS
from interval import parse_draw_interval, target_periods_for_case
from models import CaseKey, CaseSpec


# 运行类型展示名 → id（与自创方案页一致）
_RUN_LABEL_TO_ID = {v: k for k, v in RUN_TYPE_LABELS.items()}


def discover_case_specs(
    app: PlatformApp,
    *,
    smoke: bool = False,
    draw_interval_by_label: dict[str, str] | None = None,
) -> list[CaseSpec]:
    """
    在自创方案页按 UI 可选组合枚举。
    smoke=True：只返回第一个可用组合（高级开某投某只取 always_pos）。
    """
    draw_interval_by_label = draw_interval_by_label or {}
    app.open_custom_scheme_new()

    app.open_picker("lottery")
    lottery_labels = app.list_picker_option_labels()
    app.close_picker()
    if not lottery_labels:
        raise RuntimeError("彩种列表为空")

    specs: list[CaseSpec] = []

    print(f"[discover] 彩种数={len(lottery_labels)}", flush=True)

    for li, lottery_label in enumerate(lottery_labels, 1):
        print(f"[discover] lottery {li}/{len(lottery_labels)} {lottery_label}", flush=True)
        try:
            app.open_custom_scheme_new()
            app.open_picker("lottery")
            app.confirm_picker_by_label(lottery_label)
        except Exception as e:
            print(f"[discover] skip lottery={lottery_label!r}: {e}", flush=True)
            try:
                app.close_picker()
            except Exception:
                pass
            continue

        try:
            app.open_picker("runType")
            run_labels = [
                x
                for x in app.list_picker_option_labels()
                if _RUN_LABEL_TO_ID.get(x, "") not in EXCLUDED_RUN_TYPES and x != "内置计画"
            ]
            app.close_picker()
        except Exception as e:
            print(f"[discover] skip runTypes lottery={lottery_label!r}: {e}", flush=True)
            try:
                app.close_picker()
            except Exception:
                pass
            continue

        interval_raw = draw_interval_by_label.get(lottery_label, "")
        # 若只有 code 映射，调用方应已按 displayName 建好 dict
        decision = parse_draw_interval(interval_raw) if interval_raw else None
        print(f"[discover]   runTypes={len(run_labels)} interval={interval_raw or '-'}", flush=True)

        for run_label in run_labels:
            run_id = _RUN_LABEL_TO_ID.get(run_label, run_label)
            try:
                app.open_picker("runType")
                app.confirm_picker_by_label(run_label)
                app.open_picker("playType")
                play_labels = app.list_picker_option_labels()
                app.close_picker()
            except Exception as e:
                print(
                    f"[discover] skip run={lottery_label}/{run_label}: {e}",
                    flush=True,
                )
                try:
                    app.close_picker()
                except Exception:
                    pass
                continue
            if not play_labels:
                continue

            for play_label in play_labels:
                try:
                    app.open_picker("playType")
                    # 选项可能随彩种/运行类型变化；以当前面板为准
                    current_plays = app.list_picker_option_labels()
                    if play_label not in current_plays:
                        app.close_picker()
                        print(
                            f"[discover] skip stale play={lottery_label}/{run_label}/{play_label}",
                            flush=True,
                        )
                        continue
                    app.confirm_picker_by_label(play_label)
                    app.open_picker("subPlay")
                    sub_labels = app.list_picker_option_labels()
                    app.close_picker()
                except Exception as e:
                    print(
                        f"[discover] skip play={lottery_label}/{run_label}/{play_label}: {e}",
                        flush=True,
                    )
                    try:
                        app.close_picker()
                    except Exception:
                        pass
                    continue
                if not sub_labels:
                    continue

                for sub_label in sub_labels:
                    if decision is None or decision.skip:
                        skip_reason = (
                            decision.skip_reason
                            if decision
                            else f"缺少 draw_interval: {lottery_label}"
                        )
                        global_target = 0
                        skip = True
                    else:
                        skip_reason = ""
                        global_target = decision.target_periods
                        skip = False

                    # 冒烟：跳过无法判定间隔的组合，继续找下一条可跑用例
                    if smoke and skip:
                        continue

                    modes = (
                        ADV_TRIGGER_MODES
                        if run_id == ADV_TRIGGER_RUN_TYPE
                        else (("-", "-"),)
                    )
                    if smoke and run_id == ADV_TRIGGER_RUN_TYPE:
                        modes = (ADV_TRIGGER_MODES[0],)

                    for mode_id, mode_label in modes:
                        target = (
                            0
                            if skip
                            else target_periods_for_case(run_id, global_target)
                        )
                        key = CaseKey(
                            lottery_code=lottery_label,
                            run_type_id=run_id,
                            play_type_id=play_label,
                            sub_play_id=sub_label,
                            trigger_mode=mode_id,
                        )
                        specs.append(
                            CaseSpec(
                                key=key,
                                lottery_label=lottery_label,
                                run_type_label=run_label,
                                play_type_label=play_label,
                                sub_play_label=sub_label,
                                trigger_mode_label=mode_label,
                                draw_interval=interval_raw,
                                target_periods=target,
                                skip_reason=skip_reason if skip else "",
                            )
                        )
                        if smoke:
                            return specs

    if smoke and not specs:
        raise RuntimeError(
            "冒烟未找到可用组合：所有彩种均缺少可解析的 draw_interval（请检查 PLATFORM_API_BASE /public/lotteries）"
        )
    return specs
