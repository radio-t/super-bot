# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

- Build: `make build` or `cd app && go build -v`
- Test all: `make test` or `go test ./app/... -coverprofile cover.out`
- Test single package: `go test -v ./app/bot/` or `go test -v -run TestFuncName ./app/bot/`
- Lint: `make lint` or `golangci-lint run ./app/...`
- Generate mocks: `make generate` or `go generate ./app/...`
- Run locally: `make run ARGS="--super=username --dbg"` (requires `TELEGRAM_TOKEN` and `TELEGRAM_GROUP` env vars)

## Architecture

This is a **Telegram bot for the Radio-T podcast** (`github.com/radio-t/super-bot`). Go 1.24.

### Core Pattern: Plugin Bot Architecture

All bots implement the `bot.Interface` (defined in `app/bot/bot.go`):

```go
type Interface interface {
    OnMessage(msg Message) Response
    ReactOn() []string   // trigger keywords
    Help() string
}
```

`MultiBot` aggregates all bots and dispatches messages **in parallel** (4 goroutines via `syncs.SizedGroup`). Responses are collected, merged, and **sorted alphabetically** — multiple bots can respond to the same message.

### Message Flow

```
Telegram Group → TelegramListener.Do() (app/events/telegram.go)
  → Transform tbapi.Message → bot.Message
  → Check terminators (rate limiting / ban enforcement)
  → MultiBot.OnMessage() (parallel dispatch to all bots)
  → Send response back to Telegram (markdown with plain-text fallback)
  → Log to reporter (JSON lines, one file per day in logs/)
  → Execute response flags (ban, pin, unpin, delete)
```

External messages also arrive via **RTJC TCP listener** (`app/events/rtjc.go`) on port 18001 from news.radio-t.com.

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| Entry point | `app/main.go` | Config parsing, wiring, startup |
| Bot interface & MultiBot | `app/bot/bot.go` | Plugin interface + parallel dispatch |
| Individual bots | `app/bot/*.go` | WTF, News, Anecdote, DuckDuckGo, When, Banhammer, Sys, Spam, etc. |
| OpenAI/ChatGPT bot | `app/bot/openai/` | Chat with history context, summarizer, Remark42/uReadability integration |
| Telegram listener | `app/events/telegram.go` | Main event loop, message routing, ban enforcement |
| RTJC listener | `app/events/rtjc.go` | TCP listener for external message submissions |
| Terminators | `app/events/terminator.go` | Three rate limiters: all-activity, bot-activity, overall-bot-activity |
| SuperUser | `app/events/superuser.go` | Moderator privilege checks (exempt from rate limiting) |
| Reporter/Logger | `app/reporter/reporter.go` | Non-blocking JSON message persistence, batched writes |
| HTML Exporter | `app/reporter/export.go` | Generates HTML chat reports from daily logs |
| Data files | `data/` | `basic.data` (trigger→response pairs), `say.data` (random quotes), `whatsthetime.data`, `logs.html` template |

### Important Design Decisions

- **No database** — all persistence is file-based JSON lines (one file per day: `logs/YYYYMMDD.log`)
- **Channel banning** — detects messages from channels (ID 136817688) and permanently bans the channel, not the fake user
- **Markdown fallback** — if Telegram rejects markdown, automatically resends as plain text
- **SuperUsers** are exempt from rate limiting and spam checks but not from admin bans
- **OpenAI bot** maintains a rolling message history (default 5) for context-aware responses; has separate modes for direct queries (`chat!/gpt!/ai!`) and probabilistic auto-responses
- **Export mode** — passing `--export-num` flag runs HTML report generation instead of the bot listener

## Code Style

- Logging: use `lgr` package with level prefixes `[ERROR]`, `[INFO]`, `[DEBUG]`
- Config: `go-flags` with CLI args and env vars (e.g., `--telegram-token` / `TELEGRAM_TOKEN`)
- Testing: `testify` for assertions, `moq` for mock generation (`//go:generate moq ...`)
- Comments inside functions must be lowercase
- HTTP clients use retry middleware via `go-pkgz/requester` (10 attempts, 5s intervals) for external API calls

## OpenAI Bot Implementation Notes

- The OpenAI bot maintains message history to track conversation context
- Direct queries (with `chat!/gpt!/ai!/чат!` prefixes) use history while focusing on the current message
- The `chatGPTRequestWithHistoryAndFocus` function balances context awareness with response relevance
- History size is configurable via the `--history-size` flag (default: 5 messages)
