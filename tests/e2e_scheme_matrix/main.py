"""CLI 入口。"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

# 保证以脚本方式运行时可导入同目录模块
sys.path.insert(0, str(Path(__file__).resolve().parent))

from runner import Runner, install_signal_handlers
from credentials import load_credentials


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    p = argparse.ArgumentParser(description="自创方案全矩阵 E2E（见 README.md）")
    p.add_argument("--smoke", action="store_true", help="冒烟：1组合完整链路（建案/注数/开启/跑期/对账/暂停）")
    p.add_argument("--resume", action="store_true", help="断点续跑：跳过 record_compare=passed")
    p.add_argument("--headed", action="store_true", default=True, help="有头浏览器（默认）")
    p.add_argument("--headless", action="store_true", help="无头（覆盖默认有头）")
    return p.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    headed = not args.headless
    creds = load_credentials()
    runner = Runner(
        creds,
        smoke=args.smoke,
        resume=args.resume,
        headed=headed,
    )
    install_signal_handlers(runner)
    return runner.run()


if __name__ == "__main__":
    raise SystemExit(main())
