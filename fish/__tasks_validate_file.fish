function __tasks_validate_file --description "Check if task file exists"
    set file $argv[1]
    if not test -f "$file"
        echo "Error: Task file '$file' does not exist" >&2
        return 1
    end
end