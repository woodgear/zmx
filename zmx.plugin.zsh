#!/bin/zsh
export ZMX_BASE=~/.zmx

function _date_now() {
    date +"%Y-%m-%0eT%T.%6N"
}

function _zmx_compile() (
  cd ~/.zmx
  if [ -f ./aio.sh ]; then
    rm ./aio.sh
  fi
  touch ./aio.sh
  for p in $(cat ./import.sh | awk '{print $2}' | sort | uniq); do
    echo "zmx compile $p $(file ./aio.sh)"
    cat "$p" | grep -v '#!' >> ./aio.sh
  done
  echo "zcompile build "
  file ~/.zmx/aio.sh
  zcompile ./aio.sh
)

function zmx-find-base-of-action() {
  local f=$1
  if [ -z "$f" ]; then
    f=$(print -l $functrace | head -n 1 | cut -d ':' -f 1)
  fi
  local p=$(type -a $f | rg -o 'from (.*)$' -r '$1')
  local p=$(readlink -f $p)
  echo $(dirname "$p")
}

function zmx-find-path-of-action() {
  local f=$1
  if [ -z "$f" ]; then
    f=$(print -l $functrace | head -n 1 | cut -d ':' -f 1)
  fi
  local p=$(cat ~/.zmx/actions.db |grep $f| awk '{print $2}')
  local p=$(readlink -f $p)
  echo "$p"
}

function zmx-reload-shell-actions() {
  echo "action path" $SHELL_ACTIONS_PATH
  local actions_path=$SHELL_ACTIONS_PATH
  local base=$ZMX_BASE
  mkdir -p $base
  local start=$(_date_now)
  echo $start
  # will link add actions under $base/index
  _zmx_index_all_actions $base
  # run auto gen command first 
  _zmx_autogen $base
  # will build actions db under $base/action.db from $base/index
  _zmx_build_db $base $base/index
  # will gen a sh include all source xx under $base/import.sh
  _zmx_gen_import $base $base/actions.db
  # will gen a md5 include all source xx under $base/import.sh
  _zmx_gen_md5 $base $base/actions.db
#   _zmx_compile $base
  zmx-load-shell-actions
  local end=$(_date_now)
  echo $start
  echo $end
  local record="reload over, spend $(time-diff "$start" "$end")."
  echo "$record"
  echo $record >>$ZMX_BASE/record
  #   cat $ZMX_BASE/record
}

function _zmx_index_all_actions() (
  local start=$(_date_now)
  local base=$1
  cd $base
  local index_path=$base/index
  rm -rf ./index
  mkdir -p ./index
  echo "start index"
  local actions_path=$SHELL_ACTIONS_PATH
  local fail="0"
  for p in $(echo $actions_path | sed "s/:/ /g" | sort | uniq); do
    if [ ! -e "$p" ]; then
      echo "$p not exist "
      continue
    fi
    local link=$(echo $p | sed 's/\//_/g')
    echo index $p $index_path/$link
    if [ -e $index_path/$link ]; then
      rm $index_path/$link
    fi
    ln -s $p $index_path/$link
    fail="$?"
    if [[ ! "$fail" == "0" ]]; then
      break
    fi
  done
  local end=$(_date_now)
  local record="index over, spend $(time-diff "$start" "$end")."
  if [[ ! "$fail" == "0" ]]; then
    echo "sth fail"
    return 1
  fi
  echo $record
  echo $record >>$ZMX_BASE/record
)

function _zmx_autogen() (
  for p in $(echo $ZMX_GEN_PATH | sed "s/:/ /g" | sort | uniq); do
    echo "gen $p"
    $p
  done
)


function zmx-list-actions-raw() {
  local index=$ZMX_BASE/index
  rg -L --with-filename --line-number -g '*.{sh,bash,zsh}' '^function\s*([^\s()_]+).*[\(\{][^\}\)]*$' -r '${1}' $index | rg '^(.*):(.*):(.*)$' -r '$3   $1   $2'
}

function zmx-list-actions-from-zsh() {
    print -l ${(k)functions_source[(R)*aio*]}
}

# 生成函数和文件的引用关系
function _zmx_build_db() (
  local base=$1
  local index=$2
  cd $base
  echo "start build"

  local start=$(_date_now)
  zmx-list-actions-raw $index >$base/actions.db
  cat $base/actions.db
  local end=$(_date_now)
  local record="build over, spend $(time-diff "$start" "$end")."
  echo $record
  echo $record >>$ZMX_BASE/record
)

