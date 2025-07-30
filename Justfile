build:
	go build dev.rischmann.fr/tasks

run *ARGS:
	go run dev.rischmann.fr/tasks test.md

watch-build:
	watchexec --print-events -n -r -e go just build

watch-run:
	watchexec --print-events -n -r -e go just run

fmt:
	@printf "\x1b[34m===>\x1b[m  Running go fmt\n"
	go fmt ./...

check:
	@printf "\x1b[34m===>\x1b[m  Running go vet\n"
	go vet ./...
	@printf "\x1b[34m===>\x1b[m  Running staticcheck\n"
	staticcheck ./...
