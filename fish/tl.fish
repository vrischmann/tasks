function tl --description "List all tasks"
    set file (test -n "$argv[1]"; and echo "$argv[1]"; or echo "TODO.md")
    tasks --file $file ls
end