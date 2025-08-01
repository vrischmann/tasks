function tr --description "Remove task interactively"
    set file (test -n "$argv[1]"; and echo "$argv[1]"; or echo "TODO.md")
    set task_line (tasks --file $file ls | fzf --prompt="Remove task: ")
    
    if test -n "$task_line"
        set task_id (echo $task_line | awk '{print $1}')
        echo "Remove: "(echo $task_line | cut -d' ' -f3-)
        read -P "Are you sure? (y/N): " confirmation
        
        if test "$confirmation" = "y" -o "$confirmation" = "Y"
            tasks --file $file rm $task_id
            echo "Task $task_id removed"
        else
            echo "Cancelled"
        end
    end
end