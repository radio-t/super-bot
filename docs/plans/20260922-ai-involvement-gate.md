# AI Involvement Gate for Unsolicited Replies

## Overview
- the OpenAI bot's unsolicited replies feel random and out of context. It fires when a message ends
  with `?` and a 7% dice roll passes, and it answers from the last 5 messages sent as bare text with
  no authors, where synthetic `idle` ticks can take every slot
- replace the dice with a jev (typesafe.ai) decision per candidate message, repair the history the
  bot sees, and switch the model to `gpt-6-luna` with reasoning effort `medium`
- policy: requests only. The bot joins when a message invites an answer from any participant,
  whether or not it ends with `?`. It never volunteers remarks on plain statements
- explicit commands (`chat!`, `gpt!`, `ai!`, `чат!`) keep their current path, prompt and limits;
  they bypass jev and the auto-reply limits
- the tool-using harness idea (history search, reply chains) is out of scope, recorded in
  `docs/backlog/` for after the fixed version has been observed

## Context (from discovery)
- `app/bot/openai/openai.go`: `OnMessage` (history add at :74, auto path :76-100, direct path
  :104-150), `shouldAnswerWithHistory` (:262), `chatGPTRequestWithHistory` (:275),
  `chatGPTRequestWithHistoryAndFocus` (:295, assumes the current message is the history tail at
  :307), `isReasoningModel` (:381), dice field `rand` (:49, :68)
- `app/main.go:105` dumps all options at DEBUG, including the OpenAI token today
- `.golangci.yml` excludes tests from lint (`tests: false`) and enables `unused`, so new
  unexported code needs a production caller in the same task
- `app/bot/openai/openaihistory.go`: `LimitedMessageHistory`, count-bounded only
- `app/events/telegram.go:184`: idle tick sends `bot.Message{Text: "idle"}` (ChatID 0) every
  `IdleDuration` (default 30s); `:139` passes messages from every chat, including private ones,
  to the bots
- `app/main.go:62-76`: OpenAI flags, `:129` sets `HistoryReplyProbability`; `:302`
  `makeOpenAIHttpClient` wraps a 10-attempt repeater. There is no `app/main_test.go`
- tests are `package openai` and import `app/bot/openai/mocks`, so any type in a mocked interface
  signature must live outside package `openai`, or the mocks package creates an import cycle
- `OnMessage` of one bot never runs concurrently with itself: one select loop in
  `TelegramListener.Do`, and `MultiBot` runs different bots in parallel (`bot.go:155`).
  `Summary` runs from summarizer goroutines (`summary.go:148`) and must not touch the new state
- earlier measurements on 2026-09-22: `gpt-5.6-luna`/medium 2.2-4.3s per reply against 3.9s for
  `gpt-5-mini`/low; jev 0.26-0.36s per call from a workstation
