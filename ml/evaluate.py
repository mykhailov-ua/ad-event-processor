#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from feature_spec import FEATURE_DIMS, row_to_vector


def main() -> int:
    parser = argparse.ArgumentParser(description="Evaluate fraud model artifacts against labeled fixtures")
    parser.add_argument("--model", default="internal/fraudscoring/testdata/model.txt")
    parser.add_argument("--fixtures", default="testdata/ml")
    parser.add_argument("--metrics-out", default="", help="write JSON metrics to this path")
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parent.parent
    fixtures_dir = repo_root / args.fixtures
    if not fixtures_dir.is_dir():
        print(f"evaluate: fixtures dir missing: {fixtures_dir}", file=sys.stderr)
        return 1

    rows = 0
    for path in sorted(fixtures_dir.glob("features_*.json")):
        with open(path, encoding="utf-8") as handle:
            payload = json.load(handle)
        row = payload.get("row", {})
        expected = payload.get("vector")
        if expected is None:
            continue
        actual = row_to_vector(row)
        if len(actual) != FEATURE_DIMS:
            print(f"evaluate: {path.name}: vector dims {len(actual)} != {FEATURE_DIMS}", file=sys.stderr)
            return 1
        rows += 1

    metrics = {"rows": rows, "status": "skeleton"}
    if args.metrics_out:
        out = repo_root / args.metrics_out
        out.parent.mkdir(parents=True, exist_ok=True)
        with open(out, "w", encoding="utf-8") as handle:
            json.dump(metrics, handle, indent=2)
            handle.write("\n")

    print(f"evaluate: checked {rows} fixture row(s)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