function _zmx_gen_import() (
  echo "start gen import"
  local start=$(_date_now)
  local base=$1
  local db=$2
  # echo $base $db
  cat $db | awk '{print $2}' | sort | uniq | xargs -I {} echo "source {} || true" >$base/import.sh
  # TODO alias
  local end=$(_date_now)
  local record="gen-import over, spend $(time-diff "$start" "$end")."
  echo $record
  echo $record >>$ZMX_BASE/record
)

function _zmx_gen_md5() (
  echo "start gen md5"
  local start=$(_date_now)
  local base=$1
  local db=$2
  rm -rf $base/md5
  mkdir $base/md5
  cd $base/md5
  echo $base $db
  cat $db | awk '{print $2}' | sort | uniq | xargs -I {} sh -c 'sp={};md5p=$(echo {} | sed "s|/|_|g");md5=$(md5sum $sp | awk "{print $1}");echo $md5 > ./$md5p.md5'
  local end=$(_date_now)
  local record="gen-md5 over, spend $(time-diff "$start" "$end")."
  echo $record
  echo $record >>$ZMX_BASE/record
)

function time-diff_() {
    if which time-diff > /dev/null ;then 
        time-diff $@
    else
        return ""
    fi
}

function zmx-log() {
    echo "$(_date_now) $@" >>$ZMX_BASE/.zmx.log
}

function zmx-watch-log() {
    tail -F $ZMX_BASE/.zmx.log
}

function zmx-load-shell-actions() {
  # local actions_path=$SHELL_ACTIONS_PATH
  # echo "start load " $actions_path
  local start=$(_date_now)
  echo "start source"
  # echo $actions_path
  if [ ! -f $ZMX_BASE/import.sh ]; then
    echo "no actions found ignore"
    return
  fi

  if [ -f $ZMX_BASE/aio.sh.zwc ];then
   echo "load actions from cache"
   source $ZMX_BASE/aio.sh
   else
   echo "load actions from import"
    source $ZMX_BASE/import.sh
  fi
  echo "end source $?"
  # for action in $(print -rl ${(k)functions_source[(R)*shell-actions*]});do
  # done
  local count=$(count-actions)
  local fn_count=$(zmx-list-actions-from-zsh | wc -l)
  local end=$(_date_now)
  local record="load over, actions-db-fn $count zsh-fn $fn_count spend $(time-diff_ "$start" "$end")."
  echo $record
  echo $record >>$ZMX_BASE/record
}

function edit-x-actions() {
  local cmd=$(list-x-actions | fzf)
  local source_file=$(type $cmd | rg -o '.* from (.*)' -r '$1' | tr -d '\n\r')

  local cmd_start_line=$(grep -no "$cmd()" $source_file | cut -d ':' -f 1 | tr -d '\n\r')
  echo $source_file
  echo $cmd_start_line
  # @keyword: vim edit file in special line
  vim +$cmd_start_line $source_file
}

function which-x-actions() {
  local cmd=$(list-x-actions | fzf)
  which $cmd
}

function list-x-actions() {
  cat $ZMX_BASE/actions.db | awk '{print $1}'
}

function count-actions() {
  list-x-actions | wc -l
}

function _zmx_before_run_action() {
  local name=$1
}

function mx-without-zle() {
  local name=$(list-x-actions | fzf)
  if [[ -z "$name" ]]; then
    echo "canceled"
    return
  fi
  if [[ $(zmx-action-have-arg $name) == "true" ]]; then
    _zmx_before_run_action $name
    echo "$name"
  else
    _zmx_before_run_action $name
    eval $name
  fi
}

function mx() {
  local name=$(list-x-actions | fzf)
  if [[ $(zmx-action-have-arg $name) == "true" ]]; then
    _zmx_before_run_action $name
    LBUFFER+=$name
    LBUFFER+=" "
    zle reset-prompt
  else
    _zmx_before_run_action $name
    eval $name
    zle reset-prompt
  fi
}

function zmx-actions-info() {
  local name=$1
  rg "$name" $ZMX_BASE/actions.db
}

function zmx-action-have-arg() {
  local name=$1
  read name source_file line <<<$(zmx-actions-info $name)
  if grep "function\s*$name" $source_file -A 1 | grep -q 'arg-len'; then
    echo "true"
  else
    echo "false"
  fi
}

