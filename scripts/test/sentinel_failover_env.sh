#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

cp .env.example .env
sed -i 's/your_redis_password_here/sentinel_fault_ci/' .env
