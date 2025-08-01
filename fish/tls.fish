function tls --description "List incomplete tasks only"
    set file (test -n "$argv[1]"; and echo "$argv[1]"; or echo "TODO.md")
    tasks --file $file ls | grep "\[ \]"
end