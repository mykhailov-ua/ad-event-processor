#!/usr/bin/env python3
"""Evaluate fraud shadow precision from ClickHouse proxy labels."""
from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

from ch_client import ch_config_from_env, connect_client, ping_client
from shadow_precision import format_markdown, run_drift_analysis, run_shadow_precision

REPO_ROOT = Path(__file__).resolve().parent.parent
ARTIFACT_DIR = os.environ.get("FRAUD_ARTIFACT_DIR", str(REPO_ROOT / "var" / "fraudscore" / "artifacts"))
DEFAULT_OUT_DIR = REPO_ROOT / "var" / "fraudscore"
_LOG = "fraudtrain/evaluate"


def _default_threshold() -> float:
    raw = os.environ.get("FRAUD_POLICY_ML_THRESHOLD", "0.6")
    try:
        return float(raw)
    except ValueError:
        return 0.6


def _default_hours() -> int:
    raw = os.environ.get("FRAUD_EVAL_HOURS", "168")
    try:
        return max(1, int(raw))
    except ValueError:
        return 168


def evaluate_shadow(
    *,
    hours: int,
    threshold: float,
    allow_offline: bool = False,
) -> dict:
    try:
        client = connect_client()
        if not ping_client(client):
            raise ConnectionError("clickhouse ping failed")
        
        report = run_shadow_precision(client, hours=hours, threshold=threshold)

        meta_path = Path(ARTIFACT_DIR) / "metadata.json"
        if meta_path.is_file():
            with meta_path.open(encoding="utf-8") as f:
                meta = json.load(f)
            raw_stats = meta.get("raw_stats")
            if raw_stats:
                drift = run_drift_analysis(client, hours=hours, training_stats=raw_stats)
                report["drift"] = drift
        
        return report
    except (ConnectionError, OSError, TimeoutError, ValueError, ImportError) as exc:
        if allow_offline:
            cfg = ch_config_from_env()
            return {
                "status": "skipped",
                "reason": str(exc),
                "host": cfg.host,
                "port": cfg.port,
                "database": cfg.database,
                "hours": hours,
                "threshold": threshold,
                "labeled_rows": 0,
            }
        raise


def write_report(report: dict, output: Path, fmt: str) -> None:
    output.parent.mkdir(parents=True, exist_ok=True)
    if fmt == "json":
        with output.open("w", encoding="utf-8") as handle:
            json.dump(report, handle, indent=2)
            handle.write("\n")
        return
    if fmt == "markdown":
        output.write_text(format_markdown(report), encoding="utf-8")
        return
    raise ValueError(f"unsupported format: {fmt}")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Shadow precision/recall from ml_shadow_scores vs proxy labels",
    )
    parser.add_argument("--hours", type=int, default=_default_hours(), help="lookback window (default 168)")
    parser.add_argument("--threshold", type=float, default=_default_threshold())
    parser.add_argument(
        "--format",
        choices=("json", "markdown", "both"),
        default="both",
        help="output format",
    )
    parser.add_argument(
        "--output",
        default="",
        help="output path stem or file (default: var/fraudscore/shadow_eval_report)",
    )
    parser.add_argument(
        "--allow-offline",
        action="store_true",
        help="exit 0 when ClickHouse is unreachable or empty",
    )
    parser.add_argument(
        "--min-labeled-rows",
        type=int,
        default=0,
        help="fail when labeled_rows below this (0 = disabled)",
    )
    args = parser.parse_args()

    report = evaluate_shadow(
        hours=args.hours,
        threshold=args.threshold,
        allow_offline=args.allow_offline,
    )

    stem = Path(args.output) if args.output else DEFAULT_OUT_DIR / "shadow_eval_report"
    if args.format in {"json", "both"}:
        json_path = stem if str(stem).endswith(".json") else stem.with_suffix(".json")
        write_report(report, json_path, "json")
        print(f"{_LOG}: wrote {json_path}", file=sys.stderr)
    if args.format in {"markdown", "both"}:
        md_path = stem if str(stem).endswith(".md") else stem.with_suffix(".md")
        write_report(report, md_path, "markdown")
        print(f"{_LOG}: wrote {md_path}", file=sys.stderr)

    if args.format == "json":
        print(json.dumps(report, indent=2))
    elif args.format == "markdown":
        print(format_markdown(report))
    else:
        print(json.dumps(report, indent=2))

    status = str(report.get("status", ""))
    labeled = int(report.get("labeled_rows", 0))
    if status == "skipped" and args.allow_offline:
        return 0
    if status == "empty" and args.allow_offline:
        return 0
    if status == "empty":
        print(f"{_LOG}: no labeled rows in ClickHouse", file=sys.stderr)
        return 1
    if args.min_labeled_rows > 0 and labeled < args.min_labeled_rows:
        print(
            f"{_LOG}: labeled_rows={labeled} below --min-labeled-rows={args.min_labeled_rows}",
            file=sys.stderr,
        )
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
