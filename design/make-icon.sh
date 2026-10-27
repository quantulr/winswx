#!/usr/bin/env bash
# Converts an artwork into resources/icon.png: its white backdrop turns
# transparent, the art is trimmed, padded to a square, scaled to 512 and
# compressed. Needs ImageMagick.
#
#	bash design/make-icon.sh path/to/art.png
set -euo pipefail
cd "$(dirname "$0")/.."

if [ $# -ne 1 ]; then
	echo "usage: $0 <artwork.png>" >&2
	exit 2
fi
artwork=$1

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# Flood the white backdrop with transparency from a corner, keeping the
# rounded edge of the art as it is.
magick "$artwork" -alpha on -fuzz 3% -fill none \
	-draw "color 0,0 floodfill" "$tmp/cut.png"

# Trim the transparent border, pad to a square, and scale to the icon size.
magick "$tmp/cut.png" -trim +repage "$tmp/trimmed.png"
w=$(magick identify -format "%w" "$tmp/trimmed.png")
h=$(magick identify -format "%h" "$tmp/trimmed.png")
side=$((w > h ? w : h))
magick "$tmp/trimmed.png" \
	-background none -gravity center -extent "${side}x${side}" \
	-strip -filter Lanczos -resize 512x512 \
	-define png:compression-level=9 -define png:compression-filter=5 \
	resources/icon.png
