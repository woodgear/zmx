#!/bin/zsh
source ~/.loadhome.sh 2>&1 > /dev/null
source ~/sm/project/zmx/zmx.plugin.zsh 2>&1 >/dev/null
zmx-load-shell-actions 2>&1 >/dev/null
eval "$@"
