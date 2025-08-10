# Builds the project.
build:
	go build dev.rischmann.fr/tasks/v2

# Watches for changes in Go files and rebuilds the project automatically.
watch-build:
	watchexec --print-events -n -r -e go just build

# Formats the Go source code using gofmt and goimports.
fmt:
	@printf "\x1b[34m===>\x1b[m  Running go fmt\n"
	gofmt -s -w .
	@printf "\x1b[34m===>\x1b[m  Running goimports\n"
	goimports -local dev.rischmann.fr -w .

# Performs static analysis checks on the Go source code.
check:
	@printf "\x1b[34m===>\x1b[m  Running go vet\n"
	go vet ./...
	@printf "\x1b[34m===>\x1b[m  Running staticcheck\n"
	staticcheck ./...

# Generates and displays a test coverage report.
cover:
	@printf "\x1b[34m===>\x1b[m  Running test coverage\n"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@printf "\x1b[32m===>\x1b[m  Coverage report generated: coverage.html\n"
