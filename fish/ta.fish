function ta --description "Add task quickly"
    set file (test -n "$argv[2]"; and echo "$argv[2]"; or echo "TODO.md")
    if test -n "$argv[1]"
        tasks --file $file add "$argv[1]"
        echo "Added: $argv[1]"
    else
        echo "Usage: ta 'task description' [file]"
    end
end