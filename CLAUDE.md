# Super-Bot Development Guide

## Build Commands
- Build: `make build`
- Test all: `make test`
- Test single: `go test -v ./app/bot/path/to/specific_test.go`
- Run tests with coverage: `go test ./app/... -coverprofile cover.out`
- Lint: `make lint` (requires golangci-lint)
- Run locally: `make run ARGS="--super=username --dbg"`
- Generate mocks: `make generate`

## Code Style Guidelines
- Go version: 1.21+
- Format with gofmt/goimports
- Error handling: check all errors, log with proper level [ERROR], [INFO], [DEBUG]
- Use lgr package for structured logging
- Testing: use testify for assertions and mocks
- Config: use go-flags for CLI args and env vars with proper descriptions
- Imports: group standard lib, 3rd party, and internal packages
- Naming: use CamelCase for exported, camelCase for private
- Comments: all comments inside functions must be lowercase

## Clean Code Principles
- Follow interfaces for testability (see bot.Interface)
- Use Context for cancellation
- Unit test coverage should be maintained
- Error messages should be descriptive and actionable