
echo "load zmx"


function zmx_preexec() {
	# 如果是zmx的action的话，如果是dirty的，先source一下
	local cmd=$(echo $1 | awk '{ print $1}')
	if type "$cmd" |grep -v -q 'shell-action' ; then
		return
	fi
	local p=$(type -a $cmd |rg -o 'from (.*)$' -r '$1')
	local md5p=$(echo $p | sed "s shell-actions shell-actions-md5 ").md5
	local pmd5=$(md5sum $p | awk '{ print $1}')
	local md5pp=$(cat $md5p)
	if [ "$pmd5" != "$md5pp" ] ; then
		source $p
		echo  $pmd5 > $md5p
		echo "source and update md5 $md5p $pmd5"
	fi
}

function edit-x-actions() {
    cmd=$(list-x-actions|fzf)
    source_file=$(type $cmd|rg -o '.* from (.*)' -r '$1'  |tr -d '\n\r')

    cmd_start_line=$(grep -no "$cmd()" $source_file |cut -d ':' -f 1 |tr -d '\n\r')
    echo $source_file
    echo $cmd_start_line
    # @keyword: vim edit file in special line
    vim +$cmd_start_line $source_file
}

function which-x-actions() {
    cmd=$(list-x-actions|fzf)
    which $cmd
}

function list-x-actions() {
    print -rl ${(k)functions_source[(R)*shell-actions*]}
}

function count-actions() {
    print -rl ${(k)functions_source[(R)*shell-actions*]} |wc -l
}

function function date-ms() {
	date +"%Y %m %e %T.%6N"
}

function function time-diff-ms() {
	local start=$1
	local end=$2

	local output=$( bash <<-EOF
	python3 - <<-START
		from datetime import datetime
		import humanize
		start = datetime.strptime("$start","%Y %m %d %H:%M:%S.%f")
		end = datetime.strptime("$end","%Y %m %d %H:%M:%S.%f")
		print(humanize.precisedelta(end-start, minimum_unit="microseconds"))
	START
	EOF
	)
	echo $output
}

function zmx-find-base-of-action() {
	local f=$1
	if [ -z "$f" ] ; then
		f=$(print -l $functrace | head -n 1 | cut -d ':' -f 1)
	fi
	local p=$(type -a $f |rg -o 'from (.*)$' -r '$1')
	local p=$(readlink -f $p)
	echo $(dirname "$p")
}

function zmx-find-path-of-action() {
	local f=$1
	if [ -z "$f" ] ; then
		f=$(print -l $functrace | head -n 1 | cut -d ':' -f 1)
	fi
	local p=$(type -a $f |rg -o 'from (.*)$' -r '$1')
	local p=$(readlink -f $p)
	echo "$p"
}

function zmx-load-shell-actions() {
    # local actions_path=$SHELL_ACTIONS_PATH
    # echo "start load " $actions_path
    local start=$(date-ms)
    echo "start source"
    # echo $actions_path 
    if [ ! -f ~/.zsh/actions.sh ]; then
        echo "no actions found ignore"
        return
    fi
    source ~/.zsh/actions.sh
    echo "end source $?"
    for action in $(print -rl ${(k)functions_source[(R)*shell-actions*]});do 
        short=$(echo $action | sed 's/-//g')
        alias $short=$action
    done
    local count=$(count-actions)
    local end=$(date-ms)
    echo "load $count actions,spend $(time-diff-ms "$start" "$end")."
}

function zmx-reload-shell-actions() {
    echo "action path" $SHELL_ACTIONS_PATH
    local actions_path=$SHELL_ACTIONS_PATH
    local summary_path=~/.zsh/actions.sh
    local index_path=~/.zsh/shell-actions
    # clear
    rm -rf ~/.zsh/shell-actions-md5
    rm -rf $index_path
    rm -rf $summary_path

    mkdir -p  $index_path
    mkdir -p  ~/.zsh/shell-actions-md5
    touch $summary_path
    echo "start index"
    for p in $(echo $actions_path| sed "s/:/ /g")
    do
		local link=$(echo $p|sed 's/\//_/g')
        echo index $p $link
        ln -s $p  $index_path/$link
    done
    echo "end index"
    # 直接以sh而不是文件夹形式加进来的
	while IFS= read -r sh_path
	do
        echo "add $sh_path"
        echo "source $sh_path" >> $summary_path
	done <<< "$(fd -L --glob '*.*sh' $index_path)"
    echo "end summary"
	# generated md5
	fd -L --glob "*.sh" $index_path -x bash -c 'md5=$(md5sum {}| cut -d " " -f 1);p=$(echo {} | sed "s shell-actions shell-actions-md5 " );mkdir -p $(dirname $p) ;echo $md5 > $p.md5'
    echo "end md5 cache"
    zmx-load-shell-actions
}

function zmx-add-path() {
	local p=$(readlink -f $1)
	if [ -z "$p" ]; then
		echo "empty path"
		return
	fi
	local new_path="export SHELL_ACTIONS_PATH=\$SHELL_ACTIONS_PATH:$p"
	echo "new_path" $new_path
	echo "\n$new_path"  >>  ~/.$(hostname).env
	# make new env work
	source ~/.loadhome.sh
	# reload
	zmx-reload-shell-actions
	zsh
}

function mx() {
    cmd=$(print -rl ${(k)functions_source[(R)*shell-actions*]} |grep -v _ | fzf)
    source_file=$(echo $functions_source[$cmd])
    if grep "$cmd" $source_file -A 1 |grep -q 'arg-len'; then
        LBUFFER+=$cmd
        LBUFFER+=" "
        zle reset-prompt
    else
        eval $cmd
        zle reset-prompt
    fi
}

function lmx() {
    # mx local action
    local source_file=$(fd -a actions.sh ./)
    echo "source" $source_file
    local cmd=$(cat $source_file |rg "^\s*function\s(.*)\s*\{$" -r  '$1' |grep -v '_.*'| fzf --preview "grep  {} $source_file -A 5" )
    local cmd=$(echo $cmd|sed "s/()\s//g")
    echo "cmd $cmd"
    if [ -z "$cmd" ] ; then
        echo "empty cmd ignore"
        zle reset-prompt
        return
    fi
    local annotation=$(grep "function\s$cmd" $source_file -A 1)
    echo "arg-annno $annotation"
    if  echo $annotation |grep -q 'arg-len'; then
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

function zmx-bind-key() {
    zle -N  mx
    zle -N  lmx
    bindkey ',xx' lmx
    bindkey ',xm' mx
}


function zmx-gen-wapper() {
    mkdir -p ~/.zsh/shell-actions-wapper
    local total=$(list-x-actions | wc -l)
    local index=1
    for action in $(print -rl ${(k)functions_source[(R)*shell-actions*]});do 
        echo "
       $action "\$@"
" > ~/.zsh/shell-actions-wapper/$action.sh
    chmod a+x ~/.zsh/shell-actions-wapper/$action.sh
        echo init $action $index/$total
        ((index=index+1))
    done
}