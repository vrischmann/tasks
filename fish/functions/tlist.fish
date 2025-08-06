function tlist --description "List incomplete tasks only"
    set file (__tasks_get_file $argv[1])
    tasks --color=always --file $file ls | grep "\[ \]"
end
