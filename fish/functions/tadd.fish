function tadd --description "Interactively add a task after selecting an existing one with fzf"
    set file (__tasks_get_file $argv[1])
    
    # Check dependencies
    __tasks_check_command tasks; or return 1
    __tasks_check_command fzf; or return 1
    __tasks_validate_file "$file"; or return 1

    # Get all tasks and sections with line numbers
    set task_list (__tasks_get_list "$file")
    set list_status $status
    
    if test $list_status -eq 2
        # No tasks found - add at end
        echo "No tasks found in '$file'. Adding at the end."
        read -P "Enter task content: " task_content
        if test -n "$task_content"
            tasks --file "$file" add "$task_content"
        else
            echo "No content provided. Cancelled."
            return 1
        end
        return 0
    else if test $list_status -ne 0
        return $list_status
    end

    # Use fzf to select a task to add after
    set selected_line (printf '%s\n' $task_list | fzf \
        --prompt="Select task to add after (ESC to add at end): " \
        --layout=reverse \
        --border \
        --preview-window=hidden \
        --no-multi)

    # Check if user cancelled or wants to add at the end
    if test -z "$selected_line"
        echo "Adding at the end of the file."
        set after_id ""
    else
        # Extract and validate the ID
        set after_id (__tasks_extract_id "$selected_line")
        __tasks_validate_id "$after_id"; or return 1
        
        echo "Selected task ID: $after_id"
        echo "Task: $selected_line"
    end

    # Ask if user wants to add a task or section
    echo "What would you like to add?"
    echo "1) Task"
    echo "2) Section"
    read -P "Enter choice (1 or 2, default: 1): " item_type
    
    # Default to task if empty or invalid input
    if test -z "$item_type"; or test "$item_type" != "2"
        set item_type "1"
    end
    
    if test "$item_type" = "2"
        # Adding a section
        echo "Section levels: 1 (# Header), 2 (## Header), 3 (### Header), 4 (#### Header), 5 (##### Header), 6 (###### Header)"
        read -P "Enter section level (1-6, default: 1): " section_level
        
        # Validate section level
        if test -z "$section_level"
            set section_level "1"
        else if not string match -qr '^[1-6]$' "$section_level"
            echo "Invalid section level. Using level 1."
            set section_level "1"
        end
        
        read -P "Enter section title: " section_content
        
        if test -z "$section_content"
            echo "No content provided. Cancelled."
            return 1
        end
        
        # Add the section
        if test -n "$after_id"
            echo "Adding section (level $section_level) after ID $after_id..."
            tasks --file "$file" add --after "$after_id" --section --level "$section_level" "$section_content"
        else
            echo "Adding section (level $section_level) at the end..."
            tasks --file "$file" add --section --level "$section_level" "$section_content"
        end
    else
        # Adding a task
        read -P "Enter new task content: " task_content

        if test -z "$task_content"
            echo "No content provided. Cancelled."
            return 1
        end

        # Add the task
        if test -n "$after_id"
            echo "Adding task after ID $after_id..."
            tasks --file "$file" add --after "$after_id" "$task_content"
        else
            echo "Adding task at the end..."
            tasks --file "$file" add "$task_content"
        end
    end

    if test $status -eq 0
        echo "Task added successfully!"
        
        # Optionally show the updated incomplete tasks
        echo ""
        echo "Updated incomplete tasks:"
        tlist "$file"
    else
        echo "Error: Failed to add task" >&2
        return 1
    end
end