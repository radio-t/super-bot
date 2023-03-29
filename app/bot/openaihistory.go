package bot

// LimitedMessageHistory is a limited message history for OpenAI bot
// It's using to make context answers in the chat
// This isn't thread safe structure
type LimitedMessageHistory struct {
	limit    int
	count    int
	messages []Message
}

// NewLimitedMessageHistory makes a new LimitedMessageHistory with limit
func NewLimitedMessageHistory(limit int) LimitedMessageHistory {
	return LimitedMessageHistory{
		limit:    limit,
		count:    0,
		messages: make([]Message, 0, limit),
	}
}

// Add adds a new message to the history
func (l *LimitedMessageHistory) Add(message Message) {
	l.count++
	l.messages = append(l.messages, message)
	if len(l.messages) > l.limit {
		l.messages = l.messages[1:]
	}
}
