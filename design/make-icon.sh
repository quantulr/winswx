#!/usr/bin/env bash
# Draws resources/icon.png: an X in the MyGO!!!!! member colors behind
# "winsw", with one member color per letter. Needs ImageMagick and Arial
# Rounded MT Bold.
set -euo pipefail
cd "$(dirname "$0")/.."

font="C:/Windows/Fonts/ARLRDBD.TTF"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# The background: the rounded square, its washes, and the X.
magick -background none design/icon.svg "$tmp/background.png"

letters=(w i n s w)
colors=("#5FB3DE" "#FB7FA0" "#00A897" "#F0B94A" "#A966BA")
angles=(-8 5 -5 6 -7)
offsets=("-222-137" "-57-167" "+98-132" "-17+138" "+178+108")

args=()
for i in "${!letters[@]}"; do
	# Each letter, with a deep indigo outline that keeps it readable small.
	magick -background none -font "$font" -pointsize 300 \
		-stroke "#2B2440" -strokewidth 12 -fill "${colors[$i]}" \
		label:"${letters[$i]}" -rotate "${angles[$i]}" "$tmp/$i.png"
	args+=(\( "$tmp/$i.png" \) -gravity center -geometry "${offsets[$i]}" -composite)
done

magick -background none "$tmp/background.png" "${args[@]}" resources/icon.png
