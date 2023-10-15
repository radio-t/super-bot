package bot

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"strings"
)

// SpamLocalFilter bot, checks if user is a spammer using internal matching
type SpamLocalFilter struct {
	dry       bool
	superUser SuperUser
	threshold float64

	enabled       bool
	tokenizedSpam []map[string]int
	approvedUsers map[int64]bool
}

// NewSpamLocalFilter makes a spam detecting bot
func NewSpamLocalFilter(spamSamples io.Reader, threshold float64, superUser SuperUser, dry bool) *SpamLocalFilter {
	log.Printf("[INFO] Spam bot (local), threshold=%0.2f", threshold)
	res := &SpamLocalFilter{dry: dry, approvedUsers: map[int64]bool{}, superUser: superUser, threshold: threshold}

	scanner := bufio.NewScanner(spamSamples)
	for scanner.Scan() {
		tokenizedSpam := res.tokenize(scanner.Text())
		res.tokenizedSpam = append(res.tokenizedSpam, tokenizedSpam)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[WARN] failed to read spam samples, error=%v", err)
		res.enabled = false
	} else {
		res.enabled = true
	}
	return res
}

// OnMessage checks if user already approved and if not checks if user is a spammer
func (s *SpamLocalFilter) OnMessage(msg Message) (response Response) {
	if !s.enabled {
		return Response{}
	}

	if s.approvedUsers[msg.From.ID] || msg.From.ID == 0 {
		return Response{}
	}

	if s.superUser.IsSuper(msg.From.Username) {
		return Response{} // don't check super users for spam
	}

	displayUsername := DisplayName(msg)
	if !s.isSpam(msg.Text) {
		if id := msg.From.ID; id != 0 {
			s.approvedUsers[id] = true
			log.Printf("[INFO] user %s is not a spammer id %d, added to aproved", displayUsername, msg.From.ID)
		}
		return Response{} // not a spam
	}

	log.Printf("[INFO] user %s detected as spammer, msg: %q", displayUsername, msg.Text)
	if s.dry {
		return Response{
			Text: fmt.Sprintf("this is spam from %q, but I'm in dry mode, so I'll do nothing yet", displayUsername),
			Send: true, ReplyTo: msg.ID,
		}
	}
	return Response{Text: fmt.Sprintf("this is spam! go to ban, %q (id:%d)", displayUsername, msg.From.ID),
		Send: true, ReplyTo: msg.ID, BanInterval: permanentBanDuration, DeleteReplyTo: true}
}

// Help returns help message
func (s *SpamLocalFilter) Help() string { return "" }

// ReactOn keys
func (s *SpamLocalFilter) ReactOn() []string { return []string{} }

// isSpam checks if a given message is similar to any of the known bad messages.
func (s *SpamLocalFilter) isSpam(message string) bool {
	tokenizedMessage := s.tokenize(message)
	maxSimilarity := 0.0
	for _, spam := range s.tokenizedSpam {
		similarity := s.cosineSimilarity(tokenizedMessage, spam)
		if similarity > maxSimilarity {
			maxSimilarity = similarity
		}
		if similarity >= s.threshold {
			return true
		}
	}
	log.Printf("[DEBUG] spam similarity: %0.2f", maxSimilarity)
	return false
}

// tokenize takes a string and returns a map where the keys are unique words (tokens)
// and the values are the frequencies of those words in the string.
func (s *SpamLocalFilter) tokenize(inp string) map[string]int {
	tokenFrequency := make(map[string]int)
	tokens := strings.Fields(inp)
	for _, token := range tokens {
		tokenFrequency[strings.ToLower(token)]++
	}
	return tokenFrequency
}

// cosineSimilarity calculates the cosine similarity between two token frequency maps.
func (s *SpamLocalFilter) cosineSimilarity(a, b map[string]int) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	dotProduct := 0      // sum of product of corresponding frequencies
	normA, normB := 0, 0 // square root of sum of squares of frequencies

	for key, val := range a {
		dotProduct += val * b[key]
		normA += val * val
	}
	for _, val := range b {
		normB += val * val
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	// cosine similarity formula
	return float64(dotProduct) / (math.Sqrt(float64(normA)) * math.Sqrt(float64(normB)))
}
