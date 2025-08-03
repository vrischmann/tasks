function __tasks_fzf_select_multi --description "Common fzf multi-task selection"
    set task_list $argv[1..-2] 
    set prompt $argv[-1]
    
    printf '%s\n' $task_list | fzf \
        --prompt="$prompt" \
        --layout=reverse \
        --border \
        --preview-window=hidden \
        --tac \
        --no-sort \
        --bind 'space:toggle' \
        -m
end