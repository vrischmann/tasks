function __tasks_fzf_select --description "Common fzf task selection with customizable prompt"
    set task_list $argv[1..-2]
    set prompt $argv[-1]
    
    printf '%s\n' $task_list | fzf \
        --prompt="$prompt" \
        --layout=reverse \
        --border \
        --preview-window=hidden \
        --no-sort \
        --bind 'space:toggle' \
        --no-multi
end