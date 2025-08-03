function __tasks_get_list --description "Get task list with error handling"
    set file $argv[1]
    set task_list (tasks --file "$file" ls 2>/dev/null)
    
    if test $status -ne 0
        echo "Error: Failed to list tasks from '$file'" >&2
        return 1
    end
    
    if test -z "$task_list"
        echo "No tasks found in '$file'." >&2
        return 2
    end
    
    printf '%s\n' $task_list
end