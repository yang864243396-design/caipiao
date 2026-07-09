#!/usr/bin/env python3
"""Parse Word-exported HTML .doc for API integration plan inputs."""
import html
import quopri
import re
import sys
from pathlib import Path


def decode_qp_text(raw: str) -> str:
    # normalize broken quoted-printable from mso exports
    raw = raw.replace("=\n", "").replace("=3D", "=")
    try:
        return quopri.decodestring(raw.encode("latin-1", errors="ignore")).decode("utf-8", errors="ignore")
    except Exception:
        return raw


def strip_html(text: str) -> str:
    text = re.sub(r"(?is)<(script|style).*?>.*?</\1>", " ", text)
    text = re.sub(r"(?is)<br\s*/?>", "\n", text)
    text = re.sub(r"(?is)</p>", "\n", text)
    text = re.sub(r"(?is)<[^>]+>", " ", text)
    text = html.unescape(text)
    text = re.sub(r"[ \t\u00a0]+", " ", text)
    text = re.sub(r"\n{3,}", "\n\n", text)
    return text.strip()


def main() -> None:
    src = Path(sys.argv[1])
    dst = Path(sys.argv[2])
    raw = src.read_bytes()
    text = raw.decode("utf-8", errors="ignore")
    if "<html" not in text.lower():
        text = decode_qp_text(text)
    plain = strip_html(text)

    # pull high-signal blocks
    lines = []
    for ln in plain.splitlines():
        ln = ln.strip()
        if not ln:
            continue
        if any(k in ln for k in (
            "http", "curl", "API", "api", "接口", "登录", "投注", "开奖", "余额",
            "短信", "邮件", "挂机", "彩种", "期号", "赔率", "token", "Token",
            "security", "lottery", "bet", "draw", "member", "user",
        )):
            lines.append(ln)

    # also extract curl blocks
    curls = re.findall(r"curl\s+[^\\]+(?:\\[^\\]+)*", plain, flags=re.I)
    urls = sorted(set(re.findall(r"https?://[A-Za-z0-9_./?=&%-]+", plain)))

    out = []
    out.append("# Extracted API signals\n")
    out.append("## URLs\n")
    for u in urls:
        out.append(f"- {u}")
    out.append("\n## curl samples\n")
    for c in curls[:40]:
        out.append(c.replace("\\", "\n  "))
        out.append("")
    out.append("\n## Filtered lines\n")
    out.extend(lines[:300])

    dst.write_text("\n".join(out), encoding="utf-8")
    print(f"urls={len(urls)} curls={len(curls)} lines={len(lines)} -> {dst}")


if __name__ == "__main__":
    main()
