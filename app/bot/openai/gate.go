package openai

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/radio-t/super-bot/app/bot/openai/jev"
)

const (
	invitesThreshold    = 0.7
	answerableThreshold = 0.65
	spamThreshold       = 0.3
	autoReplyCooldown   = 15 * time.Minute
	autoReplyDailyCap   = 10
	jevHourlyBudget     = 150

	invitesQuestion = "Does `message` genuinely request an answer from any knowledgeable chat participant? " +
		"Use `history` and the quoted reply parent to identify the addressee and intent. Judge the current message, not questions inside quotes."
	invitesTrue  = "A genuine, still-open request for information that welcomes an uninvolved participant's answer, even without a question mark."
	invitesFalse = "A plain statement, banter, rhetorical question, already answered request, or a question aimed at a specific person's own experience. " +
		"Being in a reply thread alone does not make the answer false."
	answerableQuestion = "Can `message` be answered briefly and concretely by someone who has only general, stable knowledge " +
		"and the supplied text, with no personal experience, no browsing and no knowledge of recent events?"
	answerableTrue = "The question and its subject are clear from `message`, its quoted parent and `history`, and the supplied text " +
		"plus stable general knowledge fully addresses it."
	answerableFalse = "The message polls other people's personal experience or opinions (has anyone tried, how is it for you, what do you use), " +
		"or the answer requires information not available in the supplied text: a specific item mentioned elsewhere (that link, that skill, " +
		"that post), recent news, releases, prices or status, an unseen link, image or voice message, missing discussion, " +
		"or the author's private environment. An unrelated link or media item alone is not a reason to reject."
	spamQuestion    = "Does `message` contain unsolicited advertising, recruitment, job offers, paid-service promotion or other solicitation?"
	spamTrue        = "The message solicits customers, recruits, payments, subscriptions or promotional engagement."
	spamFalse       = "Ordinary conversation or a genuine question, including discussion of products, prices, paid services or employment without solicitation."
	autoReplyPrompt = "Answer the current message's request directly in Russian, at most 50 words, using the supplied conversation context. " +
		"Provide only concrete information. If necessary context is missing or you cannot give a useful answer, reply with an empty string. " +
		"Do not greet, praise, editorialize, ask follow-up questions or invite chat. No emojis or exclamations; neutral tone."
)

type jevState struct {
	Message snapshotEntry   `json:"message"`
	History []snapshotEntry `json:"history"`
}

type snapshotEntry struct {
	Author    string         `json:"author"`
	Time      string         `json:"time,omitempty"`
	Text      string         `json:"text"`
	Truncated bool           `json:"truncated,omitempty"`
	Image     bool           `json:"image,omitempty"`
	ReplyTo   *snapshotReply `json:"reply_to,omitempty"`
}

type snapshotReply struct {
	Author      string `json:"author"`
	Text        string `json:"text"`
	Truncated   bool   `json:"truncated,omitempty"`
	Unavailable bool   `json:"unavailable,omitempty"`
}

type involvementScores struct {
	invites    float64
	answerable float64
	spam       float64
}

type autoReplyLimits struct {
	lastReply time.Time
	day       string
	count     int
	hour      string
	jevCalls  int
}

func (s historySnapshot) jevState() jevState {
	state := jevState{Message: s.current.jevEntry(), History: make([]snapshotEntry, 0, len(s.previous))}
	for _, entry := range s.previous {
		state.History = append(state.History, entry.jevEntry())
	}
	return state
}

func (e historyEntry) jevEntry() snapshotEntry {
	entry := snapshotEntry{Author: e.author, Text: e.text.value, Truncated: e.text.truncated, Image: e.image}
	if !e.sent.IsZero() {
		entry.Time = e.sent.Format(time.RFC3339)
	}
	if e.reply != nil {
		entry.ReplyTo = &snapshotReply{Author: e.reply.author, Text: e.reply.text.value,
			Truncated: e.reply.text.truncated, Unavailable: e.reply.unavailable}
	}
	return entry
}

func (o *OpenAI) shouldJoin(s historySnapshot) bool {
	if o.params.Jev == nil || s.oversized {
		return false
	}
	questions := map[string]jev.Question{
		"invites":    {Type: "noul", Instructions: invitesQuestion, Criteria: map[string]string{"true": invitesTrue, "false": invitesFalse}},
		"answerable": {Type: "noul", Instructions: answerableQuestion, Criteria: map[string]string{"true": answerableTrue, "false": answerableFalse}},
		"spam":       {Type: "noul", Instructions: spamQuestion, Criteria: map[string]string{"true": spamTrue, "false": spamFalse}},
	}
	started := time.Now()
	response, err := o.params.Jev.Ask(context.Background(), s.jevState(), questions)
	if err != nil {
		log.Printf("[WARN] jev gate request failed: %v", err)
		return false
	}
	scores, err := o.gateScores(response)
	if err != nil {
		log.Printf("[WARN] jev gate response invalid: %v", err)
		return false
	}
	join := scores.invites >= invitesThreshold && scores.answerable >= answerableThreshold && scores.spam < spamThreshold
	log.Printf("[DEBUG] jev gate model=%q invites=%g answerable=%g spam=%g join=%t elapsed=%s",
		response.Model, scores.invites, scores.answerable, scores.spam, join, time.Since(started))
	return join
}

func (o *OpenAI) gateScores(response jev.Response) (involvementScores, error) {
	if response.Model == "" {
		return involvementScores{}, fmt.Errorf("missing model")
	}
	for _, name := range []string{"invites", "answerable", "spam"} {
		answer, ok := response.Answers[name]
		if !ok || answer.Type != "noul" || answer.Noul == nil {
			return involvementScores{}, fmt.Errorf("missing or invalid %s answer", name)
		}
		value := *answer.Noul
		if value < 0 || value > 1 || math.IsNaN(value) || math.IsInf(value, 0) {
			return involvementScores{}, fmt.Errorf("invalid %s probability", name)
		}
	}
	return involvementScores{invites: *response.Answers["invites"].Noul,
		answerable: *response.Answers["answerable"].Noul, spam: *response.Answers["spam"].Noul}, nil
}

func (l *autoReplyLimits) allowed(now time.Time) bool {
	l.reset(now)
	return (l.lastReply.IsZero() || now.Sub(l.lastReply) >= autoReplyCooldown) &&
		l.count < autoReplyDailyCap && l.jevCalls < jevHourlyBudget
}

func (l *autoReplyLimits) mark(now time.Time) {
	l.reset(now)
	l.lastReply = now
	l.count++
}

func (l *autoReplyLimits) noteJevCall(now time.Time) {
	l.reset(now)
	l.jevCalls++
}

func (l *autoReplyLimits) reset(now time.Time) {
	if day := now.Format("2006-01-02"); day != l.day {
		l.day, l.count = day, 0
	}
	if hour := now.Format("2006-01-02T15Z07:00"); hour != l.hour {
		l.hour, l.jevCalls = hour, 0
	}
}
