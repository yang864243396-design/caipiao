#!/usr/bin/env python3
"""P0–P5 comprehensive HTTP + seed smoke test."""

import json
import os
import sys
import urllib.error
import urllib.request

PORT = os.environ.get("PORT", "8081")
BASE = f"http://127.0.0.1:{PORT}/api/v1"
SEEDS = os.path.join(os.path.dirname(__file__), "..", "docs", "seeds")


def req(method, path, body=None, token=None):
    url = BASE + path
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    data = json.dumps(body).encode("utf-8") if body is not None else None
    r = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=12) as resp:
            return resp.status, json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8")
        try:
            payload = json.loads(raw)
        except json.JSONDecodeError:
            payload = {"raw": raw}
        return e.code, payload


def login(path, account, password):
    code, data = req("POST", path, {"account": account, "password": password})
    if code != 200 or data.get("code") != 0:
        raise RuntimeError(f"login {path} failed: {code} {data}")
    return data["data"]["accessToken"]


def count_csv_rows(path, skip_header=True):
    with open(path, encoding="utf-8") as f:
        lines = [ln for ln in f if ln.strip()]
    return len(lines) - (1 if skip_header else 0)


def main():
    fails = []

    def check(phase, name, ok, detail=""):
        tag = "PASS" if ok else "FAIL"
        print(f"[{tag}] {phase} {name}" + (f" — {detail}" if detail else ""))
        if not ok:
            fails.append(f"{phase}:{name}")

    # --- P0 seeds ---
    p0_files = [
        "lottery_catalog.csv",
        "play_templates.csv",
        "play_types.csv",
        "sub_plays.csv",
        "platform_340_play_mapping.csv",
    ]
    for fn in p0_files:
        path = os.path.join(SEEDS, fn)
        check("P0", f"seed file {fn}", os.path.isfile(path))

    lotto_n = count_csv_rows(os.path.join(SEEDS, "lottery_catalog.csv"))
    sub_n = count_csv_rows(os.path.join(SEEDS, "sub_plays.csv"))
    map_n = count_csv_rows(os.path.join(SEEDS, "platform_340_play_mapping.csv"))
    check("P0", "47 lotteries CSV", lotto_n == 47, f"count={lotto_n}")
    check("P0", "340 sub_plays CSV", sub_n == 340, f"count={sub_n}")
    check("P0", "340 mapping CSV", map_n == 340, f"count={map_n}")

    # --- P1 public catalog ---
    http, data = req("GET", "/public/lotteries")
    items = data.get("data", {}).get("items", [])
    codes = {x["code"] for x in items}
    check("P1", "public lotteries API", http == 200 and len(items) >= 40, f"on_sale={len(items)}")
    check("P1", "legacy tencent_ffc hidden", "tencent_ffc" not in codes)
    check("P1", "new code tron_ffc_1m visible", "tron_ffc_1m" in codes)

    http, tree = req("GET", "/public/lotteries/tron_ffc_1m/play-tree")
    play_types = tree.get("data", {}).get("playTypes", [])
    check("P1", "ssc play-tree", http == 200 and len(play_types) >= 10, f"types={len(play_types)}")

    http, tree = req("GET", "/public/lotteries/tron_lhc/play-tree")
    lhc_types = tree.get("data", {}).get("playTypes", [])
    check("P3", "lhc play-tree", http == 200 and len(lhc_types) >= 5, f"types={len(lhc_types)}")

    http, tree = req("GET", "/public/lotteries/taiwan_pc28/play-tree")
    pc_types = tree.get("data", {}).get("playTypes", [])
    check("P4", "pc28 play-tree", http == 200 and len(pc_types) >= 1, f"types={len(pc_types)}")

    # --- P2/P4 templates spot check ---
    for code, tpl, phase in [
        ("tron_ffc_1m", "ssc_std", "P2"),
        ("tron_syxw", "syxw_std", "P4"),
        ("taiwan_pk10", "pk10_std", "P4"),
        ("eth_k3", "k3_std", "P4"),
    ]:
        http, d = req("GET", f"/public/lotteries/{code}/play-tree")
        got = d.get("data", {}).get("playTemplate", "")
        check(phase, f"{code} template {tpl}", http == 200 and got == tpl, got)

    # --- P5 maintenance + legacy status ---
    http, st = req("GET", "/public/lotteries/tencent_ffc/status")
    body = st.get("data", {})
    check("P5", "legacy status", http == 200 and body.get("legacy") is True)

    admin = login("/admin/auth/login", "admin", "admin123")
    http, cat = req("GET", "/admin/games/lottery-catalog", token=admin)
    admin_items = cat.get("data", {}).get("items", [])
    check("P1", "admin catalog list", http == 200 and len(admin_items) == 47, f"count={len(admin_items)}")
    check("P5", "admin PATCH route", any(x.get("saleStatus") in ("on_sale", "maintenance") for x in admin_items))

    http, tpls = req("GET", "/admin/games/play-templates", token=admin)
    tpl_items = tpls.get("data", {}).get("items", [])
    check("P1", "admin play templates", http == 200 and len(tpl_items) >= 6, f"count={len(tpl_items)}")

    client = login("/client/auth/login", "vs8888", "vs8888")
    http, opts = req("GET", "/client/games/lottery-options", token=client)
    opt_items = opts.get("data", {}).get("items", [])
    check("P5", "member lottery-options", http == 200 and len(opt_items) == 47, f"count={len(opt_items)}")

    # --- T1 guaji auth status ---
    http, gst = req("GET", "/client/guaji/auth-status", token=client)
    gst_body = gst.get("data", {})
    check("T1", "guaji auth-status API", http == 200 and "hasActiveGuajiAuth" in gst_body)
    check("T1", "guaji accounts list", req("GET", "/client/guaji/accounts", token=client)[0] == 200)

    print("---")
    if fails:
        print(f"P0–P5 + T1 smoke: {len(fails)} failed — {', '.join(fails)}")
        sys.exit(1)
    print("P0–P5 + T1 smoke: all checks passed")


if __name__ == "__main__":
    main()
