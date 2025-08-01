function tt --description "Toggle task completion status"
    set file (test -n "$argv[1]"; and echo "$argv[1]"; or echo "TODO.md")
    set task_line (tasks --file $file ls | fzf --bind 'space:toggle' --prompt="Toggle task: ")
    
    if test -n "$task_line"
        set task_id (echo $task_line | awk '{print $1}')
        if echo $task_line | grep -q "\[x\]"
            # Task is completed, mark as incomplete
            tasks --file $file undo $task_id
            echo "Task $task_id marked as incomplete"
        else
            # Task is incomplete, mark as completed
            tasks --file $file done $task_id
            echo "Task $task_id marked as done"
        end
    end
end