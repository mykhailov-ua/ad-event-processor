#!/usr/bin/env bash
# Role: Create Python venv under model/ with pip, requirements, and pyright.
# Execution context: Repo root; downloads get-pip.py (network required).
# Env knobs: none.
# Verify: bash scripts/dev/codegen/model_venv.sh && model/.venv/bin/python -c 'import sys; print(sys.version)'
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT/model"

rm -rf .venv
python3 -m venv .venv --without-pip
curl -fsSL https://bootstrap.pypa.io/get-pip.py | .venv/bin/python
.venv/bin/pip install -r requirements.txt -r requirements-dev.txt

echo "model_venv: OK"
echo "  interpreter: $ROOT/model/.venv/bin/python"
echo "  pyright:     $ROOT/model/.venv/bin/pyright ($(.venv/bin/pyright --version))"
