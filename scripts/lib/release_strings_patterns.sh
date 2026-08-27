#!/usr/bin/env bash

# Core licensing symbol anchors garble must strip on every release binary.
release_strings_forbidden_core() {
  printf '%s\n' \
    'IngestAllowed' \
    'VerifyLicense' \
    'VerifyJWT' \
    'license file verification' \
    'internal/licensing' \
    'BEGIN PUBLIC' \
    'ede21d8e759af2ba68a74149d28f37a859d33497accee01e8f8ac712bd455c70' \
    'ede21d8e759af2ba'
}

# Extra patterns for processor/control (-literals=1). Tracker keeps literals off so path
# strings like deploy/vendor/license_public.key remain; byte needles are obfuscated there too.
release_strings_forbidden_literals_on() {
  printf '%s\n' \
    'embeddedPubKey' \
    'license_public' \
    'EdDSA'
}

release_strings_cmd_from_binary() {
  basename "$1"
}

release_strings_scan_patterns() {
  local bin="$1"
  local cmd
  cmd="$(release_strings_cmd_from_binary "$bin")"

  release_strings_forbidden_core
  case "$cmd" in
    processor | control)
      release_strings_forbidden_literals_on
      ;;
  esac
}

# Raw byte needles that must not survive garble when -literals=1 (processor/control).
release_strings_scan_binary_bytes() {
  local bin="$1"
  local cmd
  cmd="$(release_strings_cmd_from_binary "$bin")"
  case "$cmd" in
    processor | control) ;;
    *) return 0 ;;
  esac

  python3 - "$bin" << 'PY'
import sys
path = sys.argv[1]
data = open(path, "rb").read()
needles = [
    bytes.fromhex("4ade8cd05761e63205a68f0e6bfd0775"),
    bytes.fromhex("a73c915e22fb14886d01ce47b97230dd"),
    bytes.fromhex("1659578e49989bda8d27ff6323600cf6"),
    bytes.fromhex("4f8a6319e5567bc402ad38719e255086"),
]
for needle in needles:
    if needle in data:
        print(f"raw embed needle ({len(needle)} bytes) found in {path}", file=sys.stderr)
        sys.exit(1)
PY
}

release_strings_scan_binary() {
  local bin="$1"
  local pat

  if [[ ! -f "$bin" ]]; then
    echo "release_strings_gate: missing binary: $bin" >&2
    return 1
  fi

  while IFS= read -r pat; do
    [[ -z "$pat" ]] && continue
    if strings "$bin" | rg -qi "$pat"; then
      echo "release_strings_gate: forbidden pattern '$pat' in $bin" >&2
      strings "$bin" | rg -i "$pat" | head -5 >&2 || true
      return 1
    fi
  done < <(release_strings_scan_patterns "$bin")

  release_strings_scan_binary_bytes "$bin"
}
