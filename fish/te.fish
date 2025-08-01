function te --description "Edit task interactively"
    set file (test -n "$argv[1]"; and echo "$argv[1]"; or echo "TODO.md")
    set task_id (tasks --file $file ls | fzf --prompt="Edit task: " | awk '{print $1}')
    
    if test -n "$task_id"
        tasks --file $file edit $task_id
    end
end