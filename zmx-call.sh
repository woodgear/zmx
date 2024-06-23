#!/bin/zsh
source ~/.loadhome.sh > /dev/null
source /home/cong/sm/project/zmx/zmx.plugin.zsh >/dev/null
zmx-load-shell-actions >/dev/null
eval "$@"
