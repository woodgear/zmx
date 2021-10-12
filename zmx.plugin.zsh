echo "load zmx"


source_it() {
    local p=$1
    if [ -f "$p" ]; then
        if [[ $p == *.sh ]]; then
            echo "source file $p"
            . $p
            if [[ $? -ne 0 ]]; then
             echo "source $p fail"
            fi
        fi
    fi

    if [ -d "$p" ]; then
        echo "source dir $p start"
        for file in $p/*; do
            source_it "$file"
        done
        echo "source dir $p over"
    fi
}

awesome-shell-actions-load() {
    echo "load start"
    awesome_shell_actions_path=$1
    if [ -d $awesome_shell_actions_path ] 
    then 
        echo "find awesome-shell-actions in ${awesome_shell_actions_path} start load"
        source_it $awesome_shell_actions_path/scripts
        if [[ $? -ne 0 ]]; then
            echo "source actions fail"
        fi
        for action in $(print -rl ${(k)functions_source[(R)*awesome*]});do 
            short=$(echo $action | sed 's/-//g')
            alias $short=$action
        done
    else
        echo "cloud not find awesome-shell-actions in $awesome_shell_actions_path ignore"
    fi
    echo "load over"
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

load_awesome_shell_actions() {
    awesome_shell_actions_path=$1
    if [ -d $awesome_shell_actions_path ] 
    then 
        . $awesome_shell_actions_path/scripts/awesome-shell-actions.sh
        awesome-shell-actions-load $awesome_shell_actions_path

    else
        echo "cloud not find awesome-shell-actions in $awesome_shell_actions_path ignore"
    fi
}

load_shell_actions() {
    local actions_path=$1
    echo "start load"
    echo $actions_path 
    rm -rf ~/.zsh/shell-actions
    mkdir -p  ~/.zsh/shell-actions
    for p in $(echo $actions_path| sed "s/:/ /g")
    do
       ln -s $p  ~/.zsh/shell-actions
       source_it ~/.zsh/shell-actions
    done
}

mx() {
    cmd=$(print -rl ${(k)functions_source[(R)*shell-actions*]} | fzf)
    source_file=$(echo $functions_source[$cmd])
    if grep "$cmd" $source_file -A 1 |grep -q 'arg-len'; then
        set_args_doc $cmd
        LBUFFER+=$cmd
        LBUFFER+=" "
        zle reset-prompt
    else
        eval $cmd
        zle reset-prompt
    fi
}

zle -N  mx
bindkey ',xm' mx