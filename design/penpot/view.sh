#!/bin/bash
# usage: view.sh out.png maxwidth name1 name2 ...   (renders each svg, joins them side by side)
out=$1; mw=$2; shift 2
files=()
for n in "$@"; do ~/penpot-work/render.sh $n >/dev/null; files+=("$HOME/penpot-work/png/$n.png"); done
magick "${files[@]}" -background '#444' +append -resize ${mw}x\> ~/penpot-work/png/$out
echo ~/penpot-work/png/$out
