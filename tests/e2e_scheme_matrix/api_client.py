"""本平台 API：彩种、云端启停、投注记录。"""
from __future__ import annotations

from typing import Any

import httpx

from models import BetRecord


def _unwrap(data: Any) -> Any:
    if isinstance(data, dict) and "data" in data and ("code" in data or "success" in data):
        return data.get("data")
    return data


class PlatformApiClient:
    def __init__(self, base_url: str, timeout: float = 30.0) -> None:
        self.base_url = base_url.rstrip("/")
        self._client = httpx.Client(
            base_url=self.base_url,
            timeout=timeout,
            follow_redirects=True,
            trust_env=False,  # 避免系统 HTTP(S)_PROXY 劫持内网 192.168.*
        )

    def close(self) -> None:
        self._client.close()

    def __enter__(self) -> PlatformApiClient:
        return self

    def __exit__(self, *args: object) -> None:
        self.close()

    def set_bearer(self, token: str) -> None:
        token = (token or "").strip()
        if token:
            self._client.headers["Authorization"] = f"Bearer {token}"

    def import_cookies_from_playwright(self, cookies: list[dict[str, Any]]) -> None:
        for c in cookies:
            self._client.cookies.set(
                c["name"],
                c["value"],
                domain=c.get("domain") or None,
                path=c.get("path") or "/",
            )

    def get_json(self, path: str, **params: Any) -> Any:
        r = self._client.get(path, params={k: v for k, v in params.items() if v is not None} or None)
        if r.status_code == 404:
            return None
        r.raise_for_status()
        return _unwrap(r.json())

    def post_json(self, path: str, body: dict | None = None) -> Any:
        r = self._client.post(path, json=body or {})
        r.raise_for_status()
        return _unwrap(r.json())

    def fetch_lotteries(self) -> list[dict[str, Any]]:
        data = self.get_json("/public/lotteries")
        if isinstance(data, list):
            return data
        if isinstance(data, dict):
            for key in ("items", "data", "lotteries"):
                if isinstance(data.get(key), list):
                    return data[key]
        return []

    def lottery_interval_by_display_name(self) -> dict[str, str]:
        out: dict[str, str] = {}
        for row in self.fetch_lotteries():
            name = str(row.get("displayName") or row.get("name") or "").strip()
            interval = str(row.get("drawInterval") or row.get("draw_interval") or "").strip()
            if name:
                out[name] = interval
        return out

    def fetch_running_schemes(self, run_mode: str = "real") -> list[dict[str, Any]]:
        data = self.get_json("/client/cloud/schemes/running", runMode=run_mode, limit=0)
        if isinstance(data, dict) and isinstance(data.get("items"), list):
            return data["items"]
        if isinstance(data, list):
            return data
        return []

    def find_instance_by_name(self, scheme_name: str, run_mode: str = "real") -> dict[str, Any] | None:
        name = scheme_name.strip()
        for row in self.fetch_running_schemes(run_mode=run_mode):
            if str(row.get("schemeName") or "").strip() == name:
                return row
        return None

    def start_instance(self, instance_id: str) -> dict[str, Any]:
        return self.post_json(f"/client/cloud/instances/{instance_id}/start")

    def stop_instance(self, instance_id: str) -> dict[str, Any]:
        return self.post_json(f"/client/cloud/instances/{instance_id}/stop")

    def get_instance(self, instance_id: str) -> dict[str, Any] | None:
        for row in self.fetch_running_schemes(run_mode="real"):
            if str(row.get("id") or "") == instance_id:
                return row
        for row in self.fetch_running_schemes(run_mode="sim"):
            if str(row.get("id") or "") == instance_id:
                return row
        return None

    def fetch_scheme_bet_records(
        self,
        scheme_id: str,
        *,
        mode: str = "real",
        days: int = 3,
        limit: int = 100,
    ) -> list[BetRecord]:
        """分页拉取方案投注记录，映射为 BetRecord。"""
        items: list[dict[str, Any]] = []
        cursor: str | None = None
        while True:
            params: dict[str, Any] = {"mode": mode, "days": days, "limit": limit}
            if cursor:
                params["cursor"] = cursor
            data = self.get_json(f"/client/cloud/bet-records/{scheme_id}", **params)
            if data is None:
                break
            page = data.get("records") if isinstance(data, dict) else None
            if not isinstance(page, dict):
                break
            batch = page.get("items") or []
            items.extend(batch)
            meta = page.get("page") or {}
            if not meta.get("hasMore"):
                break
            cursor = meta.get("nextCursor")
            if not cursor:
                break
            if len(items) >= 500:
                break

        out: list[BetRecord] = []
        for it in items:
            period = str(it.get("periods") or it.get("period") or "").strip()
            status = str(it.get("status") or "")
            status_label = {
                "hit": "中",
                "won": "中",
                "miss": "挂",
                "lost": "挂",
                "pending": "待开奖",
                "cancelled": "已撤单",
            }.get(status, status)
            pnl = it.get("pnl")
            try:
                pnl_f = float(pnl) if pnl is not None else None
            except (TypeError, ValueError):
                pnl_f = None
            payout = pnl_f if pnl_f is not None and pnl_f > 0 else 0.0
            mult_raw = str(it.get("multiplier") or "1")
            try:
                mult = float(mult_raw.split("/")[0] if "/" in mult_raw else mult_raw)
            except ValueError:
                mult = None
            amount = it.get("amount")
            try:
                amount_f = float(amount) if amount is not None else None
            except (TypeError, ValueError):
                amount_f = None
            out.append(
                BetRecord(
                    lottery_label="",
                    period=period,
                    content=str(it.get("betContent") or "").strip(),
                    bet_count=None,
                    amount=amount_f,
                    win_status=status_label,
                    payout=payout,
                    draw_numbers="",  # 本端明细接口无开奖号字段
                    multiplier=mult,
                )
            )
        return out
