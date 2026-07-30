
_SAFE_REALPATH=

safe_die() {
	printf 'safe_paths: ERROR: %s\n' "$*" >&2
	exit 1
}

safe_assert_not_dangerous() {
	local p="$1"
	local label="${2:-path}"
	[[ -n "$p" ]] || safe_die "$label is empty"
	case "$p" in
	/ | . | .. | ./* | ../* | */.. | */../* | *'..'* | *'*'*)
		safe_die "$label is unsafe: $p"
		;;
	esac
}

safe_realpath() {
	local p="$1"
	if command -v realpath >/dev/null 2>&1; then
		realpath -m "$p"
	else
		local dir base
		dir="$(cd "$(dirname "$p")" 2>/dev/null && pwd || echo "$(dirname "$p")")"
		base="$(basename "$p")"
		printf '%s/%s' "$dir" "$base"
	fi
}

safe_assert_repo_root() {
	[[ -f "$ROOT/go.mod" ]] || safe_die "ROOT is not a Go module root: $ROOT"
	[[ -d "$ROOT/.git" || -f "$ROOT/.git" ]] || safe_die "ROOT is not a git checkout: $ROOT"
}

safe_assert_under_repo() {
	local p="$1"
	local label="${2:-path}"
	safe_assert_not_dangerous "$p" "$label"
	safe_assert_repo_root
	local real
	real="$(safe_realpath "$p")"
	[[ "$real" != "/" ]] || safe_die "$label resolves to /"
	[[ "$real" != "$ROOT" ]] || safe_die "$label must not be repo root"
	[[ "$real" == "$ROOT"/* ]] || safe_die "$label outside repo: $real"
	_SAFE_REALPATH="$real"
}

safe_rm_rf() {
	local target="$1"
	safe_assert_under_repo "$target" "safe_rm_rf target"
	local real="$_SAFE_REALPATH"
	local base
	base="$(basename "$real")"
	[[ -n "$base" && "$base" != "." && "$base" != ".." ]] || safe_die "unsafe rm target basename: $base"
	rm -rf "$real"
}

safe_worktree_dir() {
	local p="$1"
	safe_assert_not_dangerous "$p" "BASELINE_WORKTREE"
	safe_assert_repo_root
	local real
	real="$(safe_realpath "$p")"
	[[ "$real" != "/" ]] || safe_die "BASELINE_WORKTREE resolves to /"
	[[ "$real" != "$ROOT" ]] || safe_die "BASELINE_WORKTREE must not be repo root"
	if [[ "$real" == "$ROOT/.cache/"* ]]; then
		printf '%s\n' "$real"
		return 0
	fi
	local parent base
	parent="$(dirname "$real")"
	base="$(basename "$real")"
	if [[ "$parent" == "$(dirname "$ROOT")" ]] && [[ "$base" == *baseline* ]]; then
		printf '%s\n' "$real"
		return 0
	fi
	safe_die "unsafe BASELINE_WORKTREE: $real (use $ROOT/.cache/perf-baseline-worktree)"
}

safe_validate_buf_gen_yml() {
	local f="$1"
	[[ -f "$f" ]] || safe_die "buf.gen.yaml missing: $f"
	local out
	while IFS= read -r line || [[ -n "$line" ]]; do
		[[ "$line" =~ ^[[:space:]]*out:[[:space:]]*(.+)$ ]] || continue
		out="${BASH_REMATCH[1]}"
		out="${out// /}"
		out="${out#\"}"
		out="${out%\"}"
		out="${out#\'}"
		out="${out%\'}"
		case "$out" in
		. | .. | / | '' | ./* | ../* | /* | *..*)
			safe_die "unsafe buf out in $f: $out"
			;;
		esac
		[[ "$out" != /* ]] || safe_die "unsafe absolute buf out in $f: $out"
	done <"$f"
}

safe_validate_sqlc_yml() {
	local f="$1"
	[[ -f "$f" ]] || safe_die "sqlc.yaml missing: $f"
	local out
	while IFS= read -r line || [[ -n "$line" ]]; do
		[[ "$line" =~ ^[[:space:]]*out:[[:space:]]*\"?([^\"# ]+)\"? ]] || continue
		out="${BASH_REMATCH[1]}"
		safe_assert_not_dangerous "$out" "sqlc out"
		[[ "$out" == internal/* ]] || safe_die "sqlc out must start with internal/: $out"
		[[ "$out" == */db || "$out" == */sqlc ]] || safe_die "sqlc out must end with /db or /sqlc: $out"
		safe_assert_under_repo "$ROOT/$out" "sqlc out"
	done <"$f"
}

safe_sync_proto_gen() {
	local stage="$ROOT/api/gen"
	safe_assert_under_repo "$stage" "proto staging"
	local stage_real="$_SAFE_REALPATH"
	[[ -d "$stage_real" ]] || safe_die "missing api/gen after buf generate (run from repo root)"

	local found=0
	local src svc dst
	shopt -s nullglob
	for src in "$stage_real"/internal/*/pb; do
		[[ -d "$src" ]] || continue
		svc="$(basename "$(dirname "$src")")"
		dst="$ROOT/internal/$svc/pb"
		safe_assert_under_repo "$dst" "proto dest"
		local dst_real="$_SAFE_REALPATH"
		mkdir -p "$dst_real"
		rsync -a "$src/" "$dst_real/"
		found=1
	done
	shopt -u nullglob
	[[ "$found" -eq 1 ]] || safe_die "api/gen has no internal/*/pb trees; check buf output"
}

safe_validate_codegen_configs() {
	safe_assert_repo_root
	safe_validate_buf_gen_yml "$ROOT/api/buf.gen.yaml"
	safe_validate_sqlc_yml "$ROOT/sqlc.yaml"
}
