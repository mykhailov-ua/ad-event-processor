#!/usr/bin/env bash
# Preflight for dev BPF load-test probes.
set -euo pipefail

ok=0
warn=0

check() {
	local name=$1
	shift
	if "$@" >/dev/null 2>&1; then
		printf 'bpf-requirements: OK   %s\n' "$name"
	else
		printf 'bpf-requirements: FAIL %s\n' "$name" >&2
		ok=1
	fi
}

warn_check() {
	local name=$1
	shift
	if "$@" >/dev/null 2>&1; then
		printf 'bpf-requirements: OK   %s\n' "$name"
	else
		printf 'bpf-requirements: WARN %s (optional)\n' "$name" >&2
		warn=1
	fi
}

[[ -r /sys/kernel/btf/vmlinux ]] && btf=1 || btf=0
if [[ "$btf" == "1" ]]; then
	printf 'bpf-requirements: OK   BTF vmlinux\n'
else
	printf 'bpf-requirements: WARN BTF vmlinux missing (some kernels still load tracepoints)\n' >&2
	warn=1
fi

if [[ "$(id -u)" == "0" ]]; then
	printf 'bpf-requirements: OK   root privileges\n'
else
	if [[ -r /sys/kernel/debug/tracing ]]; then
		printf 'bpf-requirements: OK   tracing fs readable\n'
	else
		printf 'bpf-requirements: WARN not root; BPF attach may need sudo\n' >&2
		warn=1
	fi
fi

GO_BIN="$(command -v go 2>/dev/null || true)"
if [[ -z "$GO_BIN" && -x /usr/local/go/bin/go ]]; then
	GO_BIN=/usr/local/go/bin/go
fi
if [[ -n "$GO_BIN" ]]; then
	check go "$GO_BIN" version
else
	printf 'bpf-requirements: FAIL go\n' >&2
	ok=1
fi
warn_check clang clang --version
warn_check bpftool bpftool version
warn_check perf perf --version

if [[ "$ok" -ne 0 ]]; then
	exit 1
fi
exit 0
