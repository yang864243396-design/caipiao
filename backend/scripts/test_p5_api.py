#!/usr/bin/env python3
"""P5 HTTP API smoke test against local server."""

import json
import os
import sys
import urllib.error
import urllib.request

_port = os.environ.get("PORT", "8081")
BASE = f"http://127.0.0.1:{_port}/api/v1"
TEST_CODE = "taiwan_pc28"


def req(method, path, body=None, token=None):
    url = BASE + path
    data = None
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if body is not None:
        data = json.dumps(body).encode("utf-8")
    r = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=8) as resp:
            return resp.status, json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8")
        try:
            payload = json.loads(raw)
        except json.JSONDecodeError:
            payload = {"raw": raw}
        return e.code, payload


def login_admin():
    code, data = req("POST", "/admin/auth/login", {"account": "admin", "password": "admin123"})
    if code != 200 or data.get("code") != 0:
        raise RuntimeError(f"admin login failed: {code} {data}")
    return data["data"]["accessToken"]


def login_client():
    code, data = req("POST", "/client/auth/login", {"account": "vs8888", "password": "vs8888"})
    if code != 200 or data.get("code") != 0:
        raise RuntimeError(f"client login failed: {code} {data}")
    return data["data"]["accessToken"]


def main():
    fails = []

    def check(name, ok, detail=""):
        status = "PASS" if ok else "FAIL"
        print(f"[{status}] {name}" + (f" — {detail}" if detail else ""))
        if not ok:
            fails.append(name)

    # legacy / invalid / on_sale status
    for code, expect in [
        ("tencent_ffc", {"legacy": True, "exists": False}),
        ("no_such_lottery_x", {"legacy": False, "exists": False}),
        ("tron_ffc_1m", {"legacy": False, "exists": True, "saleStatus": "on_sale"}),
    ]:
        http, data = req("GET", f"/public/lotteries/{code}/status")
        body = data.get("data", data)
        ok = http == 200
        for k, v in expect.items():
            ok = ok and body.get(k) == v
        check(f"GET status {code}", ok, str(body))

    admin_token = login_admin()
    client_token = login_client()

    # snapshot original
    http, data = req("GET", "/admin/games/lottery-catalog", token=admin_token)
    items = data.get("data", {}).get("items", [])
    orig = next((x for x in items if x["code"] == TEST_CODE), None)
    if not orig:
        print(f"FAIL: {TEST_CODE} not in catalog")
        sys.exit(1)

    def restore():
        req(
            "PATCH",
            f"/admin/games/lottery-catalog/{TEST_CODE}",
            {
                "displayName": orig["displayName"],
                "outboundLotteryCode": orig.get("outboundLotteryCode") or TEST_CODE,
                "sortOrder": orig["sortOrder"],
                "saleStatus": "on_sale",
            },
            admin_token,
        )

    try:
        # enter maintenance
        http, data = req(
            "PATCH",
            f"/admin/games/lottery-catalog/{TEST_CODE}",
            {"enterMaintenance": True, "saleStatus": "maintenance"},
            admin_token,
        )
        row = data.get("data", {})
        check("PATCH enter maintenance", http == 200 and row.get("saleStatus") == "maintenance", str(data))

        http, data = req("GET", f"/public/lotteries/{TEST_CODE}/play-tree")
        check("play-tree blocked", http == 403, f"http={http}")

        http, data = req("GET", f"/client/games/{TEST_CODE}/detail", token=client_token)
        check("detail blocked", http == 403, f"http={http} body={data}")

        http, data = req(
            "POST",
            f"/client/games/{TEST_CODE}/bets",
            {
                "amount": 1,
                "multiplier": 1,
                "betPayload": {
                    "playTemplate": "pc28_std",
                    "typeId": "pc28_20",
                    "subId": "dxds",
                    "betMode": "dxds",
                    "groupContent": "大",
                },
            },
            client_token,
        )
        check("place-bet blocked", http == 403, f"http={http} body={data}")

        http, data = req("GET", "/client/games/lottery-options", token=client_token)
        items = data.get("data", {}).get("items", [])
        maint_row = next((x for x in items if x.get("code") == TEST_CODE), None)
        check(
            "lottery-options includes maintenance",
            http == 200 and maint_row is not None and maint_row.get("saleStatus") == "maintenance",
            str(maint_row),
        )

        http, data = req("GET", f"/public/lotteries/{TEST_CODE}/status")
        body = data.get("data", {})
        check("status maintenance", body.get("saleStatus") == "maintenance", str(body))

        http, data = req(
            "PATCH",
            f"/admin/games/lottery-catalog/{TEST_CODE}",
            {
                "displayName": "台湾28-P5API",
                "outboundLotteryCode": "taiwan_pc28_p5",
                "sortOrder": orig["sortOrder"],
                "saleStatus": "maintenance",
            },
            admin_token,
        )
        row = data.get("data", {})
        check(
            "PATCH maintenance fields",
            http == 200 and row.get("displayName") == "台湾28-P5API",
            str(data),
        )

        http, data = req("GET", "/public/lotteries")
        items = data.get("data", {}).get("items", [])
        hidden = all(x["code"] != TEST_CODE for x in items)
        check("public list hides maintenance", hidden)

    finally:
        restore()

    http, data = req("GET", f"/public/lotteries/{TEST_CODE}/play-tree")
    check("play-tree restored", http == 200, f"http={http}")

    print("---")
    if fails:
        print(f"P5 API: {len(fails)} failed — {', '.join(fails)}")
        sys.exit(1)
    print("P5 API: all checks passed")


if __name__ == "__main__":
    main()
