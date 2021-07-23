echo "load zmx"
mx() {
    cmd=$(print -rl ${(k)functions_source[(R)*awesome*]} | fzf)
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