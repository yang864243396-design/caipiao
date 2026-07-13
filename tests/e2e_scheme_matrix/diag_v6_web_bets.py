"""一次性诊断：已登录态下拉取 /api/web_bets/ 看返回结构。"""
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
    storage = Path("reports/v6_storage.json")
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        ctx, v6 = V6App.launch(
            browser,
            creds.v6_url,
            storage_state=storage if storage.is_file() else None,
        )
        v6.login(creds.v6_user, creds.v6_pass)
        page = v6.page
        page.goto(f"{creds.v6_url}/user/betRecord", wait_until="domcontentloaded")
        page.wait_for_timeout(2000)
        storage_dump = page.evaluate(
            """() => {
              const out = { local: {}, session: {} };
              try {
                for (let i = 0; i < localStorage.length; i++) {
                  const k = localStorage.key(i);
                  const v = localStorage.getItem(k) || '';
                  out.local[k] = v.length > 120 ? v.slice(0, 120) + '…' : v;
                }
              } catch (e) {}
              try {
                for (let i = 0; i < sessionStorage.length; i++) {
                  const k = sessionStorage.key(i);
                  const v = sessionStorage.getItem(k) || '';
                  out.session[k] = v.length > 120 ? v.slice(0, 120) + '…' : v;
                }
              } catch (e) {}
              return out;
            }"""
        )
        Path("reports/v6_storage_keys.json").write_text(
            json.dumps(storage_dump, ensure_ascii=False, indent=2), encoding="utf-8"
        )
        probe = page.evaluate(
            """async (base) => {
              const origin = base || location.origin;
              const tryKeys = [];
              try {
                for (let i = 0; i < localStorage.length; i++) tryKeys.push('L:' + localStorage.key(i));
                for (let i = 0; i < sessionStorage.length; i++) tryKeys.push('S:' + sessionStorage.key(i));
              } catch (e) {}
              const tokenCandidates = [];
              const scan = (store, prefix) => {
                try {
                  for (let i = 0; i < store.length; i++) {
                    const k = store.key(i);
                    const v = store.getItem(k) || '';
                    if (/token|access|auth|jwt/i.test(k) || /Bearer|^eyJ|[a-f0-9]{20,}/i.test(v)) {
                      tokenCandidates.push({ from: prefix + k, sample: v.slice(0, 40) });
                    }
                    // nested JSON
                    if (v.startsWith('{') && /token|access/i.test(v)) {
                      try {
                        const o = JSON.parse(v);
                        const walk = (obj, path) => {
                          if (!obj || typeof obj !== 'object') return;
                          for (const [kk, vv] of Object.entries(obj)) {
                            if (typeof vv === 'string' && vv.length > 20 && /token|access|jwt/i.test(kk + vv.slice(0,8))) {
                              tokenCandidates.push({ from: prefix + path + '.' + kk, sample: vv.slice(0, 40) });
                            } else if (vv && typeof vv === 'object') walk(vv, path + '.' + kk);
                          }
                        };
                        walk(o, k);
                      } catch (e) {}
                    }
                  }
                } catch (e) {}
              };
              scan(localStorage, 'L:');
              scan(sessionStorage, 'S:');
              const url = origin + '/api/web_bets/?limit=5&page=1';
              const cookieOnly = await fetch(url, { credentials: 'include' });
              let cookieBody = '';
              try { cookieBody = await cookieOnly.text(); } catch (e) { cookieBody = String(e); }
              const results = [{
                mode: 'cookie',
                status: cookieOnly.status,
                body: cookieBody.slice(0, 400),
              }];
              for (const t of tokenCandidates.slice(0, 5)) {
                const headers = { Accept: 'application/json', Authorization: 'Bearer ' + (t.sample.length >= 40 ? '' : '') };
              }
              // try common header patterns with full tokens from storage
              const fullTokens = [];
              const collect = (store) => {
                try {
                  for (let i = 0; i < store.length; i++) {
                    const k = store.key(i);
                    let v = store.getItem(k) || '';
                    if (v.startsWith('{')) {
                      try {
                        const o = JSON.parse(v);
                        const flat = JSON.stringify(o);
                        const m = flat.match(/"(access_token|accessToken|token|Authorization)"\\s*:\\s*"([^"]+)"/i);
                        if (m) fullTokens.push(m[2]);
                      } catch (e) {}
                    } else if (/^[A-Za-z0-9._\\-]{20,}$/.test(v) && /token|access|auth/i.test(k)) {
                      fullTokens.push(v);
                    }
                  }
                } catch (e) {}
              };
              collect(localStorage);
              collect(sessionStorage);
              for (const tok of [...new Set(fullTokens)].slice(0, 4)) {
                const resp = await fetch(url, {
                  credentials: 'include',
                  headers: { Accept: 'application/json', Authorization: 'Bearer ' + tok },
                });
                let body = '';
                try { body = await resp.text(); } catch (e) { body = String(e); }
                results.push({ mode: 'bearer', status: resp.status, tok: tok.slice(0, 24), body: body.slice(0, 400) });
              }
              return { tryKeys, tokenCandidates, results };
            }""",
            creds.v6_url,
        )
        Path("reports/v6_web_bets_probe.json").write_text(
            json.dumps(probe, ensure_ascii=False, indent=2), encoding="utf-8"
        )
        print(json.dumps(probe, ensure_ascii=False, indent=2)[:2000])
        ctx.close()
        browser.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
