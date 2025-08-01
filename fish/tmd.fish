function tmd --description "Mark multiple tasks as done"
    set file (test -n "$argv[1]"; and echo "$argv[1]"; or echo "TODO.md")
    set task_lines (tasks --file $file ls | grep "\[ \]" | fzf -m --prompt="Select tasks to complete: ")
    
    if test -n "$task_lines"
        for task_line in $task_lines
            set task_id (echo $task_line | awk '{print $1}')
            tasks --file $file done $task_id
            echo "Task $task_id marked as done"
        end
    end
end