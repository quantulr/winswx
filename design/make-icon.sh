#!/usr/bin/env bash
# Draws resources/icon.png: a solid rounded square with a Segoe UI Black X
# behind "winsw", whose letters wear the MyGO!!!!! member colors, two on
# the first row and two on the second. Needs ImageMagick, Segoe UI Black
# and Arial Rounded MT Bold.
set -euo pipefail
cd "$(dirname "$0")/.."

letterfont="C:/Windows/Fonts/ARLRDBD.TTF"
xfont="C:/Windows/Fonts/seguibl.ttf"
base="#F5F2FA"
xtint="#E3DAF0"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# The X, trimmed to its ink so that it sits exactly in the middle of the
# flat background; the pair is then clipped to the rounded square.
magick -background none -font "$xfont" -pointsize 1500 -fill "$xtint" \
	label:"X" -trim +repage "$tmp/x.png"
magick -size 1024x1024 "xc:$base" "$tmp/x.png" -gravity center \
	-composite "$tmp/flat.png"
magick -size 1024x1024 xc:black -fill white \
	-draw "roundrectangle 0,0 1023,1023 228,228" "$tmp/mask.png"
magick "$tmp/flat.png" "$tmp/mask.png" -alpha off -compose CopyOpacity \
	-composite "$tmp/background.png"

letters=(w i n s w)
colors=("#5FB3DE" "#FB7FA0" "#00A897" "#F0B94A" "#A966BA")
angles=(-8 5 -5 6 -7)
offsets=("-266-160" "-68-195" "+118-154" "-20+160" "+224+126")

args=()
for i in "${!letters[@]}"; do
	# Each letter, with a deep indigo outline that keeps it readable small.
	magick -background none -font "$letterfont" -pointsize 360 \
		-stroke "#2B2440" -strokewidth 14 -fill "${colors[$i]}" \
		label:"${letters[$i]}" -rotate "${angles[$i]}" "$tmp/$i.png"
	args+=(\( "$tmp/$i.png" \) -gravity center -geometry "${offsets[$i]}" -composite)
done

magick -background none "$tmp/background.png" "${args[@]}" resources/icon.png
