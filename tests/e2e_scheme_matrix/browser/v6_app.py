"""第三方 v6hs1 Playwright 页面对象。"""
from __future__ import annotations

import re

from pathlib import Path

from playwright.sync_api import Browser, BrowserContext, Page

from name_match import match_lottery_label
from play_profile import (
    SSC_POS_LABELS,
    PickMode,
    play_tab_candidates,
    sub_play_candidates,
)


class V6App:
    def __init__(self, page: Page, base_url: str) -> None:
        self.page = page
        self.base_url = base_url.rstrip("/")

    @classmethod
    def launch(
        cls,
        browser: Browser,
        base_url: str,
        *,
        storage_state: str | Path | None = None,
    ) -> tuple[BrowserContext, "V6App"]:
        kwargs: dict = {
            "viewport": {"width": 1400, "height": 900},
            "locale": "zh-CN",
        }
        if storage_state:
            p = Path(storage_state)
            if p.is_file():
                kwargs["storage_state"] = str(p)
                print(f"[v6] using storage_state={p}", flush=True)
        context = browser.new_context(**kwargs)
        page = context.new_page()
        return context, cls(page, base_url)

    def dismiss_dialogs(self) -> None:
        page = self.page
        for _ in range(4):
            closed = False
            for name in (r"^确定$", r"^确认$", r"^我知道了$", r"^关闭$"):
                btn = page.get_by_role("button", name=re.compile(name))
                try:
                    if btn.count() and btn.first.is_visible(timeout=300):
                        btn.first.click(timeout=2_000)
                        page.wait_for_timeout(200)
                        closed = True
                        break
                except Exception:
                    pass
            if closed:
                continue
            try:
                tip = page.get_by_text("24小时内不再弹出", exact=False)
                if tip.count() and tip.first.is_visible(timeout=300):
                    tip.first.click(timeout=2_000)
                    page.wait_for_timeout(200)
                    closed = True
            except Exception:
                pass
            if not closed:
                break

    def login(self, username: str, password: str) -> None:
        page = self.page
        print("[v6] goto home…", flush=True)
        page.goto(f"{self.base_url}/", wait_until="domcontentloaded", timeout=90_000)
        page.wait_for_timeout(2_000)
        self.dismiss_dialogs()

        if self._is_logged_in(username):
            print("[v6] already logged in", flush=True)
            self.dismiss_dialogs()
            return

        print("[v6] open login…", flush=True)
        opened = page.evaluate(
            """() => {
              const nodes = [...document.querySelectorAll('div,span,a,button')];
              for (const el of nodes) {
                const t = (el.textContent || '').trim();
                if (t !== '登录' && t !== '账号登录') continue;
                if (el.children.length > 2) continue;
                const r = el.getBoundingClientRect();
                if (r.width <= 0 || r.height <= 0 || r.y > 100 || r.width > 160) continue;
                el.click();
                return true;
              }
              return false;
            }"""
        )
        if not opened:
            for _ in range(3):
                self.dismiss_dialogs()
                if self._is_logged_in(username):
                    print("[v6] already logged in (after dismiss)", flush=True)
                    return
                candidates = [
                    page.get_by_text("登录", exact=True),
                    page.locator("text=登录"),
                ]
                for loc in candidates:
                    try:
                        if loc.count() and loc.first.is_visible(timeout=2_000):
                            loc.first.click(timeout=10_000, force=True)
                            opened = True
                            break
                    except Exception:
                        continue
                if opened:
                    break
                page.wait_for_timeout(800)
        if not opened:
            # 会话已登录时首页常无「登录」按钮，勿当失败
            if self._is_logged_in(username):
                print("[v6] already logged in (no login entry)", flush=True)
                self.dismiss_dialogs()
                return
            page.wait_for_timeout(1_000)
            self.dismiss_dialogs()
            if self._is_logged_in(username):
                print("[v6] already logged in (retry)", flush=True)
                return
            raise RuntimeError("未找到第三方「登录」入口")

        page.wait_for_timeout(800)
        # 切到「账号登录」页签（避免停在验证登录）
        try:
            tab = page.get_by_text("账号登录", exact=True)
            if tab.count() and tab.first.is_visible(timeout=1_000):
                tab.first.click(timeout=3_000)
                page.wait_for_timeout(400)
        except Exception:
            pass

        print("[v6] fill credentials…", flush=True)
        user = page.locator('input[name="username"]')
        pwd = page.locator('input[name="password"]')
        if not user.count():
            user = page.locator('input[placeholder*="账号"], input[placeholder*="用户"]')
        if not pwd.count():
            pwd = page.locator('input[type="password"]')
        user.first.wait_for(state="visible", timeout=20_000)
        user.first.fill(username)
        pwd.first.fill(password)

        submit = page.locator("button").filter(has_text=re.compile(r"^登录$"))
        if submit.count():
            submit.last.click(timeout=10_000)
        else:
            page.get_by_role("button", name=re.compile(r"登录")).last.click(timeout=10_000)
        page.wait_for_timeout(1_500)
        self.dismiss_dialogs()

        # 站点常显示「登录中请稍后…」，需等到消失
        for i in range(40):
            body = ""
            try:
                body = page.locator("body").inner_text() or ""
            except Exception:
                pass
            if "登录中" in body or "请稍后" in body:
                page.wait_for_timeout(500)
                continue
            if self._is_logged_in(username):
                print("[v6] login done", flush=True)
                self.dismiss_dialogs()
                return
            page.wait_for_timeout(500)
            if i in (8, 16, 24):
                self.dismiss_dialogs()
                print(f"[v6] login waiting… i={i}", flush=True)
        tip = ""
        try:
            tip = (page.locator("body").inner_text() or "")[:500]
        except Exception:
            pass
        # 站点开启阿里云 captcha 时自动登录会一直「登录中」
        raise RuntimeError(
            f"第三方登录未成功（未看到用户名 {username}）。"
            f"若 /auth/app_id 返回 captcha=true，请先运行 "
            f"`python save_v6_session.py` 手动过验证码并保存会话。"
            f" body≈{tip!r}"
        )

    def _is_logged_in(self, username: str) -> bool:
        page = self.page
        # 优先读 localStorage（UI 文案偶发未渲染时仍可靠）
        try:
            via_store = page.evaluate(
                """(user) => {
                  try {
                    const st = JSON.parse(localStorage.getItem('state') || '{}');
                    const u = st && st.user;
                    if (u && u.isAuthenticated && u.token) {
                      if (!user) return true;
                      const name = (u.username || u.name || u.account || '');
                      if (!name || String(name).includes(user) || user.includes(String(name))) return true;
                      return true; // 有有效 token 即视为已登录
                    }
                  } catch (e) {}
                  return false;
                }""",
                username,
            )
            if via_store:
                return True
        except Exception:
            pass
        try:
            if page.get_by_text(username, exact=False).count():
                if page.get_by_text(username, exact=False).first.is_visible(timeout=800):
                    return True
        except Exception:
            pass
        # 已登录常见：充值/提现/退出；登录框应消失
        return bool(
            page.evaluate(
                """(user) => {
                  const body = document.body ? document.body.innerText : '';
                  if (user && body.includes(user)) return true;
                  const hasLoginForm = [...document.querySelectorAll('input[type="password"]')]
                    .some(i => {
                      const r = i.getBoundingClientRect();
                      return r.width > 0 && r.height > 0;
                    });
                  if (hasLoginForm) return false;
                  return /充值|提现|退出|余额|会员/.test(body);
                }""",
                username,
            )
        )

    def open_lottery_menu(self) -> list[str]:
        page = self.page
        print("[v6] open lottery menu…", flush=True)
        self.dismiss_dialogs()
        # 顶部「彩票」多为 hover 下拉，需先暴露菜单再采标签
        opened = page.evaluate(
            """() => {
              const navs = [...document.querySelectorAll('div,span,a')].filter(el => {
                const t = (el.innerText || '').trim();
                if (t !== '彩票' && t !== '游戏' && t !== '购彩') return false;
                if (el.children.length > 3) return false;
                const r = el.getBoundingClientRect();
                return r.width > 0 && r.height > 0 && r.y < 120 && r.x < 800;
              });
              if (!navs.length) return false;
              const el = navs[0];
              el.dispatchEvent(new MouseEvent('mouseenter', {bubbles:true}));
              el.dispatchEvent(new MouseEvent('mouseover', {bubbles:true}));
              el.click();
              return true;
            }"""
        )
        if not opened:
            for name in ("彩票", "游戏", "购彩"):
                loc = page.get_by_text(name, exact=True)
                try:
                    if loc.count() and loc.first.is_visible(timeout=800):
                        loc.first.hover(timeout=3_000)
                        loc.first.click(timeout=3_000)
                        opened = True
                        print(f"[v6] clicked menu entry {name}", flush=True)
                        break
                except Exception:
                    continue
        else:
            print("[v6] hovered/clicked 彩票 nav", flush=True)
        page.wait_for_timeout(900)

        labels = self._collect_menu_labels()
        print(f"[v6] menu labels={len(labels)} sample={labels[:12]}", flush=True)
        if not labels:
            raise RuntimeError("未能打开第三方彩种菜单或菜单为空")
        return labels

    def open_lottery_by_platform_label(self, platform_label: str) -> str:
        page = self.page
        tabs0 = self.list_play_tabs()
        if len(tabs0) >= 3:
            print(f"[v6] already on lottery tabs={tabs0[:6]}", flush=True)
            return platform_label

        return self._open_lottery_fresh(platform_label)

    def ensure_lottery_ready(self, platform_label: str) -> str:
        """确保投注页玩法页签可用；丢失则回首页重开。"""
        tabs = self.list_play_tabs()
        if len(tabs) >= 3:
            return platform_label
        print("[v6] play tabs missing, reopen lottery…", flush=True)
        try:
            self.page.goto(f"{self.base_url}/", wait_until="domcontentloaded", timeout=60_000)
            self.page.wait_for_timeout(800)
            self.dismiss_dialogs()
        except Exception:
            pass
        return self._open_lottery_fresh(platform_label)

    def _open_lottery_fresh(self, platform_label: str) -> str:
        page = self.page
        last_err: Exception | None = None
        for attempt in range(3):
            try:
                return self._open_lottery_fresh_once(platform_label)
            except Exception as e:
                last_err = e
                print(f"[v6] open lottery retry {attempt+1}/3: {e}", flush=True)
                try:
                    page.goto(f"{self.base_url}/", wait_until="domcontentloaded", timeout=60_000)
                    page.wait_for_timeout(1000)
                    self.dismiss_dialogs()
                except Exception:
                    pass
        assert last_err is not None
        raise last_err

    def _open_lottery_fresh_once(self, platform_label: str) -> str:
        page = self.page
        labels = self.open_lottery_menu()
        matched = match_lottery_label(platform_label, labels)
        if not matched:
            labels = self.open_lottery_menu()
            matched = match_lottery_label(platform_label, labels)
        if not matched:
            raise RuntimeError(
                f"第三方菜单未匹配到彩种: {platform_label!r} candidates={labels[:30]}"
            )
        print(f"[v6] click lottery {matched!r}", flush=True)
        clicked = page.evaluate(
            """(name) => {
              const nodes = [...document.querySelectorAll('div,span,a,li,button')];
              const hits = [];
              for (const e of nodes) {
                const t = (e.textContent || '').trim();
                if (t !== name) continue;
                if (e.children.length > 2 && (e.innerText || '').trim() !== name) continue;
                const r = e.getBoundingClientRect();
                if (r.width <= 0 || r.height <= 0 || r.width > 400 || r.height > 80) continue;
                let score = 0;
                if (r.y >= 40 && r.y <= 520) score += 80;
                else if (r.y < 40) score -= 40;
                if (r.width >= 40 && r.width <= 200) score += 20;
                hits.push({ score, el: e, x: r.x + r.width/2, y: r.y + r.height/2 });
              }
              if (!hits.length) return null;
              hits.sort((a,b) => b.score - a.score);
              const best = hits[0];
              const clickable = best.el.closest('a,button,[role="button"],div') || best.el;
              clickable.scrollIntoView({block:'center', inline:'nearest'});
              clickable.click();
              return {x: best.x, y: best.y};
            }""",
            matched,
        )
        if not clicked:
            loc = page.get_by_text(matched, exact=True)
            n = min(loc.count(), 8)
            ok = False
            for i in range(n):
                try:
                    el = loc.nth(i)
                    if el.is_visible(timeout=500):
                        el.click(timeout=5_000, force=True)
                        ok = True
                        break
                except Exception:
                    continue
            if not ok:
                raise RuntimeError(f"无法点击第三方彩种: {matched}")
        page.wait_for_timeout(2_500)
        self.dismiss_dialogs()
        tabs: list[str] = []
        for _ in range(28):
            tabs = self.list_play_tabs()
            if len(tabs) >= 3:
                print(f"[v6] lottery ready tabs={tabs[:8]}", flush=True)
                return matched
            page.wait_for_timeout(400)
        # 菜单点击可能没进厅：再试一次坐标点击 / 强制点可见节点
        try:
            page.evaluate(
                """(name) => {
                  const nodes = [...document.querySelectorAll('div,span,a,li,button')];
                  for (const e of nodes) {
                    const t = (e.textContent || '').trim();
                    if (t !== name) continue;
                    const r = e.getBoundingClientRect();
                    if (r.width <= 0 || r.height <= 0) continue;
                    e.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
                    return true;
                  }
                  return false;
                }""",
                matched,
            )
            page.wait_for_timeout(2_000)
            self.dismiss_dialogs()
            for _ in range(15):
                tabs = self.list_play_tabs()
                if len(tabs) >= 3:
                    print(f"[v6] lottery ready (2nd click) tabs={tabs[:8]}", flush=True)
                    return matched
                page.wait_for_timeout(400)
        except Exception:
            pass
        raise RuntimeError(
            f"点击彩种后未出现玩法页签: {matched} url={page.url} tabs={tabs}"
        )

    def list_play_tabs(self) -> list[str]:
        return self.page.evaluate(
            """() => {
              const wanted = new Set(['常用玩法','一星','前三码','中三码','后三码','前二码','后二码','龙虎','任选','五星','四星','大小单双','不定位','前中后三','前后三','前后二','前后四','趣味']);
              const boxes = [...document.querySelectorAll('div')];
              for (const box of boxes) {
                const kids = [...box.children];
                if (kids.length < 3 || kids.length > 40) continue;
                const topTexts = kids.map(k => (k.innerText || '').trim().split('\\n')[0]);
                const hit = topTexts.filter(t => wanted.has(t));
                if (hit.length >= 4) return hit;
              }
              return [];
            }"""
        )

    def _play_panel_markers(self, play_type_label: str) -> tuple[str, ...]:
        """确认大类页签已切入投注区（勿只认「前三直选」，混合/特殊号也会挂在同页）。"""
        lab = (play_type_label or "").strip()
        if lab.endswith("三码"):
            lab = lab[:-1]
        if lab in ("前三", "中三", "后三"):
            return (
                f"{lab}直选",
                f"{lab}组选",
                f"{lab}混合",
                f"{lab}特殊",
                f"{lab}和值",
                "直选复式",
                "混合组选",
                "特殊号",
            )
        if lab in ("前二", "后二") or lab.endswith("二码"):
            short = lab.replace("码", "")
            return (f"{short}直选", f"{short}组选", "直选复式", "组选复式")
        return ()

    def select_play(self, play_type_label: str, sub_play_label: str) -> None:
        page = self.page
        self.dismiss_dialogs()
        if len(self.list_play_tabs()) < 3:
            # 页签丢失时无法切玩法
            raise RuntimeError(
                f"第三方玩法页签为空，无法切换 {play_type_label}/{sub_play_label}"
            )
        markers = self._play_panel_markers(play_type_label)
        clicked_tab = False
        for attempt in range(4):
            for cand in play_tab_candidates(play_type_label):
                if self._click_play_tab(cand):
                    clicked_tab = True
                    break
                page.wait_for_timeout(400)
            page.wait_for_timeout(700)
            body_tab = page.locator("body").inner_text()
            if markers and any(m in body_tab for m in markers):
                break
            if not markers and clicked_tab:
                break
            # 未切到目标大类：再点一次
            clicked_tab = False
            if attempt < 3:
                print(f"[v6] 玩法页签未就绪 attempt={attempt+1} play={play_type_label}")
        if not clicked_tab and not (markers and any(m in page.locator("body").inner_text() for m in markers)):
            tabs = self.list_play_tabs()
            raise RuntimeError(
                f"第三方未找到玩法页签: {play_type_label} "
                f"candidates={play_tab_candidates(play_type_label)} tabs={tabs}"
            )
        body_tab = page.locator("body").inner_text()
        if markers and not any(m in body_tab for m in markers):
            raise RuntimeError(
                f"第三方玩法页签切换失败（目标 {play_type_label}/{sub_play_label}）"
                f" markers={markers}"
            )

        if not sub_play_label:
            return
        clicked = False
        for cand in sub_play_candidates(play_type_label, sub_play_label):
            if self._click_text_near_play(cand):
                clicked = True
                break
        page.wait_for_timeout(600)
        body2 = page.locator("body").inner_text()
        # 校验子玩法：优先面板特征，避免「页签文案存在但未点中」
        expect_keys: list[str] = []
        panel_ok = False
        if "组三" in sub_play_label:
            expect_keys = ["组三"]
            panel_ok = self._digit_pool_0_9_ready()
            if not panel_ok:
                for cand in sub_play_candidates(play_type_label, sub_play_label):
                    self._click_text_near_play(cand)
                    page.wait_for_timeout(400)
                    if self._digit_pool_0_9_ready():
                        panel_ok = True
                        clicked = True
                        break
            body2 = page.locator("body").inner_text()
        elif "组六" in sub_play_label:
            expect_keys = ["组六", "前三组六", "中三组六", "后三组六"]
            panel_ok = self._digit_pool_0_9_ready()
            if not panel_ok:
                # 先点「组选」分区标题再点子玩法，避免仍停在直选复式多位号池
                for zone in ("前三组选", "中三组选", "后三组选", "组选"):
                    self._click_text_near_play(zone)
                    page.wait_for_timeout(200)
                retry_cands = list(sub_play_candidates(play_type_label, sub_play_label))
                for pref in ("前三组六", "中三组六", "后三组六", "组六"):
                    if pref not in retry_cands:
                        retry_cands.insert(0, pref)
                seen_z: set[str] = set()
                for cand in retry_cands:
                    if not cand or cand in seen_z:
                        continue
                    seen_z.add(cand)
                    self._click_text_near_play(cand)
                    page.wait_for_timeout(500)
                    if self._digit_pool_0_9_ready():
                        panel_ok = True
                        clicked = True
                        break
            body2 = page.locator("body").inner_text()
        elif "跨度" in sub_play_label:
            expect_keys = ["跨度", "直选跨度", "前三直选跨度", "中三直选跨度", "后三直选跨度"]
            panel_ok = self._kuadu_pool_ready()
            if not panel_ok:
                for zone in ("前三直选", "中三直选", "后三直选", "直选"):
                    self._click_text_near_play(zone)
                    page.wait_for_timeout(200)
                retry_cands = list(sub_play_candidates(play_type_label, sub_play_label))
                for pref in ("前三直选跨度", "中三直选跨度", "后三直选跨度", "直选跨度", "跨度"):
                    if pref not in retry_cands:
                        retry_cands.insert(0, pref)
                seen_k: set[str] = set()
                for cand in retry_cands:
                    if not cand or cand in seen_k:
                        continue
                    seen_k.add(cand)
                    self._click_text_near_play(cand)
                    page.wait_for_timeout(500)
                    if self._kuadu_pool_ready() or self._digit_pool_0_9_ready():
                        panel_ok = True
                        clicked = True
                        break
            body2 = page.locator("body").inner_text()
        elif "和值尾数" in sub_play_label or ("尾数" in sub_play_label and "和值" in sub_play_label):
            expect_keys = ["和值尾数", "尾数"]
            panel_ok = self._digit_pool_0_9_ready()
            if not panel_ok:
                for cand in sub_play_candidates(play_type_label, sub_play_label):
                    self._click_text_near_play(cand)
                    page.wait_for_timeout(400)
                    if self._digit_pool_0_9_ready():
                        panel_ok = True
                        clicked = True
                        break
            body2 = page.locator("body").inner_text()
        elif "包胆" in sub_play_label:
            expect_keys = ["包胆", "组选包胆"]
            panel_ok = self._digit_pool_0_9_ready()
            if not panel_ok:
                for zone in ("前三组选", "中三组选", "后三组选", "组选"):
                    self._click_text_near_play(zone)
                    page.wait_for_timeout(200)
                retry_cands = list(sub_play_candidates(play_type_label, sub_play_label))
                for pref in ("前三组选包胆", "中三组选包胆", "后三组选包胆", "组选包胆", "包胆"):
                    if pref not in retry_cands:
                        retry_cands.insert(0, pref)
                seen_b: set[str] = set()
                for cand in retry_cands:
                    if not cand or cand in seen_b:
                        continue
                    seen_b.add(cand)
                    self._click_text_near_play(cand)
                    page.wait_for_timeout(500)
                    if self._digit_pool_0_9_ready():
                        panel_ok = True
                        clicked = True
                        break
            body2 = page.locator("body").inner_text()
        elif "和值" in sub_play_label:
            expect_keys = ["直选和值", "组选和值", "和值"]
            panel_ok = self._hezhi_pool_ready()
            if not panel_ok:
                # 先按完整候选点，再扫短名；勿限制 cand 必须已在 candidates 里
                retry_cands = list(sub_play_candidates(play_type_label, sub_play_label))
                if "组选" in sub_play_label:
                    retry_cands = ["组选和值", "和值"] + retry_cands
                else:
                    retry_cands = ["直选和值", "和值"] + retry_cands
                seen: set[str] = set()
                for cand in retry_cands:
                    if not cand or cand in seen:
                        continue
                    seen.add(cand)
                    self._click_text_near_play(cand)
                    page.wait_for_timeout(500)
                    if self._hezhi_pool_ready():
                        panel_ok = True
                        clicked = True
                        break
            body2 = page.locator("body").inner_text()
            print(f"[v6] hezhi select panel_ok={panel_ok}", flush=True)
        elif "混合组选" in sub_play_label:
            expect_keys = ["混合组选"]
            panel_ok = self._danshi_panel_ready()
        elif "特殊号" in sub_play_label:
            expect_keys = ["豹子", "对子", "顺子", "特殊号"]
            panel_ok = any(x in body2 for x in ("豹子", "对子", "顺子"))
        elif "龙虎" in play_type_label or "龙虎" in sub_play_label:
            expect_keys = ["龙", "虎"]
        elif "单式" in sub_play_label:
            expect_keys = ["单式"]
            panel_ok = self._danshi_panel_ready()
        elif "复式" in sub_play_label and "任" not in sub_play_label:
            expect_keys = ["复式"]
        elif "不定位" in sub_play_label or play_type_label == "不定位":
            expect_keys = ["不定位"]
        if expect_keys and not panel_ok and not any(k in body2 for k in expect_keys):
            for cand in sub_play_candidates(play_type_label, sub_play_label):
                self._click_text_near_play(cand)
            page.wait_for_timeout(500)
            body2 = page.locator("body").inner_text()
            if "和值" in sub_play_label and "尾数" not in sub_play_label:
                panel_ok = self._hezhi_pool_ready()
            elif "混合组选" in sub_play_label or "单式" in sub_play_label:
                panel_ok = self._danshi_panel_ready()
            elif "特殊号" in sub_play_label:
                panel_ok = any(x in body2 for x in ("豹子", "对子", "顺子"))
        soft_expect = ("混合组选" in sub_play_label) or ("特殊号" in sub_play_label)
        need_panel = (
            ("组三" in sub_play_label)
            or ("组六" in sub_play_label)
            or ("包胆" in sub_play_label)
            or ("跨度" in sub_play_label)
            or ("和值" in sub_play_label and "尾数" not in sub_play_label)
        )
        if expect_keys and not panel_ok and need_panel:
            raise RuntimeError(
                f"第三方子玩法未切换到位: {sub_play_label} expect={expect_keys} panel_ok=False"
            )
        if expect_keys and not panel_ok and not any(k in body2 for k in expect_keys):
            if soft_expect:
                print(f"[v6] soft-miss subplay expect={expect_keys} sub={sub_play_label} clicked={clicked}")
            else:
                raise RuntimeError(
                    f"第三方子玩法未切换到位: {sub_play_label} expect={expect_keys}"
                )
        if not clicked and not soft_expect and not panel_ok:
            if not any(
                x in body2
                for x in (
                    "直选",
                    "组选",
                    "和值",
                    "跨度",
                    "不定位",
                    "龙",
                    "虎",
                    "单式",
                    "复式",
                    "组三",
                    "组六",
                    "特殊号",
                    "混合",
                )
            ):
                raise RuntimeError(
                    f"第三方未找到子玩法: {sub_play_label} "
                    f"tried={sub_play_candidates(play_type_label, sub_play_label)}"
                )

    def mirror_picks(
        self,
        picked_labels: list[str],
        *,
        line_picks: list[list[str]] | None = None,
        play_type_label: str = "",
        sub_play_label: str = "",
        mode: PickMode | str = PickMode.POSITION,
        danshi_text: str = "",
        position_labels: list[str] | None = None,
    ) -> None:
        page = self.page
        if play_type_label:
            # 页签丢失则由调用方 ensure；此处再兜底一次空 tabs
            if len(self.list_play_tabs()) < 3:
                raise RuntimeError(
                    f"第三方无玩法页签，无法镜像选号 {play_type_label}/{sub_play_label}"
                )
            self.select_play(play_type_label, sub_play_label)

        mode_s = mode.value if isinstance(mode, PickMode) else str(mode or PickMode.POSITION)
        is_hezhi = "和值" in (sub_play_label or "") and "尾数" not in (sub_play_label or "")
        if is_hezhi:
            for _ in range(10):
                if self._hezhi_pool_ready():
                    break
                page.wait_for_timeout(200)
        elif mode_s == PickMode.DANSHI.value:
            for _ in range(8):
                if self._danshi_panel_ready():
                    break
                page.wait_for_timeout(200)
        elif mode_s == PickMode.ATTR.value and "特殊号" in (sub_play_label or ""):
            for _ in range(8):
                body = page.locator("body").inner_text()
                if any(x in body for x in ("豹子", "对子", "顺子")):
                    break
                page.wait_for_timeout(200)

        if mode_s == PickMode.DANSHI.value:
            self._clear_bet_selection()
            text = danshi_text or ",".join(picked_labels)
            if not self._fill_danshi(text):
                raise RuntimeError(f"第三方单式输入失败: {text!r}")
            page.wait_for_timeout(400)
            return

        if mode_s == PickMode.RENXUAN.value:
            self._clear_bet_selection()
            pos = position_labels or []
            lines = line_picks or []
            if not pos and lines:
                pos = list(SSC_POS_LABELS[: len(lines)])
            self._select_renxuan_positions(pos)
            page.wait_for_timeout(400)
            # 按位选号
            for i, digits in enumerate(lines):
                plab = pos[i] if i < len(pos) else ""
                for d in digits:
                    if not self._pick_digit_on_position(plab, d):
                        if not self._pick_pool_label(d):
                            print(f"[v6] 任选未能点选 {plab}:{d}")
                    page.wait_for_timeout(120)
            page.wait_for_timeout(400)
            return

        if mode_s in (PickMode.POOL.value, PickMode.ATTR.value) or not line_picks:
            labels = picked_labels or [x for row in (line_picks or []) for x in row]
            pool_kind = "hezhi" if is_hezhi else ("attr" if mode_s == PickMode.ATTR.value else "pool")
            # 和值号池在点「清空」后偶发卸掉；仅当已有注数时再清
            if is_hezhi:
                try:
                    pre_n = self.read_preview_bet_count()
                except Exception:
                    pre_n = 0
                if pre_n > 0:
                    self._clear_bet_selection()
                    page.wait_for_timeout(300)
                for _ in range(15):
                    if self._hezhi_pool_ready():
                        break
                    page.wait_for_timeout(200)
                print(f"[v6] hezhi pool ready={self._hezhi_pool_ready()}", flush=True)
            else:
                self._clear_bet_selection()
            for label in labels:
                if not label:
                    continue
                if is_hezhi:
                    ok = self._pick_hezhi_value(label) or self._pick_pool_label(label, kind="hezhi")
                else:
                    ok = self._pick_pool_label(label, kind=pool_kind)
                if not ok:
                    print(f"[v6] 未能点选号池 {label}", flush=True)
                page.wait_for_timeout(120)
            page.wait_for_timeout(500)
            # 组三/组六：页内常残留「组三」文案导致误判已切入；注数为 0 时强制重选子玩法再点
            if ("组三" in (sub_play_label or "")) or ("组六" in (sub_play_label or "")):
                try:
                    n = self.read_preview_bet_count()
                except Exception:
                    n = 0
                if n <= 0 and labels:
                    for attempt in range(3):
                        print(
                            f"[v6] 组选注数=0，重选子玩法后重试 {labels} attempt={attempt+1}",
                            flush=True,
                        )
                        self.select_play(play_type_label, sub_play_label)
                        self._clear_bet_selection()
                        page.wait_for_timeout(300)
                        for label in labels:
                            self._pick_pool_label(label, kind="pool")
                            page.wait_for_timeout(150)
                        page.wait_for_timeout(400)
                        try:
                            n = self.read_preview_bet_count()
                        except Exception:
                            n = 0
                        if n > 0:
                            break
            # 和值：注数异常偏低 → 重选子玩法后同号重试（不清空后再点）
            if is_hezhi:
                try:
                    n = self.read_preview_bet_count()
                except Exception:
                    n = 0
                if n <= 1 and labels:
                    for attempt in range(3):
                        print(
                            f"[v6] 和值注数异常 n={n}，重选子玩法后重试 {labels} attempt={attempt+1}",
                            flush=True,
                        )
                        self.select_play(play_type_label, sub_play_label)
                        for _ in range(12):
                            if self._hezhi_pool_ready():
                                break
                            page.wait_for_timeout(250)
                        for label in labels:
                            self._pick_hezhi_value(label) or self._pick_pool_label(
                                label, kind="hezhi"
                            )
                            page.wait_for_timeout(150)
                        page.wait_for_timeout(400)
                        try:
                            n = self.read_preview_bet_count()
                        except Exception:
                            n = 0
                        if n > 1:
                            break
            return

        self._clear_bet_selection()
        pos_labels = self._detect_position_labels()
        if len(pos_labels) >= len(line_picks):
            if len(pos_labels) > len(line_picks) and play_type_label.startswith("前"):
                pos_labels = pos_labels[: len(line_picks)]
            elif len(pos_labels) > len(line_picks) and play_type_label.startswith("后"):
                pos_labels = pos_labels[-len(line_picks) :]
            else:
                pos_labels = pos_labels[: len(line_picks)]
        for pos, digits in zip(pos_labels, line_picks):
            for d in digits:
                if not self._pick_digit_on_position(pos, d):
                    print(f"[v6] 未能点选 {pos}:{d}")
                page.wait_for_timeout(120)
        page.wait_for_timeout(400)

    def read_preview_bet_count(self) -> int:
        page = self.page
        body = page.locator("body").inner_text()
        patterns = [
            r"已选\s*(\d+)\s*注",
            r"已选\s*\n\s*(\d+)\s*\n\s*注",
            r"注数\s*[:：]?\s*(\d+)",
            r"共\s*(\d+)\s*注",
        ]
        for pat in patterns:
            m = re.search(pat, body, flags=re.M)
            if m:
                return int(m.group(1))
        loc = page.locator("text=/已选/").first
        if loc.count():
            t = loc.inner_text()
            m = re.search(r"(\d+)", t)
            if m:
                return int(m.group(1))
            try:
                nearby = loc.evaluate(
                    """(e) => {
                      const p = e.parentElement;
                      return p ? p.innerText : e.innerText;
                    }"""
                )
                m = re.search(r"已选\s*(\d+)\s*注", nearby or "")
                if m:
                    return int(m.group(1))
            except Exception:
                pass
        raise RuntimeError("无法读取第三方预览注数")

    def _click_play_tab(self, name: str) -> bool:
        rect = self.page.evaluate(
            """(name) => {
              const wanted = ['常用玩法','一星','前三码','中三码','后三码','前二码','后二码','龙虎','任选','五星','四星','大小单双','不定位','前中后三','前后三','前后二','前后四','趣味'];
              const boxes = [...document.querySelectorAll('div')];
              for (const box of boxes) {
                const kids = [...box.children];
                if (kids.length < 3 || kids.length > 40) continue;
                const topTexts = kids.map(k => (k.innerText || '').trim().split('\\n')[0]);
                const hit = topTexts.filter(t => wanted.includes(t));
                if (hit.length < 3) continue;
                const idx = topTexts.findIndex(t => t === name);
                if (idx < 0) continue;
                const el = kids[idx];
                el.scrollIntoView({block:'center', inline:'nearest'});
                const r = el.getBoundingClientRect();
                if (r.width <= 0 || r.height <= 0) continue;
                return {x: r.x + r.width / 2, y: r.y + r.height / 2};
              }
              const nodes = [...document.querySelectorAll('div,span')];
              for (const el of nodes) {
                const t = (el.innerText || '').trim();
                if (t !== name) continue;
                const r = el.getBoundingClientRect();
                if (r.width <= 0 || r.height <= 0 || r.width > 120 || r.height > 60) continue;
                const parent = el.parentElement;
                if (!parent) continue;
                const sibs = [...parent.children].map(c => (c.innerText||'').trim().split('\\n')[0]);
                if (sibs.filter(s => wanted.includes(s)).length < 2) continue;
                el.scrollIntoView({block:'center', inline:'nearest'});
                return {x: r.x + r.width / 2, y: r.y + r.height / 2};
              }
              return null;
            }""",
            name,
        )
        if not rect:
            return False
        self.page.mouse.click(rect["x"], rect["y"])
        self.page.wait_for_timeout(800)
        return True

    def _click_text_near_play(self, name: str) -> bool:
        """点击与 name 精确匹配的文案，优先选投注区附近（避开顶部导航）。"""
        rect = self.page.evaluate(
            """(name) => {
              const nodes = [...document.querySelectorAll('div,span,a,button,li')];
              const hits = [];
              for (const e of nodes) {
                const t = (e.textContent || '').trim();
                if (t !== name) continue;
                if (e.children.length > 0 && (e.innerText || '').trim() !== name) continue;
                const r = e.getBoundingClientRect();
                if (r.width <= 0 || r.height <= 0 || r.width > 400 || r.height > 80) continue;
                let score = 0;
                // 子玩法条通常在 ~320–400；号池/属性在更下方
                if (r.y >= 300 && r.y <= 420) score += 120;
                else if (r.y >= 280 && r.y <= 560) score += 100;
                else if (r.y >= 200 && r.y <= 650) score += 40;
                else score -= 40;
                if (r.x >= 80 && r.x <= 1000) score += 20;
                // 长文案（前三混合组选）优先于短歧义（混合）
                score += Math.min(40, name.length * 3);
                score += Math.min(30, r.width);
                hits.push({ score, x: r.x + r.width / 2, y: r.y + r.height / 2, el: e });
              }
              if (!hits.length) return null;
              hits.sort((a, b) => b.score - a.score);
              const best = hits[0];
              best.el.scrollIntoView({ block: 'center' });
              const r2 = best.el.getBoundingClientRect();
              return { x: r2.x + r2.width / 2, y: r2.y + r2.height / 2 };
            }""",
            name,
        )
        if not rect:
            return False
        self.page.mouse.click(rect["x"], rect["y"])
        self.page.wait_for_timeout(400)
        return True

    def _select_renxuan_positions(self, labels: list[str]) -> None:
        """任选：勾选万/千/百/十/个等位（第三方多为 checkbox/可点标签）。"""
        if not labels:
            return
        page = self.page
        # 先尝试点「全清/清空位置」类，再逐个勾选
        for name in ("清空", "复位"):
            self._click_text_near_play(name)
        page.wait_for_timeout(200)
        for lab in labels:
            aliases = [lab, f"{lab}位", lab.replace("位", "")]
            clicked = False
            for a in aliases:
                if not a:
                    continue
                # checkbox 邻近文本
                ok = page.evaluate(
                    """(name) => {
                      const nodes = [...document.querySelectorAll('label,span,div,button,li')];
                      const el = nodes.find(e => {
                        const t = (e.innerText || '').trim();
                        return t === name || t === name + '位';
                      });
                      if (!el) return null;
                      const box = el.closest('label') || el;
                      const r = box.getBoundingClientRect();
                      if (r.width <= 0 || r.height <= 0) return null;
                      box.scrollIntoView({block:'center'});
                      return {x: r.x + Math.min(12, r.width/2), y: r.y + r.height/2};
                    }""",
                    a,
                )
                if ok:
                    page.mouse.click(ok["x"], ok["y"])
                    page.wait_for_timeout(150)
                    clicked = True
                    break
                if self._click_text_near_play(a):
                    clicked = True
                    break
            if not clicked:
                print(f"[v6] 任选未能勾选位置 {lab}")

    def _clear_bet_selection(self) -> None:
        """清空当前选号，避免上一玩法残留注数干扰。"""
        page = self.page
        for name in ("清空", "复位", "清除", "重选"):
            btn = page.get_by_role("button", name=name)
            try:
                if btn.count() and btn.first.is_visible(timeout=400):
                    btn.first.click()
                    page.wait_for_timeout(300)
                    return
            except Exception:
                pass
        # 仅点投注区内文案，避免误点导航「清空」类入口；不要 scrollIntoView 以免号池滚出视口
        ok = page.evaluate(
            """() => {
              const names = new Set(['清空', '复位', '清除', '重选', '清']);
              const nodes = [...document.querySelectorAll('button,div,span,a')];
              for (const el of nodes) {
                const t = (el.textContent || '').trim();
                if (!names.has(t)) continue;
                const r = el.getBoundingClientRect();
                if (r.width <= 0 || r.height <= 0 || r.width > 120 || r.y < 350 || r.y > 850) continue;
                return {x: r.x + r.width/2, y: r.y + r.height/2};
              }
              return null;
            }"""
        )
        if ok:
            page.mouse.click(ok["x"], ok["y"])
            page.wait_for_timeout(300)

    def _hezhi_pool_ready(self) -> bool:
        """页面是否已出现和值 0~27 号池（要求同一父节点下成组，避免页内散落数字误判）。"""
        return bool(
            self.page.evaluate(
                """() => {
                  const isLeaf = (n) => {
                    const t = (n.textContent || '').trim();
                    if (!/^[0-9]{1,2}$/.test(t)) return false;
                    const v = Number(t);
                    if (v > 27) return false;
                    if (n.children.length > 0 && (n.innerText || '').trim() !== t) return false;
                    const r = n.getBoundingClientRect();
                    return r.width > 8 && r.width < 100 && r.height > 8 && r.height < 80
                      && r.y > 280 && r.y < 920;
                  };
                  const leaves = [...document.querySelectorAll('div,span,button,li,a')].filter(isLeaf);
                  const byParent = new Map();
                  for (const n of leaves) {
                    const p = n.parentElement;
                    if (!p) continue;
                    if (!byParent.has(p)) byParent.set(p, []);
                    byParent.get(p).push(n);
                  }
                  for (const nodes of byParent.values()) {
                    const nums = [...new Set(nodes.map(n => Number((n.textContent||'').trim())))]
                      .filter(x => x >= 0 && x <= 27)
                      .sort((a,b)=>a-b);
                    // 直选和值约 0~27；组选和值约 1~26；允许略少
                    if (nums.length >= 18 && nums[0] <= 1 && nums[nums.length-1] >= 18) {
                      return true;
                    }
                  }
                  return false;
                }"""
            )
        )

    def _digit_pool_0_9_ready(self) -> bool:
        """组三/组六：存在成组 0~9 单行号池（不依赖当前滚动位置）。"""
        return bool(
            self.page.evaluate(
                """() => {
                  const body = document.body ? (document.body.innerText || '') : '';
                  const posHits = ['万位','千位','百位','十位','个位'].filter(x => body.includes(x)).length;
                  // 仍停在直选多位时
                  if (posHits >= 3) return false;
                  let rows = 0;
                  for (const p of document.querySelectorAll('div,ul,section,span')) {
                    const kids = [...p.children];
                    if (kids.length < 8 || kids.length > 16) continue;
                    const nums = [...new Set(kids.map(c => (c.textContent || '').trim()).filter(t => /^[0-9]$/.test(t)))]
                      .map(Number).sort((a,b)=>a-b);
                    if (nums.length >= 8 && nums[0] === 0 && nums[nums.length - 1] === 9) {
                      rows += 1;
                    }
                  }
                  // 直选复式通常多行；组三/组六通常 1 行（偶发检测噪声允许 1~2）
                  return rows >= 1 && rows <= 2;
                }"""
            )
        )

    def _kuadu_pool_ready(self) -> bool:
        """直选跨度：成组 0~9 号池，且页面出现跨度文案；允许仍带少量定位文案。"""
        return bool(
            self.page.evaluate(
                """() => {
                  const body = document.body ? (document.body.innerText || '') : '';
                  if (!body.includes('跨度')) return false;
                  let rows = 0;
                  for (const p of document.querySelectorAll('div,ul,section,span')) {
                    const kids = [...p.children];
                    if (kids.length < 8 || kids.length > 16) continue;
                    const nums = [...new Set(kids.map(c => (c.textContent || '').trim()).filter(t => /^[0-9]$/.test(t)))]
                      .map(Number).sort((a,b)=>a-b);
                    if (nums.length >= 8 && nums[0] === 0 && nums[nums.length - 1] === 9) {
                      rows += 1;
                    }
                  }
                  return rows >= 1 && rows <= 2;
                }"""
            )
        )

    def _pick_hezhi_value(self, label: str) -> bool:
        """点和值号：优先点「父节点兄弟含 0~27」的 chip（与 V6 DOM 对齐）。"""
        label = (label or "").strip()
        if not label:
            return False
        rect = self.page.evaluate(
            """(wanted) => {
              const nodes = [...document.querySelectorAll('div,span,button,li,a')];
              const hits = [];
              for (const e of nodes) {
                const t = (e.textContent || '').trim();
                if (t !== wanted) continue;
                if (e.children.length > 0 && (e.innerText || '').trim() !== wanted) continue;
                const r = e.getBoundingClientRect();
                if (r.width < 8 || r.width > 90 || r.height < 8 || r.height > 70) continue;
                if (r.y < 400 || r.y > 780) continue;
                const p = e.parentElement;
                if (!p) continue;
                const sibs = [...p.children].map(c => (c.textContent || '').trim());
                const nums = [...new Set(sibs)].filter(v => /^[0-9]{1,2}$/.test(v)).map(Number)
                  .filter(n => n >= 0 && n <= 27);
                if (nums.length < 15 || Math.min(...nums) > 1 || Math.max(...nums) < 18) continue;
                hits.push({
                  score: nums.length * 10 + (r.y >= 480 && r.y <= 600 ? 40 : 0),
                  x: r.x + r.width / 2,
                  y: r.y + r.height / 2,
                  el: e,
                });
              }
              if (!hits.length) return null;
              hits.sort((a, b) => b.score - a.score);
              hits[0].el.scrollIntoView({ block: 'center' });
              const r2 = hits[0].el.getBoundingClientRect();
              return { x: r2.x + r2.width / 2, y: r2.y + r2.height / 2 };
            }""",
            label,
        )
        if not rect:
            return False
        self.page.mouse.click(rect["x"], rect["y"])
        self.page.wait_for_timeout(200)
        return True

    def _danshi_panel_ready(self) -> bool:
        return bool(
            self.page.evaluate(
                """() => {
                  const tas = [...document.querySelectorAll('textarea')];
                  return tas.some(t => {
                    const r = t.getBoundingClientRect();
                    return r.width > 80 && r.height > 30 && r.y > 200 && r.y < 850;
                  });
                }"""
            )
        )

    def _pick_digit_in_0_9_pool(self, label: str) -> bool:
        """组三/组六等：在成组 0~9 号球上点击（避开「已选 0 注」等文案）。"""
        label = (label or "").strip()
        if not label.isdigit() or int(label) > 9:
            return False
        ok = self.page.evaluate(
            """(wanted) => {
              const rowParents = [];
              for (const p of document.querySelectorAll('div,ul,section,span')) {
                const kids = [...p.children];
                if (kids.length < 8 || kids.length > 16) continue;
                const nums = [...new Set(kids.map(c => (c.textContent || '').trim()).filter(t => /^[0-9]$/.test(t)))]
                  .map(Number).sort((a,b)=>a-b);
                if (nums.length >= 8 && nums[0] === 0 && nums[nums.length - 1] === 9) {
                  rowParents.push(p);
                }
              }
              if (!rowParents.length) return false;
              // 优先视口内或靠近中部的号池行
              rowParents.sort((a, b) => {
                const ra = a.getBoundingClientRect();
                const rb = b.getBoundingClientRect();
                const score = (r) => {
                  let s = 0;
                  if (r.y >= 200 && r.y <= 700) s += 50;
                  if (r.width > 200) s += 20;
                  return s - Math.abs(r.y - 480);
                };
                return score(rb) - score(ra);
              });
              const row = rowParents[0];
              row.scrollIntoView({ block: 'center', inline: 'nearest' });
              const kids = [...row.children];
              let hit = null;
              for (const c of kids) {
                const t = (c.textContent || '').trim();
                if (t !== wanted) continue;
                // 优先点内层实心球
                const inner = [...c.querySelectorAll('div,span,button')].find(n => {
                  const tt = (n.textContent || '').trim();
                  return tt === wanted && n.children.length === 0;
                });
                hit = inner || c;
                break;
              }
              if (!hit) return false;
              hit.scrollIntoView({ block: 'center', inline: 'nearest' });
              hit.click();
              return true;
            }""",
            label,
        )
        if ok:
            self.page.wait_for_timeout(200)
        return bool(ok)

    def _pick_pool_label(self, label: str, *, kind: str = "pool") -> bool:
        """在和值/跨度/组选/不定位/龙虎/特殊号等号池中点选。

        kind: hezhi | attr | pool — 和值会忽略金额「50」等离群，并按空间邻近识别 0~27 号池。
        """
        label = (label or "").strip()
        if not label:
            return False
        if kind == "pool" and label.isdigit() and int(label) <= 9:
            if self._pick_digit_in_0_9_pool(label):
                return True
        if label in ("龙", "虎", "和", "大", "小", "单", "双", "豹子", "对子", "顺子", "极大", "极小"):
            if self._click_text_near_play(label):
                return True
        for _ in range(3):
            rect = self.page.evaluate(
                """({label, kind}) => {
                  const wanted = label.trim();
                  const attrs = new Set(['龙','虎','和','大','小','单','双','豹子','对子','顺子','极大','极小']);
                  const isAttr = kind === 'attr' || attrs.has(wanted);
                  const preferHezhi = kind === 'hezhi';
                  const isLeaf = (n) => {
                    const t = (n.textContent || '').trim();
                    if (!t) return false;
                    if (n.children.length > 0 && (n.innerText || '').trim() !== t) return false;
                    const r = n.getBoundingClientRect();
                    return r.width > 8 && r.width < 90 && r.height > 8 && r.height < 70
                      && r.y > 180 && r.y < 820;
                  };
                  const allLeaves = [...document.querySelectorAll('div,span,button,li,a')].filter(isLeaf)
                    .map(n => {
                      const r = n.getBoundingClientRect();
                      return {
                        n,
                        t: (n.textContent || '').trim(),
                        x: r.x + r.width / 2,
                        y: r.y + r.height / 2,
                        r,
                      };
                    });
                  const candidates = allLeaves.filter(x => x.t === wanted);
                  if (!candidates.length) return null;

                  const scored = [];
                  for (const c of candidates) {
                    // 同父兄弟
                    const sibs = allLeaves.filter(x => x.n.parentElement && x.n.parentElement === c.n.parentElement);
                    const near = allLeaves.filter(x =>
                      Math.abs(x.y - c.y) < 55 && Math.abs(x.x - c.x) < 520
                    );
                    const group = sibs.length >= 6 ? sibs : near;
                    const vals = group.map(x => x.t);
                    const uniq = new Set(vals);
                    if (isAttr && uniq.size < 2 && near.length < 2) continue;
                    if (!isAttr && uniq.size < 4 && near.length < 6) continue;
                    const numsAll = [...uniq].filter(v => /^[0-9]+$/.test(v)).map(Number);
                    const hezhiNums = numsAll.filter(n => n >= 0 && n <= 27);
                    const maxH = hezhiNums.length ? Math.max(...hezhiNums) : 0;
                    const minH = hezhiNums.length ? Math.min(...hezhiNums) : 0;
                    const maxN = numsAll.length ? Math.max(...numsAll) : 0;
                    let score = uniq.size * 6 + Math.min(near.length, 30);
                    if (c.y >= 420 && c.y <= 720) score += 50;
                    else if (c.y >= 350 && c.y <= 780) score += 20;
                    if (!isAttr && /^[0-9]+$/.test(wanted)) {
                      const w = Number(wanted);
                      const looksHezhi = hezhiNums.length >= 12 && maxH >= 18 && minH <= 1;
                      if (looksHezhi) score += 240;
                      else if (maxH >= 18 && maxH <= 27 && minH === 0) score += 140;
                      if (maxN <= 9 && w <= 9 && uniq.size >= 8) score += 70;
                      if (!looksHezhi && maxN > 30) score -= 80;
                      if (preferHezhi && !looksHezhi) score -= 100;
                      if (uniq.has(String(w - 1)) || uniq.has(String(w + 1))) score += 30;
                    }
                    if (isAttr) score += 60;
                    scored.push({ score, x: c.x, y: c.y, n: c.n });
                  }
                  scored.sort((a, b) => b.score - a.score);
                  if (!scored.length) return null;
                  // 和值：拒绝明显不像号池的候选
                  if (preferHezhi && scored[0].score < 120) return null;
                  scored[0].n.scrollIntoView({ block: 'center' });
                  const r2 = scored[0].n.getBoundingClientRect();
                  return { x: r2.x + r2.width / 2, y: r2.y + r2.height / 2, score: scored[0].score };
                }""",
                {"label": label, "kind": kind},
            )
            if rect:
                self.page.mouse.click(rect["x"], rect["y"])
                self.page.wait_for_timeout(200)
                return True
            self.page.wait_for_timeout(400)
        if kind == "hezhi":
            return False
        if self._click_text_near_play(label):
            return True
        return self._pick_digit_on_position("", label)

    def _fill_danshi(self, text: str) -> bool:
        page = self.page
        page.wait_for_timeout(500)
        for sel in ("textarea", "input[type='text']", "[contenteditable='true']"):
            loc = page.locator(sel)
            for i in range(min(loc.count(), 8)):
                el = loc.nth(i)
                try:
                    if not el.is_visible(timeout=800):
                        continue
                    box = el.bounding_box()
                    if not box or box["y"] < 120:
                        continue
                    el.click(timeout=2_000)
                    el.fill(text, timeout=3_000)
                    page.wait_for_timeout(300)
                    return True
                except Exception:
                    continue
        ok = page.evaluate(
            """(text) => {
              const tas = [...document.querySelectorAll('textarea')].filter(t => {
                const r = t.getBoundingClientRect();
                return r.width > 80 && r.height > 30 && r.y > 100;
              });
              const ta = tas[0];
              if (!ta) return false;
              ta.focus();
              ta.value = text;
              ta.dispatchEvent(new Event('input', {bubbles:true}));
              ta.dispatchEvent(new Event('change', {bubbles:true}));
              return true;
            }""",
            text,
        )
        return bool(ok)

    def _detect_position_labels(self) -> list[str]:
        order_full = [
            "万位",
            "千位",
            "百位",
            "十位",
            "个位",
            "第一位",
            "第二位",
            "第三位",
            "第四位",
            "第五位",
        ]
        body = self.page.locator("body").inner_text()
        found = [lab for lab in order_full if lab in body]
        if len(found) >= 3:
            return found
        short = self.page.evaluate(
            """() => {
              const order = ['万','千','百','十','个'];
              const hit = [];
              for (const lab of order) {
                const el = [...document.querySelectorAll('div,span,label,p,td')]
                  .find(e => {
                    const t = (e.innerText || '').trim();
                    if (t !== lab) return false;
                    const r = e.getBoundingClientRect();
                    return r.width > 0 && r.height > 0 && r.width < 60;
                  });
                if (el) hit.push(lab);
              }
              return hit;
            }"""
        )
        return list(short) if short else found

    def _pick_digit_on_position(self, pos: str, digit: str) -> bool:
        rect = self.page.evaluate(
            """({pos, digit}) => {
              const isDigitNode = (el) => {
                const t = (el.textContent || '').trim();
                return t === digit && (el.children.length === 0 || el.innerText.trim() === digit);
              };
              const clickRect = (el) => {
                const r = el.getBoundingClientRect();
                if (r.width <= 0 || r.height <= 0) return null;
                return {x: r.x + r.width / 2, y: r.y + r.height / 2};
              };
              if (!pos) {
                const nodes = [...document.querySelectorAll('button,div,span,a')].filter(isDigitNode);
                const hit = nodes.find(n => {
                  const r = n.getBoundingClientRect();
                  return r.width > 0 && r.height > 0 && r.width < 80;
                });
                return hit ? clickRect(hit) : null;
              }
              const aliases = pos.length === 1 ? [pos, pos + '位'] : [pos, pos.replace('位','')];
              const labels = [...document.querySelectorAll('div,span,label,p,td')]
                .filter(e => {
                  const t = (e.textContent || '').trim();
                  return aliases.includes(t) && ((e.children.length === 0) || e.innerText.trim() === t);
                });
              for (const lab of labels) {
                let row = lab.parentElement;
                for (let depth = 0; depth < 6 && row; depth++, row = row.parentElement) {
                  const nums = [...row.querySelectorAll('button,div,span,a')].filter(isDigitNode);
                  const uniq = new Map();
                  for (const n of nums) {
                    const t = (n.textContent || '').trim();
                    if (/^[0-9]$/.test(t) && !uniq.has(t)) uniq.set(t, n);
                  }
                  if (uniq.size >= 10 && uniq.has(digit)) return clickRect(uniq.get(digit));
                  if (nums.length && uniq.has(digit)) return clickRect(uniq.get(digit));
                }
              }
              return null;
            }""",
            {"pos": pos, "digit": str(digit)},
        )
        if not rect:
            return False
        self.page.mouse.click(rect["x"], rect["y"])
        return True

    def fetch_bet_records_raw(self) -> str:
        self.page.goto(f"{self.base_url}/user/betRecord", wait_until="domcontentloaded")
        self.page.wait_for_timeout(2_000)
        self.dismiss_dialogs()
        return self.page.locator("body").inner_text()

    def _collect_menu_labels(self) -> list[str]:
        """收集可见彩种菜单文案（优先下拉面板内）。"""
        labels = self.page.evaluate(
            """() => {
              const out = [];
              const seen = new Set();
              const push = (t) => {
                t = (t || '').trim();
                if (!t || t.includes('\\n')) return;
                t = t.replace(/\\s+/g, '');
                if (t.length > 1 && t.length <= 20 && !seen.has(t)) {
                  seen.add(t);
                  out.push(t);
                }
              };
              const isVisible = (el) => {
                const r = el.getBoundingClientRect();
                if (r.width <= 0 || r.height <= 0) return false;
                const st = window.getComputedStyle(el);
                return st.visibility !== 'hidden' && st.display !== 'none' && Number(st.opacity) !== 0;
              };
              // 优先：含「分分彩/波场」等的可见面板
              const panels = [...document.querySelectorAll('div')].filter(d => {
                if (!isVisible(d)) return false;
                const r = d.getBoundingClientRect();
                if (r.width < 200 || r.height < 80 || r.y > 600) return false;
                const t = d.innerText || '';
                return t.includes('分分彩') || (t.includes('波场') && t.includes('分彩'));
              });
              const roots = panels.length ? panels.slice(0, 3) : [];
              if (!roots.length) {
                // 退化为扫描可见短标签
                for (const el of document.querySelectorAll('span,div,a')) {
                  if (!isVisible(el)) continue;
                  if (el.children.length > 1) continue;
                  const t = (el.innerText || '').trim();
                  if (/彩|哈希|波场|币安|飞艇|快三|PK/.test(t)) push(t);
                  if (out.length >= 120) break;
                }
                return out;
              }
              for (const root of roots) {
                for (const el of root.querySelectorAll('span,div,a')) {
                  if (!isVisible(el)) continue;
                  if (el.children.length > 1) continue;
                  push((el.innerText || '').trim());
                  if (out.length >= 200) return out;
                }
              }
              return out;
            }"""
        )
        return list(labels or [])
