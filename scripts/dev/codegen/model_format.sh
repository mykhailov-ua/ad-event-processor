#!/usr/bin/env bash
# Role: Run ruff check --fix and normalize blank lines in model/ Python sources.
# Execution context: model/ directory; uses .venv/bin/ruff when system ruff is absent.
# Env knobs: none.
# Verify: bash scripts/dev/codegen/model_format.sh
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
model_dir="${root}/model"

cd "${model_dir}"

if ! command -v ruff > /dev/null 2>&1; then
  if [[ -x "${model_dir}/.venv/bin/ruff" ]]; then
    export PATH="${model_dir}/.venv/bin:${PATH}"
  else
    echo "model_format: install ruff (bash scripts/dev/codegen/model_venv.sh)" >&2
    exit 1
  fi
fi

ruff check --fix .

python3 - << 'PY'
import pathlib
import re

SKIP = {".venv", "__pycache__", ".ruff_cache", ".pytest_cache"}
TOP_LEVEL = re.compile(r"^(def |class |@)")
IMPORT = re.compile(r"^(from |import )")

def normalize_blank_lines(text: str) -> str:
    text = text.replace("\r\n", "\n")
    text = re.sub(r"[ \t]+\n", "\n", text)

    lines = text.split("\n")
    out: list[str] = []
    for line in lines:
        if TOP_LEVEL.match(line) and out and out[-1] != "":
            prev = next((out[j] for j in range(len(out) - 1, -1, -1) if out[j] != ""), "")
            if not prev.startswith("@") and (IMPORT.match(prev) or prev.endswith(")")):
                out.append("")
        out.append(line)

    collapsed: list[str] = []
    for line in out:
        if line != "":
            if collapsed and collapsed[-1] == "":
                prev = next((collapsed[j] for j in range(len(collapsed) - 1, -1, -1) if collapsed[j] != ""), "")
                if prev.startswith("@") and TOP_LEVEL.match(line):
                    collapsed.pop()
            collapsed.append(line)
            continue
        if collapsed and collapsed[-1] == "":
            continue
        collapsed.append("")

    return "\n".join(collapsed).rstrip() + "\n"

for path in sorted(pathlib.Path(".").rglob("*.py")):
    if SKIP.intersection(path.parts):
        continue
    original = path.read_text(encoding="utf-8")
    updated = normalize_blank_lines(original)
    if updated != original:
        path.write_text(updated, encoding="utf-8")
PY

ruff check .
