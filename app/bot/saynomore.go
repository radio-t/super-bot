package bot

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

// SayNoMoreCategory defines a category of stop words with responses
type SayNoMoreCategory struct {
	Words     []string
	Responses []string
}

// compiledCategory holds compiled patterns and responses for matching
type compiledCategory struct {
	patterns  []*regexp.Regexp
	responses []string
}

// SayNoMore bot bans users for using stop words
type SayNoMore struct {
	superUser   SuperUser
	minDuration time.Duration
	maxDuration time.Duration
	rand        func(n int64) int64 // tests may change it
	categories  []compiledCategory
}

// NewDefaultSayNoMore creates a new stop words bot with default categories
func NewDefaultSayNoMore(superUser SuperUser) *SayNoMore {
	return NewSayNoMore(1*time.Hour, 12*time.Hour, superUser, []SayNoMoreCategory{
		{
			Words:     []string{"крайний выпуск", "крайний раз"},
			Responses: []string{"Крайний герой", "Крайний день Помпеи", "Крайний из могикан", "Крайний звонок", "Крайняя надежда"},
		},
		{
			Words:     []string{"ихний"},
			Responses: []string{"евонный"},
		},
		{
			Words:     []string{"доброго времени суток"},
			Responses: []string{"Что дозволено Юпитеру - не дозволено быку", "Quod licet Iovi, non licet bovi"},
		},
		{
			Words:     []string{"имеет место быть"},
			Responses: []string{"более лучше"},
		},
	})
}

// NewSayNoMore creates a new stop words bot
func NewSayNoMore(minDuration, maxDuration time.Duration, superUser SuperUser, categories []SayNoMoreCategory) *SayNoMore {
	log.Printf("[INFO] SayNoMore bot with %d categories, %v-%v interval", len(categories), minDuration, maxDuration)

	compiled := make([]compiledCategory, 0, len(categories))
	for _, cat := range categories {
		if len(cat.Responses) == 0 {
			log.Printf("[WARN] SayNoMore category with words %v has no responses, skipped", cat.Words)
			continue
		}
		patterns := make([]*regexp.Regexp, len(cat.Words))
		for j, word := range cat.Words {
			patterns[j] = regexp.MustCompile(`(?:^|[^\p{L}])` + regexp.QuoteMeta(strings.ToLower(word)) + `(?:[^\p{L}]|$)`)
		}
		compiled = append(compiled, compiledCategory{patterns: patterns, responses: cat.Responses})
	}

	return &SayNoMore{
		superUser:   superUser,
		minDuration: minDuration,
		maxDuration: maxDuration,
		rand:        rand.Int63n,
		categories:  compiled,
	}
}

// OnMessage checks for stop words and bans user
func (s *SayNoMore) OnMessage(msg Message) Response {
	var matched *compiledCategory
	text := strings.ToLower(msg.Text)
	for i := range s.categories {
		for _, pattern := range s.categories[i].patterns {
			if pattern.MatchString(text) {
				matched = &s.categories[i]
				break
			}
		}
		if matched != nil {
			break
		}
	}

	if matched == nil {
		return Response{}
	}

	if s.superUser.IsSuper(msg.From.Username) {
		log.Printf("[DEBUG] SayNoMore triggered by super user %q, ignored", msg.From.Username)
		return Response{}
	}

	mention := "@" + msg.From.Username
	if msg.From.Username == "" {
		mention = msg.From.DisplayName
	}

	banDuration := s.minDuration + time.Second*time.Duration(s.rand(int64(s.maxDuration.Seconds()-s.minDuration.Seconds())))

	// pick random response from matched category
	response := matched.responses[s.rand(int64(len(matched.responses)))]

	log.Printf("[INFO] SayNoMore triggered by %q, ban for %v", msg.From.Username, banDuration)

	return Response{
		Text:        fmt.Sprintf("%s %s (%v)", EscapeMarkDownV1Text(mention), response, HumanizeDuration(banDuration)),
		Send:        true,
		BanInterval: banDuration,
		User:        msg.From,
	}
}

// ReactOn returns nil as this bot matches patterns, not specific commands
func (s *SayNoMore) ReactOn() []string {
	return nil
}

// Help returns help message
func (s *SayNoMore) Help() string {
	return "Боремся за чистоту чата"
}