function lmx() {
  # mx local action
  local source_file=$(fd -a actions.sh ./)
  echo "source" $source_file
  local cmd=$(cat $source_file | rg "^\s*function\s(.*)\s*\{$" -r '$1' | grep -v '_.*' | fzf --preview "grep  {} $source_file -A 5")
  local cmd=$(echo $cmd | sed "s/()\s//g")
  echo "cmd $cmd"
  if [ -z "$cmd" ]; then
    echo "empty cmd ignore"
    zle reset-prompt
    return
  fi
  local annotation=$(grep "function\s$cmd" $source_file -A 1)
  echo "arg-annno $annotation"
  if echo $annotation | grep -q 'arg-len'; then
    LBUFFER+="source $source_file; $cmd"
    LBUFFER+=" "
    zle reset-prompt
  else
    local full_cmd="source $source_file; $cmd"
    echo "full_cmd $full_cmd"
    local atuin_id=$(atuin history start "$full_cmd")
    eval $full_cmd
    atuin history end $atuin_id --exit "$?"
    zle reset-prompt
  fi
}

# 
function _zmx_record() {
    local ac="$1"
    local now=$(_date_now)
    echo "$now $ac" >> ~/.zmx/actions.record
}

function zmx-is-action() (
  local name=$1
  if [ -n "$(rg "^$name" $ZMX_BASE/actions.db 2>/dev/null)" ]; then
    echo "true"
    return
  fi
  echo "false"
)

function zmx_preexec() {
  # 如果是zmx的action的话，如果是dirty的，先source一下
  local name=$(echo $1 | awk '{ print $1}')
  if [[ "$(zmx-is-action $name)" == "false" ]]; then
    # TODO may be we just add a actions?
    return
  fi
  read name source_file line <<<$(zmx-actions-info $name)
  if [[ -n "$source_file" ]];then
  echo "$(_date_now) $1" >> ~/.zmx/actions.record
  # 在source 一下 这样基本就能满足很多场景了
  local source_file=$(readlink -f $source_file)
  if [[ -n "$source_file" ]];then
    source $source_file
  fi
  fi

#   _zmx_dev "$@"

#   _zmx_record "$1"

#   if [[ "$(zmx-actions-dirty $name)" == "true" ]]; then
#     echo "action $name dirty, source $source_file"
#     source $source_file
#     zmx-upate-md5 $name
#     return
#   fi
}



function zmx-bind-key() {
  zle -N mx
  zle -N lmx
  bindkey ',xx' lmx
  bindkey ',xm' mx
}

function zmx-add-path() {
  local p=$(readlink -f $1)
  if [ -z "$p" ]; then
    echo "empty path"
    return
  fi
  if [ $(zmx-list-path | grep -F "$p") ]; then
    echo "already have"
    return 1
  fi
  local new_path="export SHELL_ACTIONS_PATH=\$SHELL_ACTIONS_PATH:$p"
  echo "new_path" $new_path
  echo "\n$new_path" >>~/.env/.$(hostname).env
  # make new env work
  source ~/.loadhome.sh
  # reload
  zmx-reload-shell-actions
  zsh
}

function zmx-list-path() {
  echo $SHELL_ACTIONS_PATH | sed $'s/:/\\n/g' | sort | uniq
}


function zmx-show-recocds() {
    cat ~/.zmx/actions.record
}

function zmx-select() (
  if [[ -n "$IN_ROFI" ]]; then
    rofi -dpi 1 -dmenu
    return
  fi
  fzf
  return
)

function zmx-get-input() (
  local prompt=${1-"name: "}
  if [ -n "$2" ]; then
    echo "$2"
    return
  fi
  if [ -n "$IN_ROFI" ]; then
    echo $(zenity --entry --text="$prompt")
    return
  fi
  a=$(bash -c "read -e -p \"$prompt\" tmp; echo \$tmp")
  echo $a
  return
)

function zmx-auto-source() {
  echo "auto source loop.sh"
  while read -r file; do
    if [[ -z "$file" ]]; then
      continue
    fi
    echo "source $file"
    source $file
  done <<< "$(fd -a --follow --no-ignore loop.sh)"
    echo "auto source actions.sh"
  while read -r file; do
    if [[ -z "$file" ]]; then
      continue
    fi
    echo "source $file"
    source $file
  done <<< "$(fd -a --follow --no-ignore actions.sh)"
}