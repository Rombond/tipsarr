#!/bin/bash
# usage: render.sh name [maxdim]  renders svg/<name>.svg at native size to png/<name>.png (downscaled to maxdim if given)
mkdir -p ~/penpot-work/png
f=~/penpot-work/svg/$1.svg
python3 ~/penpot-work/clean.py $f ~/penpot-work/svg/$1.clean.svg && f=~/penpot-work/svg/$1.clean.svg
read W H < <(python3 - "$HOME/penpot-work/svg/$1.clean.svg" <<'PY'
import re,sys
s=open(sys.argv[1]).read(4000)
w=re.search(r'width="([\d.]+)"',s); h=re.search(r'height="([\d.]+)"',s)
print(int(float(w.group(1))), int(float(h.group(1))))
PY
)
chromium --headless=new --no-sandbox --disable-gpu --hide-scrollbars --default-background-color=00000000 --window-size=$W,$((H+120)) --screenshot=$HOME/penpot-work/png/$1.full.png "file://$f" >/dev/null 2>&1
magick $HOME/penpot-work/png/$1.full.png -crop ${W}x${H}+0+0 +repage $HOME/penpot-work/png/$1.png
rm -f $HOME/penpot-work/png/$1.full.png
if [ -n "$2" ]; then magick $HOME/penpot-work/png/$1.png -resize ${2}x${2}\> $HOME/penpot-work/png/$1.png; fi
echo $HOME/penpot-work/png/$1.png
