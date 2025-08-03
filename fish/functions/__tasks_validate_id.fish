function __tasks_validate_id --description "Validate that task ID is numeric"
    set task_id $argv[1]
    if not string match -qr '^\d+$' "$task_id"
        echo "Error: Could not extract valid ID from selection" >&2
        return 1
    end
end