"""诊断：前三直选和值号池结构与点选。"""
from __future__ import annotations

import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from playwright.sync_api import sync_playwright

from browser.v6_app import V6App
from credentials import load_credentials


def main() -> int:
    creds = load_credentials()
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        ctx, v6 = V6App.launch(browser, creds.v6_url)
        try:
            v6.login(creds.v6_user, creds.v6_pass)
            v6.open_lottery_by_platform_label("波场一分彩")
            v6.select_play("前三码", "前三直选和值")
            dump = v6.page.evaluate(
                """() => {
                  const isLeaf = (n) => {
                    const t = (n.textContent || '').trim();
                    if (!t) return false;
                    if (n.children.length > 0 && (n.innerText || '').trim() !== t) return false;
                    const r = n.getBoundingClientRect();
                    return r.width > 4 && r.width < 120 && r.height > 4 && r.height < 80
                      && r.y > 150 && r.y < 850;
                  };
                  const leaves = [...document.querySelectorAll('div,span,button,li,a,label')].filter(isLeaf);
                  const twelves = leaves.filter(n => (n.textContent||'').trim() === '12').map(n => {
                    const r = n.getBoundingClientRect();
                    const p = n.parentElement;
                    const sibs = p ? [...p.children].map(c => (c.textContent||'').trim()).slice(0, 40) : [];
                    return {
                      tag: n.tagName, cls: (n.className||'').toString().slice(0,80),
                      x: Math.round(r.x), y: Math.round(r.y), w: Math.round(r.width), h: Math.round(r.height),
                      sibs, parentTag: p ? p.tagName : '', parentCls: p ? (p.className||'').toString().slice(0,80) : ''
                    };
                  });
                  const nums = leaves.filter(n => /^[0-9]{1,2}$/.test((n.textContent||'').trim())).map(n => {
                    const r = n.getBoundingClientRect();
                    return { t: (n.textContent||'').trim(), y: Math.round(r.y), x: Math.round(r.x), w: Math.round(r.width) };
                  });
                  // 按 y 分行
                  const rows = {};
                  for (const n of nums) {
                    const key = String(Math.round(n.y / 20) * 20);
                    if (!rows[key]) rows[key] = [];
                    rows[key].push(n.t);
                  }
                  const rowSummary = Object.entries(rows).map(([y, vals]) => ({
                    y: Number(y), n: new Set(vals).size, sample: [...new Set(vals)].sort((a,b)=>Number(a)-Number(b)).slice(0,30)
                  })).sort((a,b)=>a.y-b.y);
                  return {
                    readyHint: document.body.innerText.includes('直选和值'),
                    twelves,
                    rowSummary: rowSummary.slice(0, 25),
                    bodyHas27: /\\b27\\b/.test(document.body.innerText),
                    hezhiReady: false
                  };
                }"""
            )
            dump["hezhiReady"] = v6._hezhi_pool_ready()
            path = Path("reports/v6_hezhi_structure.json")
            path.write_text(json.dumps(dump, ensure_ascii=False, indent=2), encoding="utf-8")
            print(json.dumps({
                "ready": dump["hezhiReady"],
                "hint": dump["readyHint"],
                "twelves": len(dump["twelves"]),
                "rows": dump["rowSummary"][:8],
                "first12": dump["twelves"][:3],
            }, ensure_ascii=False, indent=2), flush=True)

            # 尝试多种点选
            ok = v6._pick_pool_label("12", kind="hezhi")
            print("pick_pool_label", ok, flush=True)
            try:
                n = v6.read_preview_bet_count()
            except Exception as e:
                n = f"err:{e}"
            print("bet", n, flush=True)
            if not ok or (isinstance(n, int) and n <= 1):
                # 直接点第一个 12
                clicked = v6.page.evaluate(
                    """() => {
                      const nodes = [...document.querySelectorAll('div,span,button,li,a')];
                      for (const e of nodes) {
                        if ((e.textContent||'').trim() !== '12') continue;
                        if (e.children.length > 0 && (e.innerText||'').trim() !== '12') continue;
                        const r = e.getBoundingClientRect();
                        if (r.y < 400 || r.y > 750 || r.width < 8 || r.width > 80) continue;
                        e.click();
                        return {x:r.x, y:r.y, w:r.width};
                      }
                      return null;
                    }"""
                )
                print("direct_click", clicked, flush=True)
                v6.page.wait_for_timeout(500)
                try:
                    print("bet2", v6.read_preview_bet_count(), flush=True)
                except Exception as e:
                    print("bet2_err", e, flush=True)
            return 0
        finally:
            ctx.close()
            browser.close()


if __name__ == "__main__":
    raise SystemExit(main())
