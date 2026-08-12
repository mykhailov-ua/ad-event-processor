#!/usr/bin/env bash
# Deprecated symlink target: use ad-event-processor-install.sh (one release).
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/ad-event-processor-install.sh" "$@"
