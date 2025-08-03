function __tasks_extract_id --description "Extract task ID from fzf selection"
    echo "$argv[1]" | awk '{print $1}'
end