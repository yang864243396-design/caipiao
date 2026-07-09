#!/usr/bin/env python3
"""Best-effort extract readable text from legacy .doc (OLE/binary)."""
import re
import sys
from pathlib import Path


def extract(path: Path) -> list[str]:
    data = path.read_bytes()
    candidates: list[str] = []

    for enc in ("utf-16-le", "gbk", "utf-8"):
        try:
            text = data.decode(enc, errors="ignore")
        except Exception:
            continue
        for part in re.split(r"[\x00\r\n]+", text):
            part = part.strip()
            if len(part) < 4:
                continue
            if re.search(r"[\u4e00-\u9fff]{2,}", part):
                candidates.append(part)
            elif re.search(r"https?://", part, re.I):
                candidates.append(part)
            elif re.search(r"(?i)(api|token|login|bet|draw|sms|mail|email|挂机|接口|登录|投注|开奖|余额|短信|邮件)", part):
                candidates.append(part)

    # dedupe preserving order
    seen = set()
    out: list[str] = []
    for line in candidates:
        key = line[:120]
        if key in seen:
            continue
        seen.add(key)
        out.append(line)
    return out


def main() -> None:
    src = Path(sys.argv[1])
    dst = Path(sys.argv[2]) if len(sys.argv) > 2 else src.with_suffix(".extracted.txt")
    lines = extract(src)
    dst.write_text("\n".join(lines), encoding="utf-8")
    print(f"extracted {len(lines)} lines -> {dst}")


if __name__ == "__main__":
    main()
