"""从 .env 读取凭证。"""
from __future__ import annotations

import os
from dataclasses import dataclass
from urllib.parse import urlparse

from dotenv import load_dotenv

from config import ENV_PATH


@dataclass(frozen=True)
class Credentials:
    platform_url: str
    platform_api_base: str
    platform_user: str
    platform_pass: str
    v6_url: str
    v6_user: str
    v6_pass: str


def _default_api_base(platform_url: str) -> str:
    u = urlparse(platform_url)
    host = u.hostname or "127.0.0.1"
    scheme = u.scheme or "http"
    return f"{scheme}://{host}:8080/api/v1"


def load_credentials() -> Credentials:
    load_dotenv(ENV_PATH)
    missing = [
        k
        for k in (
            "PLATFORM_URL",
            "PLATFORM_USER",
            "PLATFORM_PASS",
            "V6_URL",
            "V6_USER",
            "V6_PASS",
        )
        if not (os.getenv(k) or "").strip()
    ]
    if missing:
        raise SystemExit(
            f"缺少环境变量 {', '.join(missing)}，请复制 .env.example 为 .env 并填写。"
        )
    platform_url = os.environ["PLATFORM_URL"].rstrip("/")
    api_base = (os.getenv("PLATFORM_API_BASE") or "").strip().rstrip("/")
    if not api_base:
        api_base = _default_api_base(platform_url)
    return Credentials(
        platform_url=platform_url,
        platform_api_base=api_base,
        platform_user=os.environ["PLATFORM_USER"].strip(),
        platform_pass=os.environ["PLATFORM_PASS"],
        v6_url=os.environ["V6_URL"].rstrip("/"),
        v6_user=os.environ["V6_USER"].strip(),
        v6_pass=os.environ["V6_PASS"],
    )
