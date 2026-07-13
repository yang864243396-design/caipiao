"""用户端 Playwright 页面对象。"""
from __future__ import annotations

import re
from datetime import datetime

from playwright.sync_api import Browser, BrowserContext, Page, expect

from config import (
    BET_UNIT_LABEL,
    MULT_COEFF,
    SCHEME_FUNDS,
    STOP_LOSS,
    TAKE_PROFIT,
)


class PlatformApp:
    def __init__(self, page: Page, base_url: str) -> None:
        self.page = page
        self.base_url = base_url.rstrip("/")

    @classmethod
    def launch(cls, browser: Browser, base_url: str) -> tuple[BrowserContext, "PlatformApp"]:
        context = browser.new_context(
            viewport={"width": 1280, "height": 900},
            locale="zh-CN",
        )
        page = context.new_page()
        return context, cls(page, base_url)

    def login(self, username: str, password: str) -> None:
        self.page.goto(f"{self.base_url}/login", wait_until="domcontentloaded")
        account = self.page.locator('input[autocomplete="username"]').first
        pwd = self.page.locator('input[autocomplete="current-password"]').first
        account.wait_for(state="visible", timeout=30_000)
        account.fill(username)
        pwd.fill(password)
        self.page.get_by_role("button", name="登录").click()
        self.page.wait_for_url(lambda url: "/login" not in url, timeout=60_000)
        expect(self.page).not_to_have_url(re.compile(r"/login"))

    def goto_lobby(self) -> None:
        self.page.goto(f"{self.base_url}/", wait_until="domcontentloaded")

    def open_custom_scheme_new(self) -> None:
        """打开自创方案新建页；处理登录/授权跳转与 SPA 未挂载。"""
        last_err: Exception | None = None
        for attempt in range(5):
            try:
                self.page.goto(
                    f"{self.base_url}/play/custom-scheme/new",
                    wait_until="domcontentloaded",
                    timeout=60_000,
                )
                self.page.wait_for_timeout(500)
                url = self.page.url or ""
                if "/login" in url:
                    raise RuntimeError("SESSION_EXPIRED:打开自创方案页被重定向到登录")
                if "/member/auth" in url:
                    raise RuntimeError(
                        "GUAJI_AUTH_REQUIRED:打开自创方案页被重定向到授权页"
                    )
                # 关闭可能挡住的 Element Plus 弹层
                for _ in range(3):
                    try:
                        mask = self.page.locator(".el-overlay, .el-dialog__wrapper").first
                        if mask.count() and mask.is_visible(timeout=200):
                            self.page.keyboard.press("Escape")
                            self.page.wait_for_timeout(200)
                        else:
                            break
                    except Exception:
                        break
                loc = self.page.locator("#csn-scheme-name")
                loc.wait_for(state="visible", timeout=25_000)
                return
            except Exception as e:
                last_err = e
                msg = str(e)
                if "SESSION_EXPIRED" in msg or "GUAJI_AUTH_REQUIRED" in msg:
                    raise
                try:
                    self.page.reload(wait_until="domcontentloaded", timeout=30_000)
                except Exception:
                    pass
                self.page.wait_for_timeout(600 + attempt * 400)
        raise RuntimeError(f"打开自创方案页失败: {last_err}") from last_err

    def fill_scheme_name(self, name: str) -> None:
        loc = self.page.locator("#csn-scheme-name")
        loc.wait_for(state="visible", timeout=20_000)
        loc.fill(name)

    def open_picker(self, kind: str) -> None:
        """kind: lottery | runType | playType | subPlay"""
        id_map = {
            "lottery": "csn-lbl-lottery",
            "runType": "csn-lbl-run",
            "playType": "csn-lbl-play",
            "subPlay": "csn-lbl-sub",
        }
        lbl = id_map[kind]
        # 点击同字段内的 picker 按钮
        field = self.page.locator(f"#{lbl}").locator("xpath=ancestor::div[contains(@class,'csn-field')][1]")
        field.locator("button.csn-picker").click()
        self.page.locator(".opm-panel").wait_for(state="visible", timeout=15_000)

    def list_picker_option_labels(self) -> list[str]:
        opts = self.page.locator(".opm-panel button.opm-option")
        opts.first.wait_for(state="visible", timeout=15_000)
        return [t.strip() for t in opts.all_text_contents() if t.strip()]

    def confirm_picker_by_label(self, label: str, *, retries: int = 3) -> None:
        last_err: Exception | None = None
        for attempt in range(max(1, retries)):
            try:
                panel = self.page.locator(".opm-panel")
                panel.wait_for(state="visible", timeout=15_000)
                opt = panel.get_by_role("option", name=label, exact=True)
                if opt.count() == 0:
                    # 模糊：包含关系 / 去「选」变体
                    labels = self.list_picker_option_labels()
                    matched = self._fuzzy_option(label, labels)
                    if not matched:
                        raise RuntimeError(f"无匹配选项 label={label!r} options={labels[:30]}")
                    opt = panel.get_by_role("option", name=matched, exact=True)
                opt.first.wait_for(state="visible", timeout=12_000)
                opt.first.click(timeout=12_000)
                panel.locator("button.opm-confirm").click(timeout=10_000)
                panel.wait_for(state="hidden", timeout=15_000)
                return
            except Exception as e:
                last_err = e
                self.page.wait_for_timeout(400)
                try:
                    if self.page.locator(".opm-panel").count() == 0:
                        break
                except Exception:
                    break
        raise RuntimeError(f"选择器确认失败 label={label!r}: {last_err}") from last_err

    @staticmethod
    def _fuzzy_option(wanted: str, options: list[str]) -> str | None:
        w = (wanted or "").strip()
        if not w:
            return None
        if w in options:
            return w
        # 任选二直选复式 ↔ 任二直选复式
        alts = {w, w.replace("任选", "任"), w.replace("任二", "任选二"), w.replace("任三", "任选三"), w.replace("任四", "任选四")}
        for o in options:
            if o in alts or o.replace("任选", "任") in alts:
                return o
        for o in options:
            if w in o or o in w:
                return o
        return None

    def open_and_confirm_picker(self, kind: str, label: str) -> None:
        """打开 picker 并确认；失败则重开再试。"""
        last_err: Exception | None = None
        for _ in range(3):
            try:
                self.open_picker(kind)
                self.confirm_picker_by_label(label, retries=2)
                return
            except Exception as e:
                last_err = e
                try:
                    self.close_picker()
                except Exception:
                    pass
                self.page.wait_for_timeout(500)
        raise RuntimeError(f"open_and_confirm_picker 失败 {kind}={label!r}: {last_err}") from last_err

    def close_picker(self) -> None:
        panel = self.page.locator(".opm-panel")
        if panel.count() and panel.is_visible():
            panel.locator("button.opm-close").click()
            panel.wait_for(state="hidden", timeout=10_000)

    def click_next(self) -> None:
        self.page.get_by_role("button", name=re.compile("下一步")).click()
        self.page.wait_for_url(re.compile(r"/play/bet-multiplier/advanced-scheme/"), timeout=60_000)

    def build_scheme_name(
        self,
        lottery: str,
        run_type: str,
        play_type: str,
        sub_play: str,
        trigger_label: str = "",
    ) -> str:
        stamp = datetime.now().strftime("%Y%m%d%H%M%S")
        base = f"{lottery}-{run_type}-{play_type}-{sub_play}"
        if trigger_label and trigger_label != "-":
            base = f"{base}-{trigger_label}"
        return f"{base}_{stamp}"

    def _dismiss_overlays(self) -> None:
        page = self.page
        for _ in range(3):
            try:
                page.keyboard.press("Escape")
                page.wait_for_timeout(120)
            except Exception:
                break

    def _fill_scf_input(self, element_id: str, value: str) -> None:
        """填充 el-input（id 可能在组件根或内部 native input）。"""
        page = self.page
        loc = page.locator(
            f"#{element_id} input, input#{element_id}, #{element_id}"
        )
        loc.first.wait_for(state="visible", timeout=45_000)
        el = loc.first
        el.scroll_into_view_if_needed()
        try:
            el.click(timeout=5_000)
        except Exception:
            pass
        el.fill(str(value), timeout=15_000)

    def fill_basic_config(self) -> None:
        """方案配置页：公开、资金、止损止盈、倍数、1元、正式运行。"""
        page = self.page
        page.locator("#scf-name, #scf-funds").first.wait_for(state="visible", timeout=60_000)
        self._dismiss_overlays()
        # 正式运行
        formal = page.get_by_role("button", name="正式运行")
        if formal.count():
            formal.first.click()
        # 分享公开：只点「分享状态」字段，避免误点投注单位等其它 select
        share_field = page.locator(".scf-field").filter(has_text="分享状态")
        if share_field.count() and share_field.locator(".el-select, .scf-el-select").count():
            share_field.locator(".el-select, .scf-el-select").first.click()
            opt = page.get_by_role("option", name=re.compile("公开"))
            opt.first.wait_for(state="visible", timeout=12_000)
            opt.first.click()
            self._dismiss_overlays()
        self._fill_scf_input("scf-funds", str(SCHEME_FUNDS))
        self._fill_scf_input("scf-sl", str(STOP_LOSS))
        self._fill_scf_input("scf-tp", str(TAKE_PROFIT))
        self._fill_scf_input("scf-mult", str(MULT_COEFF))
        # 投注单位
        unit_wrap = page.locator(".scf-field").filter(has_text="投注单位")
        if unit_wrap.count():
            unit_wrap.locator(".scf-el-select, .el-select").first.click()
            page.get_by_role("option", name=BET_UNIT_LABEL, exact=True).click()
            self._dismiss_overlays()

    def set_simple_bet_multiplier(self) -> None:
        page = self.page
        page.locator("button.scf-mode-card").click()
        page.wait_for_url(re.compile(r"bet-multiplier-settings"), timeout=30_000)
        page.get_by_text("简单倍投", exact=True).click()
        page.get_by_role("button", name="确认").click()
        page.wait_for_url(re.compile(r"/play/bet-multiplier/advanced-scheme/"), timeout=30_000)
        page.locator("#scf-name, #scf-funds").first.wait_for(state="visible", timeout=30_000)

    def set_adv_trigger_mode(self, mode_label: str) -> None:
        """高级开某投某：点全部随机 + 投向模式 radio。"""
        page = self.page
        rnd = page.get_by_role("button", name="全部随机")
        if rnd.count():
            rnd.first.click()
            page.wait_for_timeout(500)
        # 投向模式是 el-radio，不是 el-option
        radio = page.locator(".scf-radio-wrap, .el-radio-group").get_by_text(mode_label, exact=True)
        if radio.count():
            radio.first.click()
            page.wait_for_timeout(200)
            return
        label = page.get_by_text(mode_label, exact=True)
        if label.count():
            label.first.click()
            page.wait_for_timeout(200)

    def open_jushu_dialog(self) -> None:
        page = self.page
        btn = page.get_by_role("button", name=re.compile("添加局数"))
        if not btn.count():
            raise RuntimeError("未找到「添加局数」按钮")
        btn.first.click()
        page.locator(".el-dialog").filter(has_text=re.compile("投注号码|添加局数")).first.wait_for(
            state="visible", timeout=15_000
        )
        page.wait_for_timeout(300)

    def confirm_jushu_dialog(self) -> None:
        page = self.page
        dlg = page.locator(".el-dialog").filter(has_text=re.compile("投注号码|添加局数"))
        confirm = dlg.get_by_role("button", name=re.compile("确认添加|确认"))
        if not confirm.count():
            confirm = page.get_by_role("button", name=re.compile("确认添加"))
        confirm.first.click()
        try:
            dlg.first.wait_for(state="hidden", timeout=10_000)
        except Exception:
            page.keyboard.press("Escape")
        page.wait_for_timeout(300)

    def pick_hcw_digits(self, per_pos: int = 1) -> list[list[str]]:
        """冷热温：每位点 per_pos 个 .scf-hcw-chip。"""
        page = self.page
        section = page.locator(".scf-section").filter(has_text="方案内容")
        positions = section.locator(".scf-hcw-pos")
        if positions.count() == 0:
            raise RuntimeError("冷热温面板未出现")
        line_picks: list[list[str]] = []
        for pi in range(positions.count()):
            pos = positions.nth(pi)
            chips = pos.locator(".scf-hcw-chip:not(.is-on)")
            if chips.count() == 0:
                chips = pos.locator(".scf-hcw-chip")
            picked: list[str] = []
            for i in range(min(max(per_pos, 1), chips.count())):
                chip = chips.nth(i)
                text = (chip.inner_text() or "").strip()
                chip.click()
                if text:
                    picked.append(text)
                page.wait_for_timeout(60)
            line_picks.append(picked)
        page.wait_for_timeout(200)
        return line_picks

    def ensure_random_draw_ready(self) -> list[list[str]]:
        """随机出号：默认每位 1 码；生成预览后返回预览号码。"""
        page = self.page
        section = page.locator(".scf-section").filter(has_text="方案内容")
        if section.locator(".scf-rd-row").count() == 0:
            raise RuntimeError("随机出号面板未出现")
        gen = section.get_by_role("button", name=re.compile("生成预览"))
        if gen.count():
            gen.first.click()
            page.wait_for_timeout(400)
        line_picks: list[list[str]] = []
        rows = section.locator(".scf-rd-row")
        for i in range(rows.count()):
            prev = rows.nth(i).locator(".scf-rd-preview")
            text = (prev.inner_text() if prev.count() else "") or ""
            m = re.search(r"预览：\s*([0-9,，\s]+)", text)
            if m:
                digits = [x.strip() for x in re.split(r"[,，\s]+", m.group(1)) if x.strip()]
            else:
                digits = [str(i % 10)]
            line_picks.append(digits or [str(i % 10)])
        return line_picks

    def content_section(self):
        page = self.page
        dlg = page.locator(".el-dialog").filter(has_text=re.compile("投注号码|添加局数"))
        if dlg.count():
            try:
                if dlg.first.is_visible():
                    return dlg.first
            except Exception:
                pass
        return page.locator(".scf-section").filter(has_text="方案内容")

    def detect_content_panel(self) -> str:
        """返回 position | pool | danshi | hcw | random | trigger | jushu | empty。"""
        page = self.page
        main = page.locator(".scf-section").filter(has_text="方案内容")
        if main.locator(".scf-hcw-chip, .scf-hcw-pos").count() > 0:
            return "hcw"
        if main.locator(".scf-rd-row").count() > 0:
            return "random"
        if main.locator(".scf-trig-grid").count() > 0:
            return "trigger"
        if main.locator(".scf-jushu-list, .scf-jushu-row").count() > 0 or main.get_by_text(
            "暂无局数", exact=False
        ).count():
            return "jushu"
        section = self.content_section()
        if section.locator(".sgp-row").count() > 0:
            return "position"
        if section.locator("textarea, .el-textarea__inner").count() > 0:
            return "danshi"
        if section.locator(".sgp-chip").count() > 0:
            return "pool"
        return "empty"

    def wait_content_panel(self, prefer: str = "") -> str:
        """等到方案内容面板出现；prefer=position|pool|danshi。"""
        for _ in range(25):
            kind = self.detect_content_panel()
            if kind == "empty":
                self.page.wait_for_timeout(200)
                continue
            if prefer == "position" and kind == "pool":
                self.page.wait_for_timeout(300)
                kind2 = self.detect_content_panel()
                if kind2 == "position":
                    return kind2
                # 继续等 row；超时仍返回当前
                continue
            return kind
        return self.detect_content_panel()

    def fill_scheme_danshi(self, text: str) -> None:
        section = self.content_section()
        box = section.locator("textarea, .el-textarea__inner").first
        box.wait_for(state="visible", timeout=10_000)
        box.fill(text)
        self.page.wait_for_timeout(300)

    def clear_scheme_chips(self) -> None:
        """取消方案内容区已选 chip，避免残留。"""
        section = self.content_section()
        active = section.locator(".sgp-chip.is-active")
        # 从后往前点，避免集合变化
        for i in range(min(active.count(), 40) - 1, -1, -1):
            try:
                active.nth(i).click(timeout=1_000)
                self.page.wait_for_timeout(40)
            except Exception:
                pass
        self.page.wait_for_timeout(150)

    def pick_renxuan_positions(self, pos_labels: list[str], digits: list[str]) -> list[list[str]]:
        """任选：只在指定位行选号（其余位保持空）。"""
        section = self.content_section()
        self.clear_scheme_chips()
        rows = section.locator(".sgp-row")
        line_picks: list[list[str]] = [[] for _ in pos_labels]
        for i, (lab, digit) in enumerate(zip(pos_labels, digits)):
            row = None
            for ri in range(rows.count()):
                pos = rows.nth(ri).locator(".sgp-pos")
                text = (pos.inner_text() if pos.count() else "") or ""
                text = text.strip()
                if text == lab or text.startswith(lab):
                    row = rows.nth(ri)
                    break
            if row is None and i < rows.count():
                row = rows.nth(i)
            if row is None:
                raise RuntimeError(f"任选未找到位行: {lab}")
            chip = row.locator(".sgp-chip").filter(has_text=digit)
            hit = None
            for ci in range(min(chip.count(), 12)):
                el = chip.nth(ci)
                if (el.inner_text() or "").strip() == str(digit):
                    hit = el
                    break
            if hit is None:
                hit = row.locator(".sgp-chip:not(.is-active)").first
                digit = (hit.inner_text() or "").strip()
            hit.click()
            line_picks[i] = [str(digit)]
            self.page.wait_for_timeout(80)
        self.page.wait_for_timeout(250)
        return line_picks

    def pick_pool_mid_numeric(self, count: int = 1) -> list[str]:
        """从号池挑中间数值（和值稳定用）。"""
        section = self.content_section()
        all_chips = section.locator(".sgp-chip")
        numeric: list[tuple[str, int]] = []
        for i in range(min(all_chips.count(), 80)):
            t = (all_chips.nth(i).inner_text() or "").strip()
            if t.isdigit():
                numeric.append((t, i))
        if not numeric:
            return self.pick_pool_chips(min_clicks=count)
        numeric.sort(key=lambda x: int(x[0]))
        # 优先 8~18
        ranked = [x for x in numeric if 8 <= int(x[0]) <= 18] or numeric
        mid = len(ranked) // 2
        chosen = ranked[max(0, mid - count + 1) : mid + 1][:count]
        if not chosen:
            chosen = ranked[:count]
        picked: list[str] = []
        for lab, idx in chosen:
            el = all_chips.nth(idx)
            cls = el.get_attribute("class") or ""
            if "is-active" not in cls:
                el.click()
                self.page.wait_for_timeout(80)
            picked.append(lab)
        self.page.wait_for_timeout(250)
        return picked

    def pick_pool_chips(self, min_clicks: int = 3, *, append: bool = False) -> list[str]:
        """单号池 / 属性 chip 选号。"""
        page = self.page
        section = self.content_section()
        chips = section.locator(".sgp-chip:not(.is-active)")
        if chips.count() == 0 and not append:
            chips = section.locator(".sgp-chip")
        n = chips.count()
        if n == 0:
            raise RuntimeError("号池中未找到可点 chip")
        limit = min(max(min_clicks, 1), n)
        picked: list[str] = []
        for i in range(limit):
            chip = chips.nth(i)
            try:
                chip.wait_for(state="visible", timeout=3_000)
            except Exception:
                break
            text = (chip.inner_text() or "").strip()
            chip.click()
            picked.append(text)
            page.wait_for_timeout(80)
        page.wait_for_timeout(250)
        return picked

    def pick_pool_chips_by_labels(
        self,
        labels: list[str],
        *,
        allow_fallback_first: bool = True,
    ) -> list[str]:
        """按指定文案点号池。"""
        section = self.content_section()
        picked: list[str] = []
        all_chips = section.locator(".sgp-chip")
        texts: list[tuple[str, int]] = []
        for i in range(min(all_chips.count(), 80)):
            t = (all_chips.nth(i).inner_text() or "").strip()
            if t:
                texts.append((t, i))

        def click_idx(idx: int, lab: str) -> None:
            el = all_chips.nth(idx)
            cls = el.get_attribute("class") or ""
            if "is-active" not in cls:
                el.click()
                self.page.wait_for_timeout(80)
            picked.append(lab)

        for lab in labels:
            hit_idx = next((i for t, i in texts if t == lab), None)
            if hit_idx is None:
                continue
            click_idx(hit_idx, lab)
        if picked:
            self.page.wait_for_timeout(250)
            return picked

        numeric = [(t, i) for t, i in texts if t.isdigit()]
        if numeric:
            numeric.sort(key=lambda x: int(x[0]))
            prefer = next((x for x in numeric if 8 <= int(x[0]) <= 18), None)
            if prefer is None and allow_fallback_first:
                prefer = numeric[len(numeric) // 2]
            if prefer is not None:
                click_idx(prefer[1], prefer[0])
                self.page.wait_for_timeout(250)
                return picked

        if not allow_fallback_first:
            return []
        return self.pick_pool_chips(min_clicks=max(1, len(labels) or 1))

    def click_one_more_chip(self) -> bool:
        section = self.content_section()
        chip = section.locator(".sgp-row").first.locator(".sgp-chip:not(.is-active)").first
        if chip.count() == 0:
            chip = section.locator(".sgp-chip:not(.is-active)").first
        if chip.count() == 0:
            return False
        chip.click()
        self.page.wait_for_timeout(100)
        return True

    def pick_scheme_content_chips(self, min_clicks: int = 3) -> tuple[list[str], list[list[str]]]:
        """
        方案内容按位选号。
        返回 (扁平号码列表, 按位号码列表)。
        """
        page = self.page
        section = self.content_section()
        rows = section.locator(".sgp-row")
        picked: list[str] = []
        line_picks: list[list[str]] = []

        if rows.count() > 0:
            for ri in range(rows.count()):
                row = rows.nth(ri)
                inactive = row.locator(".sgp-chip:not(.is-active)")
                chip = inactive.first if inactive.count() else row.locator(".sgp-chip").first
                chip.wait_for(state="visible", timeout=5_000)
                text = (chip.inner_text() or "").strip()
                chip.click()
                picked.append(text)
                line_picks.append([text])
                page.wait_for_timeout(120)
            first_inactive = rows.first.locator(".sgp-chip:not(.is-active)")
            extra = max(0, min_clicks - 1)
            avail = first_inactive.count()
            for i in range(min(extra, avail)):
                chip = first_inactive.nth(i)
                text = (chip.inner_text() or "").strip()
                chip.click()
                picked.append(text)
                if line_picks:
                    line_picks[0].append(text)
                page.wait_for_timeout(80)
            page.wait_for_timeout(300)
            return picked, line_picks

        # 无分行：当单池处理
        pool = self.pick_pool_chips(min_clicks=min_clicks)
        return pool, [pool] if pool else []

    def read_platform_bet_count(self) -> int:
        page = self.page
        patterns = (
            r"注数:\s*(\d+)",
            r"预估\s*(\d+)\s*注",
            r"已选\s*(\d+)\s*注",
            r"共\s*(\d+)\s*注",
        )
        last_text = ""
        for _ in range(25):
            for sel in (
                "text=/注数:\\s*\\d+/",
                "text=/预估\\s*\\d+\\s*注/",
                ".scf-area-meta",
                ".scf-hcw-pool-units",
            ):
                loc = page.locator(sel).first
                if not loc.count():
                    continue
                try:
                    if not loc.is_visible():
                        continue
                    last_text = (loc.inner_text() or "").strip()
                except Exception:
                    continue
                for pat in patterns:
                    m = re.search(pat, last_text)
                    if m:
                        return int(m.group(1))
            body = ""
            try:
                body = page.locator(".scf-section, .el-dialog__body").first.inner_text(timeout=1_000)
            except Exception:
                try:
                    body = page.locator("body").inner_text(timeout=1_000)
                except Exception:
                    body = ""
            for pat in patterns:
                m = re.search(pat, body or "")
                if m:
                    return int(m.group(1))
            page.wait_for_timeout(200)
        raise RuntimeError(f"无法解析注数: {last_text or '(empty)'}")

    def save_to_cloud(self) -> dict:
        """点击添加至云端，拦截 add-to-cloud 响应，返回 data（含 definition/instance）。"""
        page = self.page

        def _match(resp) -> bool:
            try:
                u = resp.url
                return (
                    resp.request.method == "POST"
                    and "add-to-cloud" in u
                    and "fork-and-add-to-cloud" not in u
                ) or (
                    resp.request.method == "POST" and "fork-and-add-to-cloud" in u
                )
            except Exception:
                return False

        with page.expect_response(_match, timeout=120_000) as ri:
            btn = page.get_by_role("button", name=re.compile("添加至云端"))
            btn.click()
        resp = ri.value
        raw = resp.json()
        data = raw.get("data") if isinstance(raw, dict) and "data" in raw else raw
        # 关闭可能的成功弹窗
        page.wait_for_timeout(800)
        for name in ("确定", "确认", "我知道了", "前往云端中心"):
            b = page.get_by_role("button", name=name)
            try:
                if b.count() and b.first.is_visible():
                    b.first.click()
                    break
            except Exception:
                pass
        return data if isinstance(data, dict) else {}

    def read_access_token(self) -> str:
        return self.page.evaluate(
            """() => localStorage.getItem('client_access_token') || ''"""
        )

    def ensure_guaji_auth(self, api_base: str) -> None:
        """若第三方挂机授权过期，调用 reauth 恢复（否则无法进自创方案页）。"""
        import httpx

        token = self.read_access_token()
        if not token:
            raise RuntimeError("无 access token，无法检查挂机授权")
        base = api_base.rstrip("/")
        headers = {"Authorization": f"Bearer {token}"}
        with httpx.Client(base_url=base, timeout=45, trust_env=False, headers=headers) as c:
            st = c.get("/client/guaji/auth-status")
            st.raise_for_status()
            status = (st.json() or {}).get("data") or st.json() or {}
            expired = bool(status.get("activeAuthExpired"))
            has_active = bool(status.get("hasActiveGuajiAuth"))
            if has_active and not expired:
                print("[auth] guaji ok", flush=True)
                return
            acc = c.get("/client/guaji/accounts")
            acc.raise_for_status()
            raw = (acc.json() or {}).get("data") or acc.json() or {}
            items = raw.get("items") if isinstance(raw, dict) else raw
            if not isinstance(items, list) or not items:
                raise RuntimeError("无挂机授权账号，请先在会员中心绑定第三方账号")
            target = next((x for x in items if x.get("isActive") and x.get("authExpired")), None)
            if target is None:
                target = next((x for x in items if x.get("authExpired")), None)
            if target is None and not has_active:
                target = next((x for x in items if x.get("isActive")), items[0])
            if target is None:
                raise RuntimeError(f"挂机授权异常 status={status}")
            aid = target.get("id")
            print(
                f"[auth] reauth account id={aid} user={target.get('guajiUsername')}",
                flush=True,
            )
            r = c.post(f"/client/guaji/accounts/{aid}/reauth")
            r.raise_for_status()
            body = (r.json() or {}).get("data") or r.json() or {}
            if isinstance(body, dict) and body.get("authExpired"):
                raise RuntimeError(f"重新授权失败: {body.get('lastTokenError') or body}")
            st2 = c.get("/client/guaji/auth-status")
            st2.raise_for_status()
            status2 = (st2.json() or {}).get("data") or st2.json() or {}
            if not status2.get("hasActiveGuajiAuth") or status2.get("activeAuthExpired"):
                raise RuntimeError(f"重新授权后仍不可用: {status2}")
            print("[auth] guaji reauth ok", flush=True)

    def goto_cloud_center(self) -> None:
        self.page.goto(f"{self.base_url}/cloud", wait_until="domcontentloaded")
        self.page.wait_for_timeout(1_000)

    def start_scheme_by_name_ui(self, scheme_name: str) -> None:
        """云端中心按名称点开启（API 失败时的 UI 兜底）。"""
        self.goto_cloud_center()
        card = self.page.locator(".cc-card, [class*='scheme']").filter(has_text=scheme_name)
        if not card.count():
            # 宽松：整页找名称再找邻近按钮
            self.page.get_by_text(scheme_name, exact=False).first.click()
        btn = self.page.get_by_role("button", name=re.compile("开启方案|开启")).first
        btn.click()
        confirm = self.page.get_by_role("button", name=re.compile("确定|确认|继续"))
        if confirm.count() and confirm.first.is_visible():
            confirm.first.click()
        self.page.wait_for_timeout(1_500)

    def cookies(self) -> list[dict]:
        return self.page.context.cookies()
