#!/usr/bin/env bash
# One-liner installer: download release tarball or clone repo, then run bidshard-install.sh.
set -euo pipefail

REPO="${BIDSHARD_INSTALL_REPO:-bidshard/bidshard}"
VERSION="${BIDSHARD_VERSION:-latest}"
INSTALL_DIR="${BIDSHARD_INSTALL_DIR:-${HOME}/bidshard}"
USE_GIT="${BIDSHARD_INSTALL_FROM_GIT:-0}"
GET_SCRIPT_URL="${BIDSHARD_GET_SCRIPT_URL:-}"

log() {
	echo "bidshard-get: $*"
}

need_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "bidshard-get: required command not found: $1" >&2
		exit 1
	fi
}

resolve_latest_tag() {
	curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])'
}

install_from_tarball() {
	local tag="$1"
	local tmp
	tmp="$(mktemp -d)"
	for name in ad-event-processor-installer bidshard-installer; do
		local url="https://github.com/${REPO}/releases/download/${tag}/${name}.tar.gz"
		log "downloading ${url}"
		if curl -fsSL "$url" | tar -xz -C "$tmp" 2>/dev/null; then
			rm -rf "$INSTALL_DIR"
			mv "$tmp/bidshard" "$INSTALL_DIR"
			rm -rf "$tmp"
			return 0
		fi
	done
	rm -rf "$tmp"
	return 1
}

install_from_git() {
	need_cmd git
	if [[ -d "$INSTALL_DIR/.git" ]]; then
		log "updating existing clone at $INSTALL_DIR"
		cd "$INSTALL_DIR"
		git pull --ff-only
	else
		rm -rf "$INSTALL_DIR"
		log "cloning https://github.com/${REPO}.git"
		git clone "https://github.com/${REPO}.git" "$INSTALL_DIR"
		cd "$INSTALL_DIR"
	fi
	if [[ "$VERSION" != "latest" ]] && [[ "$VERSION" != "main" ]]; then
		git checkout "$VERSION"
	fi
}

main() {
	need_cmd curl
	need_cmd python3

	local tag="$VERSION"
	if [[ "$USE_GIT" == "1" ]]; then
		install_from_git
	elif [[ "$VERSION" == "latest" ]]; then
		tag="$(resolve_latest_tag)"
		if ! install_from_tarball "$tag"; then
			log "release tarball missing; falling back to git clone"
			USE_GIT=1
			install_from_git
		fi
	else
		if ! install_from_tarball "$tag"; then
			log "release tarball missing for ${tag}; falling back to git clone"
			install_from_git
		fi
	fi

	cd "$INSTALL_DIR"
	exec bash scripts/install/ad-event-processor-install.sh --yes "$@"
}

if [[ -n "$GET_SCRIPT_URL" ]]; then
	log "remote script URL set; prefer: curl -fsSL ${GET_SCRIPT_URL} | bash"
fi

main "$@"
