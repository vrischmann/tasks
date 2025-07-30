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

cover:
	@printf "\x1b[34m===>\x1b[m  Running test coverage\n"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@printf "\x1b[32m===>\x1b[m  Coverage report generated: coverage.html\n"
