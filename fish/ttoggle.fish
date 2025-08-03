function ttoggle --description "Toggle task completion status"
    set file (__tasks_get_file $argv[1])
    __tasks_check_command fzf; or return 1
    
    set task_list (__tasks_get_list $file); or return $status
    set selected_line (__tasks_fzf_select $task_list "Toggle task: ")
    
    if test -n "$selected_line"
        set task_id (__tasks_extract_id "$selected_line")
        __tasks_validate_id "$task_id"; or return 1
        
        if echo $selected_line | grep -q "\[x\]"
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