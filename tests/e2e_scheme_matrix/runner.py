"""主循环：建案 → 注数对比 → 开启 → 跑期 → 对账 → 暂停（分期3）。"""
from __future__ import annotations

import signal
import threading
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

from playwright.sync_api import sync_playwright

from api_client import PlatformApiClient
from bet_pick import apply_platform_pick, estimate_bet_count
from browser.platform_app import PlatformApp
from browser.v6_app import V6App
from compare import compare_bet_counts, compare_period_records
from config import MAX_PARALLEL, V6_STORAGE_STATE
from credentials import Credentials, load_credentials
from discover import discover_case_specs
from interval import parse_draw_interval
from models import CaseResult, CaseSpec, CaseStatus, ProgressState
from progress_store import (
    filter_pending_keys,
    load_progress,
    save_progress,
    track_instance,
    untrack_instance,
    upsert_case,
)
from report_excel import append_result, new_report_path
from run_wait import wait_for_periods
from v6_records import fetch_v6_bet_records


class Runner:
    def __init__(
        self,
        creds: Credentials,
        *,
        smoke: bool = False,
        resume: bool = False,
        headed: bool = True,
    ) -> None:
        self.creds = creds
        self.smoke = smoke
        self.resume = resume
        self.headed = headed
        self.progress: ProgressState = load_progress()
        self._stop = threading.Event()
        self._lock = threading.Lock()
        self._bearer = ""
        if not self.progress.report_path:
            self.progress.report_path = str(new_report_path())
            save_progress(self.progress)

    def request_stop(self) -> None:
        self._stop.set()

    def pause_all_active(self) -> None:
        """中断时暂停全部已跟踪实例。"""
        ids = list(self.progress.active_instance_ids)
        if not ids:
            return
        print(f"[runner] 暂停 {len(ids)} 个实例…")
        api = PlatformApiClient(self.creds.platform_api_base)
        try:
            if self._bearer:
                api.set_bearer(self._bearer)
            for iid in ids:
                try:
                    api.stop_instance(iid)
                    print(f"[runner] 已暂停 {iid}")
                except Exception as e:
                    print(f"[runner] 暂停失败 {iid}: {e}")
                untrack_instance(self.progress, iid)
            save_progress(self.progress)
        finally:
            api.close()

    def run(self) -> int:
        print(f"[runner] smoke={self.smoke} resume={self.resume} headed={self.headed}")
        print(f"[runner] report={self.progress.report_path}")
        print(f"[runner] api={self.creds.platform_api_base} parallel≤{MAX_PARALLEL}")

        intervals = self._load_intervals()
        if not intervals:
            print("[runner] 无法获取 draw_interval，中止。请确认后端 /public/lotteries 可用。")
            return 2
        specs = self._discover(intervals)
        keys = [s.key.as_str() for s in specs]
        pending_keys = set(filter_pending_keys(self.progress, keys, self.resume))
        pending = [s for s in specs if s.key.as_str() in pending_keys]
        print(f"[runner] 待执行={len(pending)} / 总组合={len(specs)}")

        if not pending:
            return 0

        failed = 0
        # 并行：每条用例独立 Playwright 上下文
        with ThreadPoolExecutor(max_workers=MAX_PARALLEL) as ex:
            futs = {ex.submit(self._run_one_case, spec, intervals): spec for spec in pending}
            for fut in as_completed(futs):
                if self._stop.is_set():
                    break
                spec = futs[fut]
                try:
                    ok = fut.result()
                except Exception as e:
                    print(f"[fail] 未捕获 {spec.key.as_str()}: {e}")
                    ok = False
                if not ok:
                    failed += 1

        if self._stop.is_set():
            self.pause_all_active()
        return 1 if failed else 0

    def _discover(self, intervals: dict[str, str]) -> list[CaseSpec]:
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=not self.headed)
            try:
                ctx, platform = PlatformApp.launch(browser, self.creds.platform_url)
                try:
                    platform.login(self.creds.platform_user, self.creds.platform_pass)
                    self._bearer = platform.read_access_token() or self._bearer
                    specs = discover_case_specs(
                        platform, smoke=self.smoke, draw_interval_by_label=intervals
                    )
                    print(f"[discover] 组合数={len(specs)}")
                    return specs
                finally:
                    ctx.close()
            finally:
                browser.close()

    def _load_intervals(self) -> dict[str, str]:
        last_err: Exception | None = None
        for attempt in range(1, 4):
            try:
                with PlatformApiClient(self.creds.platform_api_base) as api:
                    data = api.lottery_interval_by_display_name()
                    if data:
                        print(f"[runner] draw_interval 彩种数={len(data)} (attempt={attempt})")
                        return data
                    print(f"[warn] draw_interval 为空 (attempt={attempt})")
            except Exception as e:
                last_err = e
                print(f"[warn] 读取 draw_interval 失败 attempt={attempt}: {e}")
            time.sleep(1.5 * attempt)
        if last_err:
            print(f"[warn] 最终仍失败: {last_err}")
        return {}

    def _run_one_case(self, spec: CaseSpec, intervals: dict[str, str]) -> bool:
        if self._stop.is_set():
            return False

        key = spec.key.as_str()
        result = CaseResult(
            key=spec.key,
            lottery_label=spec.lottery_label,
            run_type_label=spec.run_type_label,
            play_type_label=spec.play_type_label,
            sub_play_label=spec.sub_play_label,
            trigger_mode_label=spec.trigger_mode_label,
            target_periods=spec.target_periods,
        )

        if spec.skip_reason:
            result.status = CaseStatus.SKIPPED
            result.stop_reason = "skipped"
            result.failure_detail = spec.skip_reason
            self._commit(result)
            print(f"[skip] {key}: {spec.skip_reason}")
            return True

        interval_raw = spec.draw_interval or intervals.get(spec.lottery_label, "")
        decision = parse_draw_interval(interval_raw)
        interval_sec = decision.seconds or 60

        with sync_playwright() as p:
            browser = p.chromium.launch(headless=not self.headed)
            p_ctx = v_ctx = None
            api = PlatformApiClient(self.creds.platform_api_base)
            try:
                p_ctx, platform = PlatformApp.launch(browser, self.creds.platform_url)
                v6_state = V6_STORAGE_STATE if V6_STORAGE_STATE.is_file() else None
                v_ctx, v6 = V6App.launch(
                    browser, self.creds.v6_url, storage_state=v6_state
                )

                print(f"[case] start {key}", flush=True)
                platform.login(self.creds.platform_user, self.creds.platform_pass)
                print("[case] platform login ok", flush=True)
                token = platform.read_access_token()
                if token:
                    with self._lock:
                        self._bearer = token
                    api.set_bearer(token)
                platform.ensure_guaji_auth(self.creds.platform_api_base)
                v6.login(self.creds.v6_user, self.creds.v6_pass)
                print("[case] v6 login ok", flush=True)

                # —— 建案 ——
                name = platform.build_scheme_name(
                    spec.lottery_label,
                    spec.run_type_label,
                    spec.play_type_label,
                    spec.sub_play_label,
                    spec.trigger_mode_label,
                )
                result.scheme_name = name

                try:
                    platform.open_custom_scheme_new()
                except RuntimeError as e:
                    em = str(e)
                    if "SESSION_EXPIRED" in em:
                        print("[case] session expired, re-login…", flush=True)
                        platform.login(self.creds.platform_user, self.creds.platform_pass)
                        platform.ensure_guaji_auth(self.creds.platform_api_base)
                        platform.open_custom_scheme_new()
                    elif "GUAJI_AUTH_REQUIRED" in em:
                        print("[case] guaji auth required, reauth…", flush=True)
                        platform.ensure_guaji_auth(self.creds.platform_api_base)
                        platform.open_custom_scheme_new()
                    else:
                        raise
                platform.fill_scheme_name(name)
                platform.open_and_confirm_picker("lottery", spec.lottery_label)
                platform.open_and_confirm_picker("runType", spec.run_type_label)
                platform.open_and_confirm_picker("playType", spec.play_type_label)
                platform.open_and_confirm_picker("subPlay", spec.sub_play_label)
                platform.click_next()

                platform.fill_basic_config()
                name_input = platform.page.locator("#scf-name")
                if name_input.count():
                    name_input.fill(name)
                platform.set_simple_bet_multiplier()
                if spec.key.run_type_id == "adv_trigger_bet":
                    platform.set_adv_trigger_mode(spec.trigger_mode_label)

                plan = apply_platform_pick(
                    platform,
                    play_type_label=spec.play_type_label,
                    sub_play_label=spec.sub_play_label,
                    prefer_bet_count=3,
                    run_type_id=spec.key.run_type_id,
                )
                try:
                    platform_n = platform.read_platform_bet_count()
                except Exception:
                    platform_n = estimate_bet_count(
                        mode=plan.mode,
                        sub_play_label=spec.sub_play_label,
                        button_labels=plan.button_labels,
                        line_picks=plan.line_picks,
                    )
                if platform_n <= 0:
                    result.status = CaseStatus.BET_COUNT_MISMATCH
                    result.create_ok = "ok"
                    result.bet_count_ok = f"invalid bet count (must >0) platform={platform_n}"
                    result.failure_detail = result.bet_count_ok
                    self._commit(result)
                    print(f"[fail] 注数 {key}: {result.bet_count_ok}")
                    return False

                v6.open_lottery_by_platform_label(spec.lottery_label)
                try:
                    v6.mirror_picks(
                        plan.button_labels,
                        line_picks=plan.line_picks,
                        play_type_label=spec.play_type_label,
                        sub_play_label=spec.sub_play_label,
                        mode=plan.mode,
                        danshi_text=plan.danshi_text,
                        position_labels=plan.position_labels,
                    )
                except RuntimeError as e:
                    if "玩法页签" in str(e) or "页签为空" in str(e) or "无玩法页签" in str(e):
                        print("[case] v6 tabs lost, reopen lottery…", flush=True)
                        v6.ensure_lottery_ready(spec.lottery_label)
                        v6.mirror_picks(
                            plan.button_labels,
                            line_picks=plan.line_picks,
                            play_type_label=spec.play_type_label,
                            sub_play_label=spec.sub_play_label,
                            mode=plan.mode,
                            danshi_text=plan.danshi_text,
                            position_labels=plan.position_labels,
                        )
                    else:
                        raise
                v6_n = v6.read_preview_bet_count()
                ok_count, count_msg = compare_bet_counts(platform_n, v6_n)
                result.bet_count_ok = count_msg
                if not ok_count:
                    result.status = CaseStatus.BET_COUNT_MISMATCH
                    result.create_ok = "ok"
                    result.failure_detail = count_msg
                    self._commit(result)
                    print(f"[fail] 注数 {key}: {count_msg}")
                    return False

                cloud_data = platform.save_to_cloud()
                result.create_ok = "ok"
                instance = cloud_data.get("instance") or {}
                definition = cloud_data.get("definition") or {}
                instance_id = str(instance.get("id") or "")
                definition_id = str(definition.get("id") or "")
                result.instance_id = instance_id
                result.definition_id = definition_id

                if not instance_id:
                    # 回退：按名称查云端列表
                    row = api.find_instance_by_name(name)
                    if row:
                        instance_id = str(row.get("id") or "")
                        result.instance_id = instance_id
                if not instance_id:
                    result.status = CaseStatus.CREATE_FAILED
                    result.failure_detail = "添加至云端后未获得 instanceId"
                    self._commit(result)
                    return False

                with self._lock:
                    track_instance(self.progress, instance_id)
                    save_progress(self.progress)

                # —— 开启 ——
                try:
                    api.start_instance(instance_id)
                    result.start_ok = "ok"
                except Exception as e:
                    try:
                        platform.start_scheme_by_name_ui(name)
                        result.start_ok = "ok(ui)"
                    except Exception as e2:
                        result.status = CaseStatus.START_FAILED
                        result.start_ok = "fail"
                        result.failure_detail = f"API:{e}; UI:{e2}"
                        self._commit(result)
                        with self._lock:
                            untrack_instance(self.progress, instance_id)
                            save_progress(self.progress)
                        print(f"[fail] 开启 {key}: {result.failure_detail}")
                        return False

                print(f"[run] 已开启 {name} id={instance_id} target={spec.target_periods}")

                # —— 跑期 ——
                # bet-records 路径的 schemeId 一般为实例 id
                wait = wait_for_periods(
                    api,
                    instance_id=instance_id,
                    scheme_id=instance_id,
                    target_periods=spec.target_periods,
                    interval_seconds=interval_sec,
                    should_abort=self._stop.is_set,
                )
                result.actual_periods = wait.actual_periods
                result.stop_reason = wait.stop_reason
                if wait.stop_reason == "early_stop":
                    result.status = CaseStatus.EARLY_STOP
                    result.failure_detail = wait.status_reason or wait.stop_reason
                elif wait.stop_reason in ("timeout", "aborted"):
                    result.status = CaseStatus.COMPARE_FAILED
                    result.failure_detail = wait.status_reason or wait.stop_reason

                platform_recs = wait.records or api.fetch_scheme_bet_records(
                    instance_id, mode="real", days=7
                )
                if not platform_recs and definition_id:
                    try:
                        platform_recs = api.fetch_scheme_bet_records(
                            definition_id, mode="real", days=7
                        )
                        if platform_recs:
                            print(
                                f"[case] bet-records via definition_id n={len(platform_recs)}",
                                flush=True,
                            )
                    except Exception:
                        pass
                for r in platform_recs:
                    r.lottery_label = spec.lottery_label

                # —— 第三方记录 ——
                try:
                    want_periods = {r.period for r in platform_recs if r.period}
                    v6_recs = fetch_v6_bet_records(
                        v6,
                        lottery_hint=spec.lottery_label,
                        scheme_hint=spec.sub_play_label,
                        limit=max(500, spec.target_periods * 20),
                        want_periods=want_periods or None,
                    )
                except Exception as e:
                    v6_recs = []
                    result.failure_detail = (result.failure_detail + f"; v6记录:{e}").strip("; ")

                ok_cmp, issues = compare_period_records(platform_recs, v6_recs)
                if ok_cmp and platform_recs:
                    result.record_compare = "passed"
                    if result.status not in (CaseStatus.EARLY_STOP,):
                        result.status = CaseStatus.PASSED
                    elif result.status == CaseStatus.EARLY_STOP:
                        # 提前终止但仍对比通过
                        result.record_compare = "passed"
                else:
                    result.record_compare = "failed"
                    detail = "; ".join(
                        f"{i.period}/{i.field}:平台={i.platform} 第三方={i.third_party}"
                        for i in issues[:20]
                    )
                    if not platform_recs:
                        detail = "本端无投注记录; " + detail
                    if not v6_recs:
                        detail = "第三方无解析记录; " + detail
                    result.failure_detail = (result.failure_detail + "; " + detail).strip("; ")
                    if result.status not in (
                        CaseStatus.EARLY_STOP,
                        CaseStatus.START_FAILED,
                    ):
                        result.status = CaseStatus.COMPARE_FAILED

                # —— 暂停 ——
                try:
                    api.stop_instance(instance_id)
                except Exception as e:
                    result.failure_detail = (result.failure_detail + f"; 暂停失败:{e}").strip("; ")

                with self._lock:
                    untrack_instance(self.progress, instance_id)
                    save_progress(self.progress)

                self._commit(result)
                print(
                    f"[{'ok' if result.record_compare == 'passed' else 'fail'}] "
                    f"{key} periods={result.actual_periods}/{result.target_periods} "
                    f"stop={result.stop_reason} compare={result.record_compare}"
                )
                return result.record_compare == "passed" or result.status == CaseStatus.EARLY_STOP

            except Exception as e:
                result.status = CaseStatus.CREATE_FAILED
                result.create_ok = result.create_ok or "fail"
                result.failure_detail = str(e)
                self._commit(result)
                print(f"[fail] 用例 {key}: {e}")
                return False
            finally:
                api.close()
                if p_ctx:
                    p_ctx.close()
                if v_ctx:
                    v_ctx.close()
                browser.close()

    def _commit(self, result: CaseResult) -> None:
        with self._lock:
            upsert_case(self.progress, result.key.as_str(), result.to_progress_dict())
            save_progress(self.progress)
            append_result(Path(self.progress.report_path), result)


def install_signal_handlers(runner: Runner) -> None:
    def _handler(signum: int, _frame: object) -> None:
        print(f"\n[runner] 收到信号 {signum}，暂停全部方案并退出…")
        runner.request_stop()
        try:
            runner.pause_all_active()
        finally:
            save_progress(runner.progress)

    signal.signal(signal.SIGINT, _handler)
    if hasattr(signal, "SIGTERM"):
        signal.signal(signal.SIGTERM, _handler)


def build_runner_from_env(**kwargs: object) -> Runner:
    return Runner(load_credentials(), **kwargs)  # type: ignore[arg-type]
