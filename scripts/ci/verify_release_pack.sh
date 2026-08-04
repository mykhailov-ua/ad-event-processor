#!/usr/bin/env bash
# Fail if customer release tarball contains vendor-only artifacts (S1.6).
set -euo pipefail

TARBALL="${1:?usage: verify_release_pack.sh <tarball>}"

FORBIDDEN_PATTERNS=(
	'license_private'
	'sku.yaml'
	'/cmd/license-issue'
	'license-issue'
)

LIST="$(tar -tzf "$TARBALL")"
for pat in "${FORBIDDEN_PATTERNS[@]}"; do
	if echo "$LIST" | grep -qi "$pat"; then
		echo "verify_release_pack: forbidden path matching '$pat' in $TARBALL" >&2
		exit 1
	fi
done

if ! echo "$LIST" | grep -q 'deploy/vendor/license_public.key'; then
	echo "verify_release_pack: missing deploy/vendor/license_public.key in $TARBALL" >&2
	exit 1
fi

echo "verify_release_pack: OK ($TARBALL)"
