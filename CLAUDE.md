# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a minimal Go project with a basic module structure. The project is in its initial state with only a main package containing an empty main function.

## Development Commands

### Build and Run
```bash
go build
./tasks
```

### Run directly
```bash
go run main.go
```

### Format code
```bash
go fmt ./...
```

### Test (when tests are added)
```bash
go test ./...
```

### Module management
```bash
go mod tidy    # Clean up dependencies
go mod download # Download dependencies
```

## Code Architecture

- **Module**: `dev.rischmann.fr/tasks`
- **Go Version**: 1.24.5
- **Structure**: Single-file application with main package in `main.go`

The project currently has a minimal structure suitable for a simple CLI tool or service. When expanding the codebase, consider organizing code into packages based on functionality.