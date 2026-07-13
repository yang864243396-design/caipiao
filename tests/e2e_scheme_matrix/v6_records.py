"""第三方投注记录：用页面 JWT 主动分页拉取 /api/web_bets/。"""
from __future__ import annotations

import re
from typing import Any

from browser.v6_app import V6App
from models import BetRecord


def fetch_v6_bet_records(
    v6: V6App,
    *,
    lottery_hint: str = "",
    scheme_hint: str = "",
    limit: int = 400,
    game_id: int | None = None,
    want_periods: set[str] | None = None,
) -> list[BetRecord]:
    """
    打开投注记录页后，读取 localStorage.state.user.token，
    带 Authorization 分页拉取 /api/web_bets/（仅 cookie 会 401）。
    """
    page = v6.page
    captured: list[dict[str, Any]] = []

    def on_resp(resp) -> None:
        try:
            if "web_bets" not in resp.url or resp.request.method != "GET":
                return
            if "/lott" in resp.url:
                return
            data = resp.json()
            items = data.get("data")
            if isinstance(items, list):
                captured.extend(items)
        except Exception:
            pass

    page.on("response", on_resp)
    try:
        raw = v6.fetch_bet_records_raw()
        page.wait_for_timeout(500)
        per_page = 50
        max_pages = max(12, (limit + per_page - 1) // per_page)
        if want_periods:
            max_pages = max(max_pages, 24)
        try:
            pulled = page.evaluate(
                """async ({ perPage, maxPages, base }) => {
                  const origin = base || location.origin;
                  let token = '';
                  try {
                    const st = JSON.parse(localStorage.getItem('state') || '{}');
                    token = (st && st.user && st.user.token) || '';
                  } catch (e) {}
                  if (!token) {
                    try {
                      for (let i = 0; i < localStorage.length; i++) {
                        const k = localStorage.key(i);
                        const v = localStorage.getItem(k) || '';
                        if (!v.startsWith('{')) continue;
                        const m = v.match(/"token"\\s*:\\s*"(eyJ[^"]+)"/);
                        if (m) { token = m[1]; break; }
                      }
                    } catch (e) {}
                  }
                  const headers = { Accept: 'application/json' };
                  if (token) headers['Authorization'] = 'Bearer ' + token;
                  const all = [];
                  let lastStatus = 0;
                  let lastBody = '';
                  for (let p = 1; p <= maxPages; p++) {
                    const url = origin + '/api/web_bets/?limit=' + perPage + '&page=' + p;
                    let resp;
                    try {
                      resp = await fetch(url, { credentials: 'include', headers });
                    } catch (e) {
                      return { items: all, error: String(e), status: lastStatus, hasToken: !!token };
                    }
                    lastStatus = resp.status;
                    if (!resp.ok) {
                      try { lastBody = await resp.text(); } catch (e) {}
                      return {
                        items: all,
                        error: 'http ' + resp.status,
                        status: resp.status,
                        body: (lastBody || '').slice(0, 200),
                        hasToken: !!token,
                      };
                    }
                    let data;
                    try { data = await resp.json(); } catch (e) {
                      return { items: all, error: 'json', status: lastStatus, hasToken: !!token };
                    }
                    const items = Array.isArray(data && data.data) ? data.data : [];
                    if (!items.length) break;
                    all.push(...items);
                    // 第三方常忽略 limit（实测单页约 20~24），勿因 <perPage 提前结束
                  }
                  return { items: all, status: lastStatus, hasToken: !!token };
                }""",
                {"perPage": per_page, "maxPages": max_pages, "base": v6.base_url},
            )
            items = []
            if isinstance(pulled, dict):
                items = pulled.get("items") or []
                print(
                    f"[v6] web_bets pull n={len(items)} status={pulled.get('status')} "
                    f"token={pulled.get('hasToken')} err={pulled.get('error') or ''}",
                    flush=True,
                )
                if pulled.get("body"):
                    print(f"[v6] web_bets body≈{pulled.get('body')!r}", flush=True)
            elif isinstance(pulled, list):
                items = pulled
            if items:
                captured.extend(items)
        except Exception as e:
            print(f"[v6] web_bets pull failed: {e}", flush=True)
    finally:
        try:
            page.remove_listener("response", on_resp)
        except Exception:
            pass

    rows = [_record_from_api(item) for item in captured]
    rows = [r for r in rows if r and r.period]
    seen: set[str] = set()
    uniq: list[BetRecord] = []
    for r in rows:
        k = f"{r.period}|{r.amount}|{r.content}|{r.multiplier}|{r.bet_count}|{r.payout}"
        if k in seen:
            continue
        seen.add(k)
        uniq.append(r)
    rows = uniq

    _ = game_id
    if lottery_hint:
        filtered = [
            r
            for r in rows
            if not r.lottery_label
            or lottery_hint in r.lottery_label
            or r.lottery_label in lottery_hint
            or "一分彩" in (r.lottery_label or "")
            or "1分彩" in (r.lottery_label or "")
        ]
        if filtered:
            rows = filtered

    if want_periods:
        hit = {r.period for r in rows} & want_periods
        print(
            f"[v6] records={len(rows)} want_periods={len(want_periods)} hit={len(hit)}",
            flush=True,
        )
    else:
        print(f"[v6] records={len(rows)}", flush=True)

    _ = scheme_hint
    _ = raw
    out_limit = max(limit, len(want_periods or ()) * 5, 200)
    return rows[:out_limit]


def _record_from_api(item: dict[str, Any]) -> BetRecord | None:
    period = str(item.get("periods") or "").strip()
    if not period:
        return None
    nested = item.get("bet_content") or {}
    inner = nested.get("bet_content") if isinstance(nested, dict) else None
    if not isinstance(inner, dict):
        contents = item.get("bet_contents")
        if isinstance(contents, list) and contents and isinstance(contents[0], dict):
            inner = contents[0]
        elif isinstance(nested, dict) and (
            "bets_nums" in nested or "multiple" in nested or "bet_content" in nested
        ):
            inner = nested
        else:
            inner = {}
    content = str(inner.get("bet_content") or "").strip()
    bets_nums = inner.get("bets_nums")
    try:
        bet_count = int(bets_nums) if bets_nums is not None else None
    except (TypeError, ValueError):
        bet_count = None
    try:
        amount = float(item.get("bet_amount")) if item.get("bet_amount") is not None else None
    except (TypeError, ValueError):
        amount = None
    payout: float | None = None
    try:
        net = float(item["net_amount"]) if item.get("net_amount") is not None else None
    except (TypeError, ValueError):
        net = None
    try:
        gross = float(item["payout_amount"]) if item.get("payout_amount") is not None else None
    except (TypeError, ValueError):
        gross = None
    if net is not None and net > 0:
        payout = net
    elif gross is not None and gross > 0:
        if amount is not None and gross > amount:
            payout = round(gross - amount, 4)
        else:
            payout = gross
    elif net is not None:
        payout = 0.0 if net <= 0 else net
    else:
        payout = None
    mult = inner.get("multiple")
    try:
        multiplier = float(mult) if mult is not None else None
    except (TypeError, ValueError):
        multiplier = None

    note = str(item.get("note") or "")
    win_status = ""
    if "不中奖" in note or "未中奖" in note:
        win_status = "挂"
    elif "中奖" in note and "不中" not in note:
        win_status = "中"
    elif item.get("settled"):
        if payout and payout > 0:
            win_status = "中"
        else:
            win_status = "挂"

    draw_numbers = ""
    m = re.search(r"开奖结果:\s*<span[^>]*>([^<]+)</span>(\d*)", note)
    if m:
        draw_numbers = (m.group(1) + m.group(2)).strip()

    return BetRecord(
        lottery_label=str(item.get("game_name") or "").strip(),
        period=period,
        content=content,
        bet_count=bet_count,
        amount=amount,
        win_status=win_status,
        payout=payout,
        draw_numbers=draw_numbers,
        multiplier=multiplier,
    )


def normalize_ssc_fushi_content(content: str) -> str:
    text = (content or "").replace("\r\n", "\n").strip()
    if not text:
        return ""
    if "\n" in text:
        parts = []
        for line in text.split("\n"):
            digits = re.sub(r"[^0-9]", "", line)
            parts.append(digits)
        return ",".join(parts)
    return text
