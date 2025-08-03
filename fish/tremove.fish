function tremove --description "Remove task interactively"
    set file (__tasks_get_file $argv[1])
    __tasks_check_command fzf; or return 1
    
    set task_list (__tasks_get_list $file); or return $status
    set selected_line (__tasks_fzf_select $task_list "Remove task: ")
    
    if test -n "$selected_line"
        set task_id (__tasks_extract_id "$selected_line")
        __tasks_validate_id "$task_id"; or return 1
        
        echo "Remove: "(echo $selected_line | cut -d' ' -f3-)
        read -P "Are you sure? (y/N): " confirmation
        
        if test "$confirmation" = "y" -o "$confirmation" = "Y"
            tasks --file $file rm $task_id
            echo "Task $task_id removed"
        else
            echo "Cancelled"
        end
    end
end