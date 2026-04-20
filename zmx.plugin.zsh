#!/bin/zsh
export ZMX_BASE=${ZMX_BASE:-$HOME/.zmx}

function _date_now() {
    date +"%Y-%m-%0eT%T.%6N"
}

function _zmx_tools_dir() {
    echo "$ZMX_BASE/tools"
}

function _zmx_plugin_dir() {
    local plugin_file=${functions_source[_zmx_plugin_dir]}
    local plugin_dir=$(cd "$(dirname "$plugin_file")" && pwd)
    echo "$plugin_dir"
}

function _zmx_runtime_call_target() {
    echo "$(_zmx_plugin_dir)/zmx-call.sh"
}

function _zmx_ensure_runtime_tools() {
    local tools_dir=$(_zmx_tools_dir)
    local target=$(_zmx_runtime_call_target)

    mkdir -p "$ZMX_BASE" "$tools_dir"
    cat >"$tools_dir/zmx-call" <<EOF
#!/bin/bash
exec "$target" "\$@"
EOF
    chmod a+x "$tools_dir/zmx-call"
    ln -sf "$tools_dir/zmx-call" "$tools_dir/zmx-call.sh"
}


function _zmx_append_record() {
  local record="$1"
  echo "$record"
  echo "$record" >>"$ZMX_BASE/record"
}

function zmx-gen-tools() (
    local filter=$1
    _zmx_ensure_runtime_tools
    while read line;do
        echo "- $line- "
        local action=$(echo "$line"|awk '{print $1}')
        local code=$(cat <<EOF
#!/bin/bash
/home/cong/sm/project/zmx/zmx-call.sh $action
EOF
)
        echo "$code"  > ~/.zmx/tools/$action.sh
        chmod a+x ~/.zmx/tools/$action.sh
    done <  <(cat ~/.zmx/actions.db|grep $filter)
)

function zmx-gen-tools-all() (
    rm -rf ~/.zmx/tools
    _zmx_ensure_runtime_tools
    while read line;do
        echo "- $line- "
        local action=$(echo "$line"|awk '{print $1}')
        local code=$(cat <<EOF
#!/bin/bash
/home/cong/sm/project/zmx/zmx-call.sh $action
EOF
)
        echo "$code"  > ~/.zmx/tools/$action.sh
        chmod a+x ~/.zmx/tools/$action.sh
    done <  <(cat ~/.zmx/actions.db)
)

function _zmx_import_sources() {
  local import_file=${1:-$ZMX_BASE/import.sh}
  if [[ ! -f "$import_file" ]]; then
    return 1
  fi

  awk '$1 == "source" && !seen[$2]++ { print $2 }' "$import_file"
}

function _zmx_invalidate_compiled_cache() {
  rm -f "$ZMX_BASE/aio.sh" "$ZMX_BASE/aio.sh.zwc"
}

function _zmx_compiled_cache_enabled() {
  case "${ZMX_ENABLE_COMPILED_CACHE:-0:l}" in
    1|true|yes|on)
      return 0
      ;;
  esac
  return 1
}

function _zmx_compiled_cache_fresh() {
  local import_file="$ZMX_BASE/import.sh"
  local aio_file="$ZMX_BASE/aio.sh"
  local cache_file="$aio_file.zwc"
  local source_file

  if [[ ! -f "$import_file" || ! -f "$aio_file" || ! -f "$cache_file" ]]; then
    return 1
  fi

  if [[ "$import_file" -nt "$aio_file" || "$import_file" -nt "$cache_file" || "$aio_file" -nt "$cache_file" ]]; then
    return 1
  fi

  while read -r source_file; do
    if [[ -z "$source_file" ]]; then
      continue
    fi
    if [[ ! -f "$source_file" || "$source_file" -nt "$cache_file" ]]; then
      return 1
    fi
  done < <(_zmx_import_sources "$import_file")
}

function _zmx_compile() (
  local base="${ZMX_BASE:-$HOME/.zmx}"
  local import_file="$base/import.sh"
  local target_file="$base/aio.sh"
  local target_cache="$target_file.zwc"
  local temp_file="$target_file.tmp.$$"
  local temp_cache="$temp_file.zwc"
  local source_file
  local appended=0

  if [[ ! -f "$import_file" ]]; then
    echo "missing import file: $import_file"
    return 1
  fi

  echo "build compiled action cache"
  rm -f "$temp_file" "$temp_cache"
  : > "$temp_file" || return 1

  while read -r source_file; do
    if [[ -z "$source_file" ]]; then
      continue
    fi
    if [[ ! -f "$source_file" ]]; then
      echo "skip missing source $source_file"
      continue
    fi

    sed '1{/^#!/d;}' "$source_file" >> "$temp_file" || return 1
    printf '\n' >> "$temp_file" || return 1
    (( appended++ ))
  done < <(_zmx_import_sources "$import_file")

  if [[ "$appended" -eq 0 ]]; then
    rm -f "$temp_file"
    echo "no source files compiled"
    return 1
  fi

  if ! zcompile "$temp_file"; then
    rm -f "$temp_file" "$temp_cache"
    return 1
  fi

  mv "$temp_file" "$target_file" || {
    rm -f "$temp_file" "$temp_cache"
    return 1
  }

  mv "$temp_cache" "$target_cache" || {
    rm -f "$target_file" "$temp_file" "$temp_cache"
    return 1
  }

  echo "compiled action cache ready ($appended files)"
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
  local p=$(awk -v name="$f" '$1 == name { print $2; exit }' "$ZMX_BASE/actions.db")
  local p=$(readlink -f "$p")
  echo "$p"
}

