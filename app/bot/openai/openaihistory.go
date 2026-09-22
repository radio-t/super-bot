package openai

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sashabaranov/go-openai"

	"github.com/radio-t/super-bot/app/bot"
)

const (
	historyMaxAge        = 30 * time.Minute
	historyEntryRunes    = 1000
	historySnapshotRunes = 8000
)

// LimitedMessageHistory is a limited message history for OpenAI bot
// It's using to make context answers in the chat
// This isn't thread safe structure
type LimitedMessageHistory struct {
	limit    int
	count    int
	messages []bot.Message
}

// NewLimitedMessageHistory makes a new LimitedMessageHistory with limit
func NewLimitedMessageHistory(limit int) LimitedMessageHistory {
	return LimitedMessageHistory{
		limit:    limit,
		count:    0,
		messages: make([]bot.Message, 0, limit),
	}
}

// Add adds a new message to the history
func (l *LimitedMessageHistory) Add(message bot.Message) {
	l.count++
	l.messages = append(l.messages, message)
	if len(l.messages) > l.limit {
		l.messages = l.messages[1:]
	}
}

type historySnapshot struct {
	current   historyEntry
	previous  []historyEntry
	oversized bool
}

type historyEntry struct {
	author string
	sent   time.Time
	text   historyText
	image  bool
	reply  *historyReply
}

type historyReply struct {
	author      string
	text        historyText
	unavailable bool
}

type historyText struct {
	value     string
	truncated bool
}

type historyAuthor struct {
	id      int64
	name    string
	channel bool
}

type historyAuthors struct {
	labels map[historyAuthor]string
	used   map[string]bool
}

func (l *LimitedMessageHistory) snapshot(cur bot.Message, now time.Time) historySnapshot {
	authors := &historyAuthors{labels: map[historyAuthor]string{}, used: map[string]bool{}}
	snapshot := historySnapshot{previous: make([]historyEntry, 0, len(l.messages))}
	for _, msg := range l.messages {
		if msg.ChatID != cur.ChatID || (cur.ID != 0 && msg.ID == cur.ID) {
			continue
		}
		if msg.Sent.IsZero() || now.Sub(msg.Sent) > historyMaxAge || msg.Sent.After(now) {
			continue
		}
		if !cur.Sent.IsZero() && msg.Sent.After(cur.Sent) {
			continue
		}
		snapshot.previous = append(snapshot.previous, l.entry(msg, authors, true))
	}
	snapshot.current = l.entry(cur, authors, false)
	snapshot.oversized = snapshot.current.text.size() > historySnapshotRunes
	if snapshot.current.reply != nil {
		snapshot.current.reply.text.limit(max(0, historySnapshotRunes-snapshot.current.text.size()))
	}
	total := snapshot.current.size()
	for _, entry := range snapshot.previous {
		total += entry.size()
	}
	for total > historySnapshotRunes && len(snapshot.previous) > 0 {
		total -= snapshot.previous[0].size()
		snapshot.previous = snapshot.previous[1:]
	}
	return snapshot
}

func (l *LimitedMessageHistory) entry(msg bot.Message, authors *historyAuthors, trim bool) historyEntry {
	entry := historyEntry{author: authors.label(msg.From, msg.SenderChat), sent: msg.Sent,
		text: historyText{value: msg.Text}, image: msg.Image != nil}
	if msg.Image != nil && entry.text.value == "" {
		entry.text.value = msg.Image.Caption
	}
	if trim {
		entry.text.limit(historyEntryRunes)
	}
	parent := msg.ReplyTo
	if parent.From != (bot.User{}) || parent.SenderChat != (bot.SenderChat{}) || parent.Text != "" || !parent.Sent.IsZero() {
		entry.reply = &historyReply{author: authors.label(parent.From, parent.SenderChat),
			text: historyText{value: parent.Text}, unavailable: parent.Text == ""}
		entry.reply.text.limit(historyEntryRunes)
	}
	return entry
}

func (s historySnapshot) chatMessages(sysPrompt, current string) []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, len(s.previous)+2)
	messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: sysPrompt})
	for _, entry := range s.previous {
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: entry.render(entry.text.value)})
	}
	messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: s.current.render(current)})
	return messages
}

func (e historyEntry) size() int {
	size := e.text.size()
	if e.reply != nil {
		size += e.reply.text.size()
	}
	return size
}

func (e historyEntry) render(current string) string {
	text := e.text
	text.value = current
	var result strings.Builder
	fmt.Fprintf(&result, "%s: %s", e.author, text.render())
	if !e.sent.IsZero() {
		fmt.Fprintf(&result, "\n[sent: %s]", e.sent.Format(time.RFC3339))
	}
	if e.image {
		result.WriteString("\n[image unavailable]")
	}
	if e.reply != nil {
		quoted := e.reply.text.render()
		if e.reply.unavailable {
			quoted = "[unavailable]"
		}
		fmt.Fprintf(&result, "\n(in reply to %s: %s)", e.reply.author, quoted)
	}
	return result.String()
}

func (t historyText) size() int {
	return utf8.RuneCountInString(t.value)
}

func (t *historyText) limit(limit int) {
	if t.size() > limit {
		t.value = string([]rune(t.value)[:limit])
		t.truncated = true
	}
}

func (t historyText) render() string {
	if t.truncated {
		return t.value + " [truncated]"
	}
	return t.value
}

func (a *historyAuthors) label(user bot.User, sender bot.SenderChat) string {
	key := historyAuthor{id: user.ID, name: user.Username + "\x00" + user.DisplayName}
	if sender.ID != 0 || sender.UserName != "" {
		user = bot.User{ID: sender.ID, Username: sender.UserName}
		key = historyAuthor{id: sender.ID, name: sender.UserName, channel: true}
	}
	known := user.ID != 0 || user.Username != "" || user.DisplayName != ""
	if key.id != 0 {
		key.name = ""
	}
	if label, ok := a.labels[key]; known && ok {
		return label
	}
	label := user.DisplayName
	if user.Username != "" {
		label = "@" + user.Username
	}
	if label == "" {
		label = fmt.Sprintf("user%d", len(a.used)+1)
	}
	base := label
	for n := len(a.used) + 1; a.used[label]; n++ {
		label = fmt.Sprintf("%s (user%d)", base, n)
	}
	a.used[label] = true
	if known {
		a.labels[key] = label
	}
	return label
}
