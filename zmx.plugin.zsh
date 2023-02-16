#!/bin/zsh

echo "load zmx"

export ZMX_BASE=~/.zmx

function date-ms() {
  date +"%Y %m %e %T.%6N"
}

function _zmx_compile() (
  cd ~/.zmx
  rm ./aio.sh
  for p in $(cat ./import.sh | awk '{print $2}' | sort | uniq); do
    echo "$p"
    cat $p | grep -v '#!' >>./aio.sh
  done
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
  local p=$(type -a $f | rg -o 'from (.*)$' -r '$1')
  local p=$(readlink -f $p)
  echo "$p"
}

function zmx-reload-shell-actions() {
  echo "action path" $SHELL_ACTIONS_PATH
  local actions_path=$SHELL_ACTIONS_PATH
  local base=$ZMX_BASE
  mkdir -p $base

  local start=$(date-ms)
  # will link add actions under $base/index
  _zmx_index_all_actions $base
  # will build actions db under $base/action.db from $base/index
  _zmx_build_db $base $base/index
  # will gen a sh include all source xx under $base/import.sh
  _zmx_gen_import $base $base/actions.db
  # will gen a md5 include all source xx under $base/import.sh
  _zmx_gen_md5 $base $base/actions.db
  _zmx_compile $base
  zmx-load-shell-actions
  local end=$(date-ms)
  local record="reload over, spend $(time-diff-ms "$start" "$end")."
  echo "$record"
  echo $record >>$ZMX_BASE/record
  #   cat $ZMX_BASE/record
}

function _zmx_index_all_actions() (
  local start=$(date-ms)
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
  local end=$(date-ms)
  local record="index over, spend $(time-diff-ms "$start" "$end")."
  if [[ ! "$fail" == "0" ]]; then
    echo "sth fail"
    return 1
  fi
  echo $record
  echo $record >>$ZMX_BASE/record
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

  local start=$(date-ms)
  zmx-list-actions-raw $index >$base/actions.db
  cat $base/actions.db
  local end=$(date-ms)
  local record="build over, spend $(time-diff-ms "$start" "$end")."
  echo $record
  echo $record >>$ZMX_BASE/record
)

function _zmx_gen_import() (
  echo "start gen import"
  local start=$(date-ms)
  local base=$1
  local db=$2
  # echo $base $db
  cat $db | awk '{print $2}' | sort | uniq | xargs -I {} echo "source {}" >$base/import.sh
  local end=$(date-ms)
  local record="gen-import over, spend $(time-diff-ms "$start" "$end")."
  echo $record
  echo $record >>$ZMX_BASE/record
)

function _zmx_gen_md5() (
  echo "start gen md5"
  local start=$(date-ms)
  local base=$1
  local db=$2
  rm -rf $base/md5
  mkdir $base/md5
  cd $base/md5
  echo $base $db
  cat $db | awk '{print $2}' | sort | uniq | xargs -I {} sh -c 'sp={};md5p=$(echo {} | sed "s|/|_|g");md5=$(md5sum $sp | awk "{print $1}");echo $md5 > ./$md5p.md5'
  local end=$(date-ms)
  local record="gen-md5 over, spend $(time-diff-ms "$start" "$end")."
  echo $record
  echo $record >>$ZMX_BASE/record
)

function zmx-load-shell-actions() {
  # local actions_path=$SHELL_ACTIONS_PATH
  # echo "start load " $actions_path
  local start=$(date-ms)
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
  #     short=$(echo $action | sed 's/-//g')
  #     alias $short=$action
  # done
  local count=$(count-actions)
  local fn_count=$(zmx-list-actions-from-zsh | wc -l)
  local end=$(date-ms)
  local record="load over, actions db $count $fn_count spend $(time-diff-ms "$start" "$end")."
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

function zmx-is-full-reload() {
  if [ -f "$ZMX_BASE/full-reload" ]; then
    echo "true"
    return
  fi
  echo "false"
}
function zmx-enable-full-reload() {
  touch $ZMX_BASE/full-reload
}
function zmx-disable-full-reload() {
  rm $ZMX_BASE/full-reload
}

# may be in dev lopp
function _zmx_dev() {
  if [[ ! "$(zmx-is-full-reload)" == "true" ]]; then
    return
  fi
  echo "force reload"
  zmx-reload-shell-actions
}

function zmx_preexec() {
  # 如果是zmx的action的话，如果是dirty的，先source一下
  local name=$(echo $1 | awk '{ print $1}')
  if [[ "$(zmx-is-action $name)" == "false" ]]; then
    # TODO may be we juist add a actions?
    return
  fi

  _zmx_dev "$@"

  read name source_file line <<<$(zmx-actions-info $name)
  if [[ "$(zmx-actions-dirty $name)" == "true" ]]; then
    echo "action $name dirty, source $source_file"
    source $source_file
    zmx-upate-md5 $name
    return
  fi
}

function _zmx_gen_md5p() {
  local file=$1
  local md5p="$ZMX_BASE/md5/$(echo $file | sed 's|/|_|g').md5"
  echo $md5p
}

function zmx-upate-md5() {
  local name=$1
  read name source_file line <<<$(zmx-actions-info $name)
  local old=$(zmx-cache-md5 $name)
  local md5p=$(_zmx_gen_md5p $source_file)
  local md5=$(zmx-cur-md5 $name)
  echo $md5 >$md5p
  local new=$(zmx-cache-md5 $name)
  echo "update md5 $md5p $md5 $old => $new"
}

function zmx-is-action() {
  local name=$1
  if [ -n "$(rg "^$name" $ZMX_BASE/actions.db)" ]; then
    echo "true"
    return
  fi
  echo "false"
}

function zmx-actions-dirty() {
  local name=$1
  local cache=$(zmx-cache-md5 $name)
  local cur=$(zmx-cur-md5 $name)
  if [[ "$cache" == "$cur" ]]; then
    echo "false"
    return
  fi
  echo "true"
}

function zmx-cache-md5() {
  local name=$1
  read name source_file line <<<$(zmx-actions-info $name)
  local md5p=$(_zmx_gen_md5p $source_file)
  # cache中可能没有
  if [ ! -f $md5p ]; then
    echo "empty-cache"
    return
  fi
  local md5=$(cat $md5p | awk '{print $1 }')
  echo $md5
}

function zmx-cur-md5() {
  local name=$1
  read name source_file line <<<$(zmx-actions-info $name)
  # actions.db中可能没有
  if [[ "$source_file" == "" ]]; then
    echo "empty-cur"
    return
  fi
  local md5=$(md5sum $source_file | awk '{print $1}')
  echo $md5
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
  if [ $(zmx-list-path | grep "$p") ]; then
    echo "already have"
    return 1
  fi
  local new_path="export SHELL_ACTIONS_PATH=\$SHELL_ACTIONS_PATH:$p"
  echo "new_path" $new_path
  echo "\n$new_path" >>~/.$(hostname).env
  # make new env work
  source ~/.loadhome.sh
  # reload
  zmx-reload-shell-actions
  zsh
}

function zmx-list-path() {
  echo $SHELL_ACTIONS_PATH | sed $'s/:/\\n/g' | sort | uniq
}
