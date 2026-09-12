#!/usr/bin/env bash
# Role: Ensure deploy/marketing/assets/product_avatar.* exist before marketing rsync.
# Source: deploy/marketing/assets/avatar.jpeg (512px SVG + JPG derivatives).
# Verify: bash scripts/ops/bundle_marketing_assets.sh && ls deploy/marketing/assets/product_avatar.*
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

ASSETS="$ROOT/deploy/marketing/assets"
SRC="$ASSETS/avatar.jpeg"
SVG="$ASSETS/product_avatar.svg"
JPG="$ASSETS/product_avatar.jpg"

if [[ -f "$SVG" && -f "$JPG" ]]; then
  if [[ ! -f "$SRC" ]]; then
    echo "bundle_marketing_assets: using committed product_avatar.* (no avatar.jpeg source)"
    WEB_ASSETS="$ROOT/web/src/assets"
    mkdir -p "$WEB_ASSETS"
    cp "$SVG" "$WEB_ASSETS/product_avatar.svg"
    cp "$JPG" "$WEB_ASSETS/product_avatar.jpg"
    exit 0
  fi
fi

if [[ ! -f "$SRC" ]]; then
  echo "bundle_marketing_assets: missing $SRC and product_avatar.*" >&2
  exit 1
fi

python3 - "$ASSETS" << 'PY'
import base64
import io
import sys
from pathlib import Path

assets = Path(sys.argv[1])
src = assets / "avatar.jpeg"
svg_path = assets / "product_avatar.svg"
jpg_path = assets / "product_avatar.jpg"

try:
    from PIL import Image
except ImportError as exc:
    raise SystemExit("bundle_marketing_assets: Pillow required (python3-pil)") from exc

img = Image.open(src).convert("RGB")
size = 512
img = img.resize((size, size), Image.Resampling.LANCZOS)

jpg_buf = io.BytesIO()
img.save(jpg_buf, format="JPEG", quality=82, optimize=True, progressive=True)
jpg_bytes = jpg_buf.getvalue()
encoded = base64.b64encode(jpg_bytes).decode("ascii")

svg = (
    '<?xml version="1.0" encoding="UTF-8"?>\n'
    f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" '
    f'viewBox="0 0 {size} {size}" role="img" aria-label="Ad Event Processor">\n'
    f'  <image width="{size}" height="{size}" preserveAspectRatio="xMidYMid slice" '
    f'href="data:image/jpeg;base64,{encoded}"/>\n'
    '</svg>\n'
)

jpg_path.write_bytes(jpg_bytes)
svg_path.write_text(svg, encoding="utf-8")
print(f"bundle_marketing_assets: wrote {svg_path.name} ({len(svg.encode())} bytes)")
print(f"bundle_marketing_assets: wrote {jpg_path.name} ({len(jpg_bytes)} bytes)")
PY

# Admin SPA embed copies the same canonical marketing assets.
WEB_ASSETS="$ROOT/web/src/assets"
mkdir -p "$WEB_ASSETS"
cp "$SVG" "$WEB_ASSETS/product_avatar.svg"
cp "$JPG" "$WEB_ASSETS/product_avatar.jpg"