- Claude's subsequent live probe on 2026-09-22: `gpt-6-luna` accepts Chat Completions with
  `reasoning_effort=medium` and `max_completion_tokens=3000`, rejects `max_tokens`, and returned
  `finish_reason=stop`. Observed latency 2.3-5.0s, reasoning tokens 220-698, maximum total tokens
  753. Stage 1 stays synchronous. Codex verified the model ID, medium effort and Chat Completions
  support in the [official model documentation](https://developers.openai.com/api/docs/models/gpt-6-luna)
- production data, 2026-06-08 to 2026-09-22: ~189 incoming group messages a day, 94 unsolicited
  replies (~0.9 a day), ~100 private-chat messages mixed into the group history
- the reporter's daily JSON logs on master (`/srv/radio-t/master-node/log/super-bot/`) carry ID,
  author, time, text and reply-to text. They omit deleted messages, so the spam stratum comes from
  the raw `incoming msg` lines in `docker logs super-bot`

## Development Approach
- **testing approach**: TDD, tests first; each task runs its new tests red before implementing,
  except characterization tests of behavior that already holds
- complete each task fully before moving to the next
- make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
  - tests are not optional - they are a required part of the checklist
  - write unit tests for new functions/methods
  - write unit tests for modified functions/methods
  - add new test cases for new code paths
  - update existing test cases if behavior changes
  - tests cover both success and error scenarios
- **CRITICAL: all tests must pass before starting next task** - no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- run tests after each change
- maintain backward compatibility

## Code-Quality Rules (HARD — verify against every task before marking complete)

These rules supplement project CLAUDE.md and are NOT optional. They are the gate for marking any task complete. If a rule is violated, the task is not done — refactor, re-test, then mark complete.

**Signatures (hard limits):**
- No function or method has 4+ parameters. `ctx context.Context` does not count toward the budget. If you need 4+, use an option struct (e.g., `type fooOpts struct { ... }`).
- No function or method has 4+ return values. Split the function into two single-purpose ones, or return a struct.
- Multiple adjacent same-type parameters (`oldLine, newLine int`) are a swap hazard — review whether they belong on a struct.

**Methods vs standalone helpers (project rule, hard):**
- If a function is called only from methods of a single struct, it MUST be a method on that struct. Calling pattern decides, not field access.
- Standalone helpers are reserved for: (a) constructors and entry points (`Parse...`, `New...`, `Decorate...`), (b) utilities shared by multiple unrelated types or by both standalone functions AND methods, (c) tiny cross-cutting helpers.
- Before adding any standalone helper, mentally walk its callers. If every caller is a method of one type, make the helper a method on that type.

**Visibility (private by default, hard):**
- Lowercase identifiers by default. Only export when an out-of-package caller exists.
- Exception (per CLAUDE.md): methods called by other structs in the same package CAN be exported for inter-component API clarity. This is the only exception. It does not extend to types, functions, constants, or variables.
- Before exporting any new identifier, grep for cross-package callers. If none, lowercase it.

**Comments (default: none, hard):**
- Default to writing no comments. Add one only when the WHY is non-obvious (a hidden invariant, a workaround, behavior that would surprise a reader).
- Exported items get godoc comments starting with the name. Unexported items get lowercase non-godoc comments — or no comment at all.
- Never describe WHAT the code does when the code itself is self-evident. Never write multi-paragraph comments on routine helpers.

**Per-task gate (before marking ANY checkbox complete):**
1. Formatter runs clean (`~/.claude/format.sh` or `gofmt -s -w` + `goimports -w`).
2. `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0` reports zero issues.
3. `go test ./... -race` passes.
4. Scan the new code for the four rule classes above. Specifically:
   - Grep new function signatures: `grep -nE '^func.*\(.*,.*,.*,.*\)' app/<path>/*.go` — any hit with 4+ comma-separated params (excluding `ctx`) is a violation. Same for the return-value side.
   - For every new standalone helper, `grep -rn 'helperName(' --include='*.go'` and confirm at least one caller is NOT a method of a single type. If all callers are methods of one type, convert.
   - For every new exported identifier, grep cross-package. If no out-of-package hit, lowercase it.
5. Only after 1–4 pass: mark the task complete.

If a previous task shipped a violation (spotted later by user, reviewer, or yourself): fix it in the next commit BEFORE starting the next task. Do not let violations accumulate.

**Project-specific additions:**
- the jev key is a secret. It reaches the program only through its env var and never appears in
  this repo: no default, no example value, no real key in any fixture (tests use a fake key), no
  log line. The value lives only in the private `master-node` deployment repo
- chat content used for labeling and tuning never enters this repo; it stays outside the tree
- jev error reporting is sanitized: status plus the error type and message fields only; echoed
  `input` and any other body fields are dropped, since a 422 detail echoes the request state

## Testing Strategy
- **unit tests**: required for every task (see Development Approach above)
- jev client tests run against `httptest.Server` with a fake key
- gate and auto-reply tests use a moq mock of the consumer-side `jevClient` interface; the mock
  references only `jev` package types, so there is no import cycle
- wiring tests cover the auto-reply on/off × key set/empty matrix through an extracted function,
  with no conditional skip
- live threshold tuning uses a test in `gate_test.go` that runs the production snapshot
  construction and gate logic on the case file named by `JEV_LIVE_CASES`; skipped when the env var
  is unset, so normal runs and CI skip it. The case loader and replay pipeline get synthetic-case
  tests in the normal suite
- no e2e tests exist in this project

## Progress Tracking
- mark completed items with `[x]` immediately when done
- add newly discovered tasks with ➕ prefix
- document issues/blockers with ⚠️ prefix
- update plan if implementation deviates from original scope
- keep plan in sync with actual work done

## Solution Overview
- **history**: record only group messages (`ChatID < 0`), which drops idle ticks (ChatID 0) and
  private chats (positive IDs). The snapshot keeps only entries from the same chat as the current
  message, drops entries older than the max age, and caps text size. Default size goes from 5 to 10
- **snapshot**: built from the actual current message, passed in by the caller, plus the recorded
  preceding messages of that chat. The buffer tail is never assumed to be the current message.
  Each entry carries an author label, time, text, a marker for a known image, and the quoted reply
  parent, marked as unavailable when its text is empty. The same snapshot feeds the jev state and
  both OpenAI message builders
- **gate**: one jev call with three independent nouls over the snapshot. Reply only when
  `invites` and `answerable` are at or above their thresholds and `spam` is below its threshold.
  Any error, timeout, missing, null or invalid answer means no reply; no fallback to the dice
- **auto-reply eligibility**: enabled, jev configured, group message, text length, not a direct
  command, cooldown and daily cap. Checked before the jev call, so a blocked candidate costs
  nothing
- **auto-reply limits**: own state, separate from `lastDT`, which rate-limits direct requests and
  issues bans. An auto reply is counted when `OnMessage` returns `Send=true`; delivery is not
  confirmed on the synchronous path
- **model**: target `gpt-6-luna` with medium reasoning. Add the explicit GPT-6 family to
  `isReasoningModel` alongside GPT-5, then update deployment env; regression tests cover both
  Luna generations and the correct token/effort fields
- question texts, criteria and thresholds are constants in one file (`gate.go`); flags cover only
  the jev key, model and timeout

## Technical Details

**Initial constants** (tuned or confirmed in Task 9):
- history max age 30 minutes; per-entry text cap 1000 runes for preceding messages and reply
  quotes, each truncation marked; total snapshot text cap 8000 runes, oldest entries dropped first.
  The current message is never truncated: if it alone exceeds the total cap, the bot abstains
- auto-reply cooldown 15 minutes; daily cap 10 (today ~0.9 a day); the day resets at local
  midnight in the container's time zone
- jev calls capped at 150 per clock hour. Eligible messages per hour since June: median 3,
  p99 60, max 128, so the cap never binds on observed traffic and stops a flood from blocking the
  message loop with one jev call per message
- nothing in this change adds an uncapped path: unsolicited replies never run in private chats,
  direct queries keep the shared 5-minute cooldown and ban (super users exempt, as today), replies
  to the bot go through the unsolicited limits, and request size is bounded by the snapshot caps
- noul thresholds: `invites` >= 0.8, `answerable` >= 0.7, `spam` < 0.3

**jev request** (`POST https://api.typesafe.ai/v1/systemone`, model pinned to `jev-1.13.0`):

```json
{"model": "jev-1.13.0",
 "state": {"message": {"author": "user1", "text": "...", "image": true,
                       "reply_to": {"author": "user2", "text": "...", "unavailable": false}},
           "history": [{"author": "user2", "time": "15:04", "text": "...", "reply_to": {...}}]},
 "questions": {"invites": {"type": "noul", "instructions": "...", "criteria": {"true": "...", "false": "..."}},
               "answerable": {"type": "noul", ...},
               "spam": {"type": "noul", ...}}}
```

- author labels: `@username`, else display name, else a snapshot-local alias (`user1`, `user2`)
  that keeps one person's messages together; numeric Telegram IDs never leave the process
- `invites`: the latest message genuinely invites an answer from any chat participant, not only
  from the person it replies to; rhetorical questions, banter and questions about one person's own
  experience are false
- `answerable`: a short answer is possible from the supplied text plus general knowledge, with no
  need for an unseen link, image, voice message, earlier discussion or the author's private
  environment
- `spam`: advertising, job offers, paid services, recruitment or other solicitation
- the state is built from unexported structs with json tags matching the shape above

**OpenAI messages**: each preceding message becomes one `user` message rendered as
`author: text`, with `(in reply to author: quoted text)` when present; the current message is
rendered last. Direct requests keep the stripped request text and `Params.Prompt` as today. The
auto-reply system prompt changes from "add one substantive remark" to answering the current
message's request, in Russian, at most 50 words

**Config** (group `jev`, `env-namespace:"JEV"`):
- `--jev.key` / `JEV_KEY`, no default. Auto-response on with an empty key: startup warning,
  no unsolicited replies; direct commands and summaries unaffected
- `--jev.model` / `JEV_MODEL`, default `jev-1.13.0`
- `--jev.timeout` / `JEV_TIMEOUT`, default `2s`; zero or negative is rejected at startup
- `--openai.history-reply-probability` is removed

**Logging** per gate decision, at DEBUG: resolved jev model, the three noul values, the decision,
elapsed time. Failures at WARN with status plus the error type and message fields only. Never the
key or the state

## What Goes Where
- **Implementation Steps** (`[ ]` checkboxes): code, tests, docs in this repo; the Task 9 case
  data stays outside the repo
- **Post-Completion** (no checkboxes): deployment env in `master-node`, production observation

## Implementation Steps

### Task 1: Support gpt-6-luna reasoning requests

**Files:**
- Modify: `app/bot/openai/openai.go`
- Modify: `app/bot/openai/openai_test.go`
- Modify: `app/main.go`

- [x] add `gpt-6-luna` to `TestIsReasoningModel` expecting true; retain `gpt-5.6-luna` coverage
- [x] add a `gpt-6-luna` + `medium` row to `TestOpenAI_chatGPTRequestInternal_ReasoningEffort`
  asserting `MaxCompletionTokens` is set, `MaxTokens` is zero and `ReasoningEffort` is `medium`
- [x] retain the `gpt-5.6-luna`/medium request row; confirm the GPT-6 rows fail before changing
  production code, then add the explicit GPT-6 family match alongside GPT-5 and verify green
- [x] ➕ remove the always-true error conditional after `TelegramListener.Do` in `app/main.go`;
  log its returned error directly (behavior-neutral SA4023 fix; every return path is non-nil)
- [x] run tests - must pass before next task

### Task 2: Record only group messages in history (bug fix)

**Files:**
- Modify: `app/bot/openai/openai.go`
- Modify: `app/bot/openai/openai_test.go`

- [x] give the existing auto and direct-history fixtures a group ChatID (they default to 0 at
  `openai_test.go:258,265,272,633,639`)
- [x] write failing test: idle messages after real ones leave the real messages in history
- [x] write failing test: a private-chat message (positive ChatID) is not recorded and triggers
  no OpenAI call on the auto path
- [x] write failing test: after a direct query from a private chat, the stored group history is
  unchanged (the `[:len-1]` tail drop at `openai.go:307` must not act on a message never recorded)
- [x] run tests, confirm the new ones fail
- [x] record into history only when `msg.ChatID < 0`; skip the auto path for non-group messages;
  drop the history tail only when the current message was recorded
- [x] run tests - must pass before next task

### Task 3: jev client package

**Files:**
- Create: `app/bot/openai/jev/jev.go`
- Create: `app/bot/openai/jev/jev_test.go`

**Design Contract:**

Type:
- `Client` (exported: constructed in `app/main.go`, called from package `openai`)
- `Params` (exported: filled in `app/main.go`)
- `Question`, `Answer`, `Response` (exported: used by package `openai` and its mock)

Methods (full signatures):
- `New(params Params) *Client`
- `(c *Client) Ask(ctx context.Context, state any, questions map[string]Question) (Response, error)`
- `(c *Client) checkAnswers(resp Response, questions map[string]Question) error`

Standalone helpers planned (justification why NOT a method):
- none

Exports (justification per item: who outside the package calls this?):
- `Client`, `New`, `Params` (fields `Key`, `Model`, `Timeout`): `app/main.go`
- `Question`, `Answer`, `Response` and their fields, `(c *Client) Ask`: `app/bot/openai`
- the endpoint URL is an unexported field set by `New`; tests in the package override it

- [x] write `httptest.Server` tests first: success with request schema and bearer auth checked
  (fake key), missing answer, wrong answer type, noul absent vs null vs 0 vs 1 vs out of range vs
  non-numeric, non-2xx with detail, malformed JSON, response slower than the client timeout,
  exactly one HTTP attempt on 429/529
- [x] write a failing test: a 422 whose detail echoes the state and the fake key produces an error
  string containing neither
- [x] run tests, confirm they fail (new package API was undefined before implementation)
- [x] implement: own `http.Client` with `Timeout` from `Params` (no repeater), one attempt;
  `Ask` enforces the timeout itself, callers pass a plain context. Redirects are not followed;
  responses are capped at 64 KiB and sanitized error details at 256 runes
- [x] godoc on every exported type, field, function and method
- [x] run tests - must pass before next task

### Task 4: Structured history snapshot, wired into both OpenAI builders

The snapshot gets its production callers in this task, so the `unused` linter (tests are excluded
by `.golangci.yml`) passes.

**Files:**
- Modify: `app/bot/openai/openaihistory.go`
- Modify: `app/bot/openai/openaihistory_test.go`
- Modify: `app/bot/openai/openai.go`
- Modify: `app/bot/openai/openai_test.go`

**Design Contract:**

Type:
- `historySnapshot` (unexported): current message, preceding entries, truncation markers

Methods (full signatures):
- `(l *LimitedMessageHistory) snapshot(cur bot.Message, now time.Time) historySnapshot`
- `(s historySnapshot) chatMessages(sysPrompt, current string) []openai.ChatCompletionMessage`
- `(o *OpenAI) chatGPTRequestWithHistory(s historySnapshot, sysPrompt string) (string, error)`
- `(o *OpenAI) chatGPTRequestWithHistoryAndFocus(s historySnapshot, reqText, sysPrompt string) (string, error)`

Standalone helpers planned (justification why NOT a method):
- none

Exports (justification per item: who outside the package calls this?):
- none

- [x] write failing tests for `snapshot`: current is `cur` even when it was never recorded;
  preceding entries come only from `cur.ChatID`; the entry equal to `cur` is not duplicated;
  age boundary at exactly 30 minutes; per-entry cap truncates and marks preceding entries and
  quotes; total cap drops the oldest first; a current message whose decisive part sits past rune
  1000 arrives intact
- [x] write failing tests for `chatMessages`: system prompt first, `author: text` rendering with
  the label fallback chain and snapshot-local aliases (no numeric IDs), image marker, reply quote
  with an empty parent marked unavailable, current message last with the caller-given text
  (stripped request for direct)
- [x] write failing tests: a direct query from a private chat sends its own stripped question and
  reply parent with no group context (inspect the request messages); two group chats do not see
  each other's history; update the rendering assertions at `openai_test.go:283-284`
- [x] run tests, confirm they fail
- [x] implement the snapshot types and methods with the constants from Technical Details; move
  both builders onto the snapshot, removing the bare-text loops and the tail assumption; the
  direct path keeps `Params.Prompt` and the stripped request text
- [x] run tests - must pass before next task

### Task 5: Involvement gate and auto-reply limits, wired into OnMessage

**Files:**
- Create: `app/bot/openai/gate.go`
- Create: `app/bot/openai/gate_test.go`
- Modify: `app/bot/openai/openai.go`
- Modify: `app/bot/openai/openai_test.go`
- Create: `app/bot/openai/mocks/jev_client.go` (generated)

**Design Contract:**

Type:
- `jevClient` interface in `openai.go` (consumer side):
  `Ask(ctx context.Context, state any, questions map[string]jev.Question) (jev.Response, error)`
- `autoReplyLimits` (unexported): last reply time, day, count for the day, jev calls in the
  current clock hour
- `jevState`, `snapshotEntry`, `snapshotReply` (unexported, json-tagged jev projection)
- `historySnapshot` gains `oversized bool`, set when the current message alone exceeds the total
  cap; read by `shouldJoin`

Methods (full signatures):
- `(s historySnapshot) jevState() jevState`
- `(o *OpenAI) shouldJoin(s historySnapshot) bool`
- `(l *autoReplyLimits) allowed(now time.Time) bool`
- `(l *autoReplyLimits) mark(now time.Time)`
- `(l *autoReplyLimits) noteJevCall(now time.Time)`
- `(o *OpenAI) answerDirect(msg bot.Message, reqText string) bot.Response`
- `(o *OpenAI) autoReply(msg bot.Message) bot.Response`

Standalone helpers planned (justification why NOT a method):
- none

Exports (justification per item: who outside the package calls this?):
- `Params.Jev jevClient`: set from `app/main.go`; godoc says nil disables unsolicited replies

- [ ] add `//go:generate` for `jevClient` next to the existing one and generate the mock
- [ ] run the existing suite green, then extract `answerDirect` unchanged (behavior-preserving)
- [ ] write failing tests for `jevState`: same labels and aliases as `chatMessages`, no numeric
  IDs, image marker, reply parent with empty text marked unavailable; a current message over the
  total cap sets `oversized` and is carried whole
- [ ] write failing table tests for `shouldJoin`: each noul at and around its threshold, spam veto
  overriding high `invites`/`answerable`, jev error, missing answer, nil client, oversized snapshot
- [ ] write failing tests for `allowed`/`mark`/`noteJevCall`: cooldown boundary, daily cap,
  reset at local midnight, the 151st jev call in an hour is refused and the budget resets on the
  next clock hour
- [ ] write failing tests for the limits end to end: a burst of eligible messages past the hourly
  jev budget makes no further jev calls; a direct query from a private chat within the 5-minute
  cooldown is refused like a group one; a reply to the bot's own message cannot bypass the
  cooldown or daily cap
- [ ] write failing integration tests: an unpunctuated request accepted by jev calls jev and
  OpenAI once each and replies to the triggering message (`ReplyTo: msg.ID`); the first genuine
  group message can qualify without a full buffer; jev rejection, error or nil client makes zero
  OpenAI calls; cooldown and cap skip the jev call; empty or failed OpenAI output does not mark;
  a nonempty auto reply marks once; auto replies never change `lastDT` or issue a ban; a direct
  query bypasses jev and the auto limits
- [ ] run tests, confirm the new ones fail
- [ ] put question texts, criteria and thresholds as constants in `gate.go`; implement
  `shouldJoin` with DEBUG decision log and WARN on failure; implement `autoReply` with the
  requests-only system prompt
- [ ] remove `shouldAnswerWithHistory`, the `?` rule, the `rand` field, and the dice tests
  (`openai_test.go:289,333,376`) with their `rand` overrides (`:253,310,354,417`)
- [ ] run tests - must pass before next task

### Task 6: Config wiring

**Files:**
- Modify: `app/main.go`
- Modify: `app/bot/openai/openai.go` (remove `Params.HistoryReplyProbability`)
- Modify: `app/bot/openai/openai_test.go`
- Create: `app/main_test.go`

**Design Contract:**

Methods (full signatures):
- `setJev(p *openai.Params, jp jev.Params, autoResp bool) error` in `app/main.go` (standalone:
  called from `main`; leaves `p.Jev` a true nil for an empty key and logs the startup warning when
  `autoResp` is on; returns an error for a non-positive timeout)
- `redactedOpts() string` in `app/main.go` (standalone: `opts` is an anonymous struct, so no
  method is possible; copies `opts`, clears `Telegram.Token`, `MashapeToken`, `OpenAI.AuthToken`,
  `UreadabilityToken` and `Jev.Key`, formats with `%+v` for the DEBUG dump at `main.go:105`)

Standalone helpers planned: `setJev`, `redactedOpts`, as above. Exports: none

- [ ] write failing tests for `setJev`: key set, key empty (`p.Jev == nil`, not a typed nil),
  zero and negative timeout, warning only when auto-response is on and the key is empty
- [ ] write failing test: the options dump contains none of the five secrets (fake values)
- [ ] run tests, confirm they fail
- [ ] add the `jev` flag group; call `setJev`; log `redactedOpts()` instead of `opts`
- [ ] remove `--openai.history-reply-probability`, `Params.HistoryReplyProbability`, the
  `main.go:129` call site and the `getDefaultTestingConfig` field (`openai_test.go:33`) together;
  raise the `--openai.history-size` default to 10
- [ ] run tests - must pass before next task

### Task 7: Freeze the labeling contract and replay pipeline

**Files:**
- Modify: `app/bot/openai/gate_test.go`

- [ ] define the case file format (JSON lines, stored outside the repo): case ID, chat, time,
  current message, strictly preceding context as the production recorder would hold it, quoted
  parent, missing-context and truncation markers, stratum and split (hidden from labelers), label
  yes/no/unjudgeable, reason and evidence needed
- [ ] write failing synthetic-case tests for the loader and replay, then the loader and replay
  that feed cases through the production `snapshot` and `shouldJoin`
- [ ] write the live replay test in `gate_test.go`, skipped unless `JEV_LIVE_CASES` names a case
  file; it calls the real jev API and reports precision, reply rate and nuisance counts per
  stratum, never pooled; run it only on demand as `JEV_LIVE_CASES=<file> go test -run <name> -count=1 ./app/bot/openai`, never exported into routine gates
- [ ] run tests - must pass before next task

### Task 8: Label cases and tune thresholds

**Files:**
- Modify: `app/bot/openai/gate.go` (tuned constants and question texts)

- [ ] build the case set outside the repo from the reporter logs and docker logs: a representative
  random stratum, the 94 past replies as a challenge stratum, and a spam stratum; split dev and
  held-out by day and by conversation; no case shows later messages or the historical bot reply
- [ ] Claude Code and codex label each case independently with the frozen rubric: "should this
  bot, text-only and uninvited, answer now"; disagreements and a sample of agreements, weighted
  to accepted cases, go to the user
- [ ] tune question texts and thresholds on dev only, freeze them, then report held-out results
  per stratum before and after
- [ ] generate replies for a small held-out set of accepted cases with `gpt-6-luna`/medium and
  inspect them against their context; gate metrics alone do not show the answers read naturally
- [ ] run the normal test suite - must pass before next task

### Task 9: Backlog item for the tool harness

**Files:**
- Create: `docs/backlog/ai-reply-tool-harness.md`

- [ ] record the postponed harness idea with codex's notes: count retrieval need across socially
  eligible requests, before any gate filters them; try archive search offline first; reuse
  `TelegramListener.Submit` for delayed replies with a narrow envelope (reply target, expiry,
  outcome); reporter logs lack parent message IDs

### Task 10: Verify acceptance criteria
- [ ] verify all requirements from Overview are implemented
- [ ] verify edge cases are handled
- [ ] run full test suite: `go test ./... -race`
- [ ] run linter: `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0`
- [ ] verify test coverage for `app/bot/openai` and `app/bot/openai/jev` is at least 80%
- [ ] grep the tree for the jev key prefix and for chat content: none present

### Task 11: [Final] Update documentation
- [ ] update README.md: jev flags, unsolicited-reply behavior, removed probability flag
- [ ] update CLAUDE.md with the gate and snapshot pattern
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion
*Items requiring manual intervention or external systems - no checkboxes, informational only*

**Deployment** (`master-node`, private repo only), on the user's go-ahead:
- super-bot env: `OPENAI_MODEL=gpt-6-luna`, `OPENAI_REASONING_EFFORT=medium`, `JEV_KEY=<key>`;
  drop `OPENAI_HISTORY_REPLY_PROBABILITY`
- ordering: land the env change before or with the image. The auto-updater pulls the new image
  after CI; until `JEV_KEY` is installed the bot runs with unsolicited replies disabled
- apply env with `make deploy-configs`, then `docker compose up -d super-bot` on master;
  `docker compose restart` does not load new env

**Observation**:
- watch the DEBUG gate log for a week: replies per day, what it answered, near-misses
- decide on the tool harness from the backlog item after that

Smells pre-check: 10 items fixed before save (jev client moved to its own package to break the
mocks import cycle; limits methods moved onto `autoReplyLimits`; snapshot takes the current message
from the caller; client owns its timeout; typed-nil guard; unexported endpoint URL; godoc required;
typed jev state; OnMessage split contract added; dead `rand` field removed)

Plan review applied: jev key and OpenAI token redacted from the DEBUG options dump; snapshot and
gate each wired into production in their own task for the `unused` linter; `setJev` extracted so
the typed-nil and warning cases are testable; private-direct tail drop guarded in Task 2; live
replay test folded into `gate_test.go` (one test file per source file); auto replies reply to the
triggering message
