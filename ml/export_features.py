#!/usr/bin/env python3
from __future__ import annotations

import argparse
import sys


def main() -> int:
    parser = argparse.ArgumentParser(description="Export fraud ML training features from ClickHouse")
    parser.add_argument("--output", default="-", help="output path or - for stdout")
    parser.add_argument("--since", default="", help="ISO-8601 lower bound on event time")
    parser.add_argument("--until", default="", help="ISO-8601 upper bound on event time")
    args = parser.parse_args()

    _ = args
    print("export_features: 0 rows", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
