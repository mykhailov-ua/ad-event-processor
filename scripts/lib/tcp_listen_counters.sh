#!/usr/bin/env bash

set -euo pipefail

tcp_listen_counter() {
  local name="$1"
  if command -v nstat > /dev/null 2>&1; then
    nstat -az 2> /dev/null | awk -v want="$name" '
			$1 == want { print $2; found=1; exit }
			END { if (!found) print 0 }
		'
    return 0
  fi
  awk -v want="$name" '
		$1 == want ":" { gsub(":", "", $1); if ($1 == want) { print $2; found=1; exit } }
		END { if (!found) print 0 }
	' /proc/net/netstat 2> /dev/null || echo 0
}

tcp_listen_snapshot_file() {
  local dest="$1"
  {
    printf '# at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'TcpExtListenOverflows=%s\n' "$(tcp_listen_counter TcpExtListenOverflows)"
    printf 'TcpExtListenDrops=%s\n' "$(tcp_listen_counter TcpExtListenDrops)"
  } > "$dest"
}

tcp_listen_delta_from_files() {
  local before="$1" after="$2"
  python3 - "$before" "$after" << 'PY'
import sys

def read(path):
    out = {}
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" not in line:
                continue
            k, v = line.split("=", 1)
            out[k] = int(v)
    return out

before = read(sys.argv[1])
after = read(sys.argv[2])
for key in ("TcpExtListenOverflows", "TcpExtListenDrops"):
    delta = after.get(key, 0) - before.get(key, 0)
    print(f"{key}_delta={delta}")
PY
}

tcp_sysctl_backlog_check() {
  local min="${1:-2048}"
  local ok=1
  for key in net.core.somaxconn net.ipv4.tcp_max_syn_backlog; do
    local val
    val="$(sysctl -n "$key" 2> /dev/null || echo 0)"
    if [[ "$val" -lt "$min" ]]; then
      printf 'tcp-backlog-check: WARN %s=%s < %s\n' "$key" "$val" "$min" >&2
      ok=0
    fi
  done
  return "$ok"
}
