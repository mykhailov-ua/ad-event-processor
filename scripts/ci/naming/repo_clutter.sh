#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0

root_env_allow=(
  .env.example
  .env.load-test
)

deploy_txt_allow=(
  deploy/feeds/proxy_vpn.txt
)

is_allowed_root_env() {
  local base="$1"
  local allowed
  for allowed in "${root_env_allow[@]}"; do
    if [[ "$base" == "$allowed" ]]; then
      return 0
    fi
  done
  return 1
}

echo "check_repo_clutter: tracked root *.txt ban..."
while IFS= read -r -d '' path; do
  if [[ "$path" == */* ]]; then
    continue
  fi
  if [[ "$path" != *.txt ]]; then
    continue
  fi
  echo "check_repo_clutter: tracked repo-root $path (scratch reports belong in var/ci/)" >&2
  fail=1
done < <(git ls-files -z)

echo "check_repo_clutter: tracked root env ban..."
while IFS= read -r -d '' path; do
  if [[ "$path" == */* ]]; then
    continue
  fi
  base="$(basename "$path")"
  if is_allowed_root_env "$base"; then
    continue
  fi
  case "$base" in
    .env | .env.* | *.env | install.compose.env)
      echo "check_repo_clutter: tracked repo-root $base (templates: .env.example; secrets: .env.secrets)" >&2
      fail=1
      ;;
  esac
done < <(git ls-files -z)

echo "check_repo_clutter: tracked empty files..."
while IFS= read -r -d '' path; do
  rel="${path#"$ROOT"/}"
  if [[ ! -s "$path" ]]; then
    echo "check_repo_clutter: zero-byte tracked file $rel (add a comment or remove)" >&2
    fail=1
  fi
done < <(git ls-files -z)

echo "check_repo_clutter: tracked runtime env in deploy/..."
while IFS= read -r -d '' path; do
  rel="${path#"$ROOT"/}"
  base="$(basename "$path")"
  case "$base" in
    *.env.example | env.example) continue ;;
  esac
  if [[ "$base" == *.env ]] || [[ "$base" == .env* ]]; then
    echo "check_repo_clutter: tracked deploy runtime env $rel (use *.env.example)" >&2
    fail=1
  fi
done < <(git ls-files -z 'deploy/')

echo "check_repo_clutter: tracked BPF disasm txt in deploy/..."
while IFS= read -r -d '' path; do
  rel="${path#"$ROOT"/}"
  if [[ "$rel" != *.disasm.txt ]]; then
    continue
  fi
  echo "check_repo_clutter: tracked BPF disasm $rel (build artifact; gitignore it)" >&2
  fail=1
done < <(git ls-files -z 'deploy/' '*.txt')

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "check_repo_clutter: ok"
