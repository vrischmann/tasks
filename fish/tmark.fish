function tmark --description "Mark multiple tasks as done"
    set file (__tasks_get_file $argv[1])
    __tasks_check_command fzf; or return 1
    
    set incomplete_tasks (tasks --file $file ls | grep "\[ \]")
    if test -z "$incomplete_tasks"
        echo "No incomplete tasks found in '$file'."
        return 0
    end
    
    set task_lines (__tasks_fzf_select_multi $incomplete_tasks "Select tasks to complete: ")

    if test -n "$task_lines"
        for task_line in $task_lines
            set task_id (__tasks_extract_id "$task_line")
            if __tasks_validate_id "$task_id"
                tasks --file $file done $task_id
                echo "Task $task_id marked as done"
            end
        end
    end
end
