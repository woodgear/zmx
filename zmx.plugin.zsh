
echo "load zmx"
init-actions() {
	local p=$1
	# source sh actions
	while IFS= read -r sh_path
	do
    	echo -- source "$sh_path"--
		source $sh_path
	done <<< "$(fd -L '.*\.sh' $p)"

	# source generated sh actions
	for c in $(seq 5)
	do
		while IFS= read -r sh_path
		do
			if [ -z "$sh_path" ]; then
				continue
			fi
    		echo -- gen $c source "$sh_path"--
			# source $sh_path
		done <<< "$(fd -L '.*\.actions\.gen$c\.sh' $p)"
	done
}


edit-x-actions() {
    cmd=$(list-x-actions|fzf)
    source_file=$(type $cmd|rg -o '.* from (.*)' -r '$1'  |tr -d '\n\r')

    cmd_start_line=$(grep -no "$cmd()" $source_file |cut -d ':' -f 1 |tr -d '\n\r')
    echo $source_file
    echo $cmd_start_line
    # @keyword: vim edit file in special line
    vim +$cmd_start_line $source_file
}

which-x-actions() {
    cmd=$(list-x-actions|fzf)
    which $cmd
}

list-x-actions() {
    print -rl ${(k)functions_source[(R)*shell-actions*]}
}

count-actions() {
    print -rl ${(k)functions_source[(R)*shell-actions*]} |wc -l
}

function date-ms() {
	date +"%Y %m %e %T.%6N"
}

function time-diff-ms() {
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

zmx-find-path-of-action() {
	# local p=$1
	# echo "$p"
    local f=$(print -l $functrace | head -n 1 | cut -d ':' -f 1)
	local p=$(type -a $f |rg -o 'from (.*)$' -r '$1')
	local p=$(readlink -f $p)
	echo "$p"
}

zmx-load-shell-actions() {
    local actions_path=$SHELL_ACTIONS_PATH
    echo "start load " $actions_path
    local start=$(date-ms)
    echo $actions_path 
    init-actions ~/.zsh/shell-actions
    for action in $(print -rl ${(k)functions_source[(R)*shell-actions*]});do 
        short=$(echo $action | sed 's/-//g')
        alias $short=$action
    done
    local count=$(count-actions)
    local end=$(date-ms)
    echo "load $count actions,spend $(time-diff-ms "$start" "$end")."
}

zmx-reload-shell-actions() {
    echo "action path" $SHELL_ACTIONS_PATH
    local actions_path=$SHELL_ACTIONS_PATH
    rm -rf ~/.zsh/shell-actions
    mkdir -p  ~/.zsh/shell-actions
    for p in $(echo $actions_path| sed "s/:/ /g")
    do
		local link=$(echo $p|sed 's/\//_/g')
        echo index $p $link
		
        ln -s $p  ~/.zsh/shell-actions/$link
    done
}

zmx-add-path() {
	local p=$(readlink -f $1)
	if [ -z "$p" ]; then
		echo "empty path"
		return
	fi
	local new_path="export SHELL_ACTIONS_PATH=\$SHELL_ACTIONS_PATH:$p"
	echo "new_path" $new_path
	echo "\n$new_path"  >>  ~/.$(hostname).env
	# reload
	zmx-reload-shell-actions
	zsh
}

mx() {
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

lmx() {
    # mx local action
    local source_file=$(fd -a actions.sh ./)
    echo "source" $source_file
    local cmd=$(cat $source_file |rg "^\s*function\s(.*)\s*\{$" -r  '$1' |grep -v '_.*'| fzf --preview "grep  {} $source_file -A 5" )
    echo "cmd $cmd"
    if [ -z "$cmd" ] ; then
        echo "empty cmd ignore"
        zle reset-prompt
        return
    fi
    if grep "$cmd" $source_file -A 1 |grep -q 'arg-len'; then
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

zle -N  mx
zle -N  lmx
bindkey ',xx' lmx
bindkey ',xm' mx
