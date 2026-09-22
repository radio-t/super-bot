# Super-Bot Development Guide

## Build Commands
- Build: `make build` or `go build -v -o super-bot ./app`
- Test all: `make test` or `go test ./app/...`
- Test single: `go test -v ./app/bot/path/to/specific_test.go`
- Run tests with coverage: `go test ./app/... -coverprofile cover.out`
- Lint: `make lint` or `golangci-lint run ./app/...`
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

## OpenAI Bot Implementation Notes
- History records only group messages (`ChatID < 0`): idle ticks carry ChatID 0, private chats are positive
- `LimitedMessageHistory.snapshot(cur, now)` builds the context for both OpenAI builders and jev: the current
  message comes from the caller (never the buffer tail), preceding entries only from `cur.ChatID`, 30-minute
  age limit, 1000/8000 rune caps; the current text is never truncated. Default history size is 10
- Unsolicited replies go through `shouldJoin` (`gate.go`): three jev nouls (invites, answerable, spam) with
  thresholds as constants; any jev error or invalid answer means no reply. `autoReplyLimits` holds the
  15-minute cooldown, 10/day cap and 150 jev calls/hour, separate from `lastDT` (direct-request bans)
- The jev client lives in `app/bot/openai/jev` because tests are `package openai` and import `mocks`; a mocked
  interface using types from package `openai` would create an import cycle
- Thresholds were tuned on labeled production cases (contract in `gate_test.go`, replay via
  `JEV_LIVE_CASES`); retune only with fresh labeled data, never against the old holdout
- `isReasoningModel` matches o1/o3/o4 and the gpt-5/gpt-6 families; those models reject `max_tokens`