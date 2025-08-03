function tadd --description "Interactively add a task after selecting an existing one with fzf"
    set file (test -n "$argv[1]"; and echo "$argv[1]"; or echo "TODO.md")

    # Check if tasks command exists
    if not command -v tasks >/dev/null 2>&1
        echo "Error: 'tasks' command not found in PATH" >&2
        return 1
    end

    # Check if fzf exists
    if not command -v fzf >/dev/null 2>&1
        echo "Error: 'fzf' command not found in PATH" >&2
        return 1
    end

    # Check if the task file exists
    if not test -f "$file"
        echo "Error: Task file '$file' does not exist" >&2
        return 1
    end

    # Get all tasks and sections with line numbers
    set task_list (tasks --file "$file" ls 2>/dev/null)
    
    if test $status -ne 0
        echo "Error: Failed to list tasks from '$file'" >&2
        return 1
    end

    if test -z "$task_list"
        echo "No tasks found in '$file'. Adding task at the end."
        read -P "Enter task content: " task_content
        if test -n "$task_content"
            tasks --file "$file" add "$task_content"
        else
            echo "No content provided. Cancelled."
            return 1
        end
        return 0
    end

    # Use fzf to select a task to add after
    set selected_line (printf '%s\n' $task_list | fzf \
        --prompt="Select task to add after (ESC to add at end): " \
        --height=40% \
        --layout=reverse \
        --border \
        --preview-window=hidden \
        --no-multi)

    # Check if user cancelled or wants to add at the end
    if test -z "$selected_line"
        echo "Adding task at the end of the file."
        set after_id ""
    else
        # Extract the ID from the selected line (first column)
        set after_id (echo "$selected_line" | awk '{print $1}')
        
        # Validate that we got a numeric ID
        if not string match -qr '^\d+$' "$after_id"
            echo "Error: Could not extract valid ID from selection" >&2
            return 1
        end
        
        echo "Selected task ID: $after_id"
        echo "Task: $selected_line"
    end

    # Prompt for the new task content
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