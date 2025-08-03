function __tasks_get_file --description "Get task file from argv or default to TODO.md"
    echo (test -n "$argv[1]"; and echo "$argv[1]"; or echo "TODO.md")
end