function zmx-reload-shell-actions() {
  local base="$ZMX_BASE"
  local start=$(_date_now)
  local call_target=$(_zmx_runtime_call_target)

  zmx reload --base "$base" --actions-path "$SHELL_ACTIONS_PATH" --gen-path "$ZMX_GEN_PATH" --call-target "$call_target" || return
  if _zmx_compiled_cache_enabled; then
    if ! _zmx_compile; then
      echo "compiled cache unavailable, fallback to import"
      _zmx_invalidate_compiled_cache
    fi
  fi
  zmx-load-shell-actions || return

  local end=$(_date_now)
  local record="reload over, spend $(time-diff_ "$start" "$end")."
  _zmx_append_record "$record"
}

function zmx-list() {
  cat ~/.zmx/actions.db | awk '{print $1}'
}

function zmx-list-actions-from-zsh() {
   local fn
   local i
   local source_file
   local -a function_items
   local -A allowed_sources

   if _zmx_compiled_cache_enabled && [[ -f "$ZMX_BASE/aio.sh" ]]; then
     allowed_sources[$ZMX_BASE/aio.sh]=1
   fi

   while read -r source_file; do
     if [[ -n "$source_file" ]]; then
       allowed_sources[$source_file]=1
     fi
   done < <(_zmx_import_sources)

   function_items=("${(@kv)functions_source}")
   for (( i = 1; i <= ${#function_items}; i += 2 )); do
     fn=${function_items[i]}
     source_file=${function_items[i + 1]}
     if [[ -n "${allowed_sources[$source_file]}" ]]; then
       print -- "$fn"
     fi
   done | sort -u
}

function zmx-help() {
  local f=${1-$(zmx-list|fzf)}
  if [[ -z "$f" ]]; then
    echo "no function selected"
    return 1
  fi

  echo "Function: $f"
  echo ""

  # 提取 @@@ 之间的帮助文本
  local help_text=$(which "$f" | sed -n '/^@@@$/,/^@@@$/p' | sed '1d;$d')
  echo "$help_text"
}

function time-diff_() {
    if command -v time-diff >/dev/null 2>&1; then
        time-diff "$@"
    else
        echo "n/a"
    fi
}

function zmx-log() {
    echo "$(_date_now) $@" >>$ZMX_BASE/.zmx.log
}

function zmx-watch-log() {
    tail -F $ZMX_BASE/.zmx.log
}

function zmx-load-shell-actions() {
  local start=$(_date_now)

  _zmx_ensure_runtime_tools || return
  echo "start source"
  if [[ ! -f "$ZMX_BASE/import.sh" ]]; then
    echo "no actions found ignore"
    return
  fi

  if _zmx_compiled_cache_enabled; then
    if ! _zmx_compiled_cache_fresh; then
      echo "compiled cache stale, rebuild"
      _zmx_invalidate_compiled_cache
      if ! _zmx_compile; then
        echo "compiled cache unavailable, fallback to import"
        _zmx_invalidate_compiled_cache
      fi
    fi
  fi

  if _zmx_compiled_cache_enabled && _zmx_compiled_cache_fresh; then
    echo "load actions from cache"
    source "$ZMX_BASE/aio.sh"
  else
    echo "load actions from import"
    source "$ZMX_BASE/import.sh"
  fi
  echo "end source $?"

  local count=$(count-actions)
  local fn_count=$(zmx-list-actions-from-zsh | wc -l)
  local end=$(_date_now)
  local record="load over, actions-db-fn $count zsh-fn $fn_count spend $(time-diff_ "$start" "$end")."
  _zmx_append_record "$record"
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
  local spec_doc='
@@@
name: zmx-actions-info
summary: Show the raw registry row for an indexed action

arg action | required | desc=Action name
@@@
'

  if shellargs is-help -- "$@"; then
    shellargs help --spec "$spec_doc"
    return
  fi

  local parsed=$(shellargs parse --spec "$spec_doc" -- "$@") || return
  local name=$(jq -r '.action' <<<"$parsed") || return
  awk -v name="$name" '$1 == name { print; exit }' "$ZMX_BASE/actions.db"
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
  if awk -v name="$name" '$1 == name { found = 1; exit } END { exit found ? 0 : 1 }' "$ZMX_BASE/actions.db" 2>/dev/null; then
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
