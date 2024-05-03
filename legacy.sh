# 太复杂了 不应该这么搞

# function zmx-list-current() {
#     fd  '(action.*\.sh|loop.sh)' $PWD
# }

# function zmx-load-current() {
#     while read -r line ;do
#         source $line
#     done < <(fd  '(action.*\.sh|loop.sh)' $PWD)
# }

# function _zmx_gen_md5p() {
#   local file=$1
#   local md5p="$ZMX_BASE/md5/$(echo $file | sed 's|/|_|g').md5"
#   echo $md5p
# }

# function zmx-upate-md5() {
#   local name=$1
#   read name source_file line <<<$(zmx-actions-info $name)
#   local old=$(zmx-cache-md5 $name)
#   local md5p=$(_zmx_gen_md5p $source_file)
#   local md5=$(zmx-cur-md5 $name)
#   echo $md5 >$md5p
#   local new=$(zmx-cache-md5 $name)
#   echo "update md5 $md5p $md5 $old => $new"
# }


# function zmx-actions-dirty() {
#   local name=$1
#   local cache=$(zmx-cache-md5 $name)
#   local cur=$(zmx-cur-md5 $name)
#   if [[ "$cache" == "$cur" ]]; then
#     echo "false"
#     return
#   fi
#   echo "true"
# }

# function zmx-cache-md5() {
#   local name=$1
#   read name source_file line <<<$(zmx-actions-info $name)
#   local md5p=$(_zmx_gen_md5p $source_file)
#   # cache中可能没有
#   if [ ! -f $md5p ]; then
#     echo "empty-cache"
#     return
#   fi
#   local md5=$(cat $md5p | awk '{print $1 }')
#   echo $md5
# }

# function zmx-cur-md5() {
#   local name=$1
#   read name source_file line <<<$(zmx-actions-info $name)
#   # actions.db中可能没有
#   if [[ "$source_file" == "" ]]; then
#     echo "empty-cur"
#     return
#   fi
#   local md5=$(md5sum $source_file | awk '{print $1}')
#   echo $md5
# }

# function zmx-is-full-reload() {
#   if [ -f "$ZMX_BASE/full-reload" ]; then
#     echo "true"
#     return
#   fi
#   echo "false"
# }
# function zmx-enable-full-reload() {
#   touch $ZMX_BASE/full-reload
# }

# function zmx-disable-full-reload() {
#   rm $ZMX_BASE/full-reload
# }

# # may be in dev lopp
# function _zmx_dev() {
#   if [[ ! "$(zmx-is-full-reload)" == "true" ]]; then
#     return
#   fi
#   echo "force reload"
#   zmx-reload-shell-actions
# }
