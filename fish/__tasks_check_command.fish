function __tasks_check_command --description "Check if a command exists in PATH"
    set cmd $argv[1]
    if not command -v $cmd >/dev/null 2>&1
        echo "Error: '$cmd' command not found in PATH" >&2
        return 1
    end
end