package bot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// SpamFilter bot, checks if user is a spammer using internal matching as well as CAS API
type SpamFilter struct {
	SpamParams

	tokenizedSpam []map[string]int
	approvedUsers map[int64]bool
}

const maxEmojiAllowed = 2

// If user is restricted for more than 366 days or less than 30 seconds from the current time,
// they are considered to be restricted forever.
var permanentBanDuration = time.Hour * 24 * 400

var stopWords = []string{
	"в личку", "писать в лс", "пишите в лс",
	"лuчные сообщенuя", "личныe cooбщeния", "личных сообщениях", "заработок удалённо",
	"заработок в интернете", "заработок в сети", "заработок в сети интернет", "для yдaлённoгo зaрaбoткa",
}

// SpamParams is a full set of parameters for spam bot
type SpamParams struct {
	SuperUser           SuperUser
	SpamSamples         io.Reader
	SimilarityThreshold float64
	MinMsgLen           int
	CasAPI              string
	HTTPClient          HTTPClient
	Dry                 bool
}

// NewSpamFilter makes a spam detecting bot
func NewSpamFilter(p SpamParams) *SpamFilter {
	log.Printf("[INFO] spam bot: %+v", p)
	res := &SpamFilter{SpamParams: p, approvedUsers: map[int64]bool{}}

	scanner := bufio.NewScanner(p.SpamSamples)
	for scanner.Scan() {
		tokenizedSpam := res.tokenize(scanner.Text())
		res.tokenizedSpam = append(res.tokenizedSpam, tokenizedSpam)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[WARN] failed to read spam samples, error=%v", err)
	} else {
		log.Printf("[INFO] loaded %d spam samples, local spam filter enabled", len(res.tokenizedSpam))
	}
	return res
}

// OnMessage checks if user already approved and if not checks if user is a spammer
func (s *SpamFilter) OnMessage(msg Message) (response Response) {
	if s.approvedUsers[msg.From.ID] || msg.From.ID == 0 || len(msg.Text) < s.MinMsgLen {
		return Response{}
	}

	if s.SuperUser.IsSuper(msg.From.Username) {
		return Response{} // don't check super users for spam
	}

	displayUsername := strings.TrimSpace(DisplayName(msg))
	isEmojiSpam, _ := s.tooManyEmojis(msg.Text, maxEmojiAllowed)
	stopWordsSpam := s.stopWords(msg.Text)
	similaritySpam := s.isSpamSimilarity(msg.Text)
	if similaritySpam || isEmojiSpam || stopWordsSpam || s.isCasSpam(msg.From.ID) {
		log.Printf("[INFO] user %s detected as spammer, msg: %q", displayUsername, msg.Text)
		if s.Dry {
			return Response{
				Text: fmt.Sprintf("this is spam from %q, but I'm in dry mode, so I'll do nothing yet", displayUsername),
				Send: true, ReplyTo: msg.ID,
			}
		}
		return Response{Text: fmt.Sprintf("this is spam! go to ban, %q (id:%d)", displayUsername, msg.From.ID),
			Send: true, ReplyTo: msg.ID, BanInterval: permanentBanDuration, DeleteReplyTo: true,
			User: User{Username: msg.From.Username, ID: msg.From.ID, DisplayName: msg.From.DisplayName},
		}
	}

	if id := msg.From.ID; id != 0 {
		s.approvedUsers[id] = true
		log.Printf("[INFO] user %s is not a spammer id %d, added to aproved", displayUsername, msg.From.ID)
	}
	return Response{} // not a spam
}

// Help returns help message
func (s *SpamFilter) Help() string { return "" }

// ReactOn keys
func (s *SpamFilter) ReactOn() []string { return []string{} }

func (s *SpamFilter) isCasSpam(msgID int64) bool {
	reqURL := fmt.Sprintf("%s/check?user_id=%d", s.CasAPI, msgID)
	req, err := http.NewRequest("GET", reqURL, http.NoBody)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return false
	}

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return false
	}
	defer resp.Body.Close()

	respData := struct {
		OK          bool   `json:"ok"` // ok means user is a spammer
		Description string `json:"description"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		log.Printf("[WARN] failed to parse response from %s, error=%v", reqURL, err)
		return false
	}
	if respData.OK {
		log.Printf("[INFO] user %q detected as spammer: %s", msgID, respData.Description)
	}
	return respData.OK
}

// isSpam checks if a given message is similar to any of the known bad messages.
func (s *SpamFilter) isSpamSimilarity(message string) bool {
	// check for spam similarity
	tokenizedMessage := s.tokenize(message)
	maxSimilarity, maxTokens, maxID := 0.0, map[string]int{}, 0
	for i, spam := range s.tokenizedSpam {
		similarity := s.cosineSimilarity(tokenizedMessage, spam)
		if similarity > maxSimilarity {
			maxSimilarity = similarity
			maxTokens = spam
			maxID = i + 1
		}
		if similarity >= s.SimilarityThreshold {
			log.Printf("[DEBUG] high spam similarity: %0.2f, line %d, toekns: %v",
				maxSimilarity, maxID, maxTokens)
			return true
		}
	}
	log.Printf("[DEBUG] low spam similarity: %0.2f", maxSimilarity)
	return false
}

// tokenize takes a string and returns a map where the keys are unique words (tokens)
// and the values are the frequencies of those words in the string.
func (s *SpamFilter) tokenize(inp string) map[string]int {

	isExcludedToken := func(token string) bool {
		list := []string{
			// english
			"and", "the", "is", "in", "on", "at", "for", "with", "not",
			"by", "be", "this", "are", "from", "or", "that", "an", "it",
			"his", "but", "he", "she", "as", "you", "do", "their", "all",
			"will", "there", "can", "i", "me", "my", "myself", "we", "our", "ours", "ourselves",
			"your", "yours", "yourself", "yourselves", "him", "himself",
			"her", "hers", "herself", "its", "itself", "they", "them",
			"theirs", "themselves", "what", "which", "who", "whom",
			"these", "those", "am", "was", "were", "been", "being", "have",
			"has", "had", "having", "does", "did", "a", "if", "because",
			"until", "while", "of", "about", "against", "between", "into",
			"through", "during", "before", "after", "above", "below", "to",
			"up", "down", "out", "off", "over", "under", "again", "further",
			"then", "once", "here", "why", "how", "any", "both", "each",
			"few", "more", "most", "other", "some", "such", "no", "nor",
			"only", "own", "same", "so", "than", "too", "very", "s", "t",
			"just", "don", "should", "now",

			// russian
			"а", "без", "более", "больше", "будет", "будто", "бы", "был", "была", "были",
			"было", "быть", "в", "вам", "вас", "вдруг", "ведь", "во", "вот", "впрочем",
			"все", "всегда", "всего", "всех", "всю", "вы", "где", "да", "даже", "два",
			"для", "до", "другой", "его", "ее", "если", "есть", "еще", "же", "за", "здесь",
			"и", "из", "или", "им", "иногда", "их", "к", "как", "какая", "какой", "когда",
			"конечно", "которого", "которые", "кто", "куда", "ли", "лучше", "между",
			"меня", "мне", "много", "может", "можно", "мой", "моя", "мы", "на", "над",
			"надо", "наконец", "нас", "не", "него", "нее", "нельзя", "нет", "ни", "нибудь",
			"никогда", "ним", "них", "ничего", "но", "ну", "о", "об", "один", "он", "она",
			"они", "оно", "опять", "от", "перед", "по", "под", "после", "потом", "потому",
			"почти", "при", "про", "раз", "разве", "с", "сам", "свое", "свою", "себе",
			"себя", "сегодня", "сейчас", "сказал", "сказала", "сказать", "со", "совсем",
			"так", "такой", "там", "тебя", "тем", "теперь", "то", "тогда", "того", "тоже",
			"только", "том", "тот", "три", "тут", "ты", "у", "уж", "уже", "хорошо", "хоть",
			"чего", "чей", "чем", "через", "что", "чтоб", "чтобы", "чуть", "эти", "этого",
			"этой", "этом", "этот", "эту", "я",
		}
		for _, w := range list {
			if strings.EqualFold(token, w) || len([]rune(token)) < 3 {
				return true
			}
		}
		return false
	}

	tokenFrequency := make(map[string]int)
	tokens := strings.Fields(inp)
	for _, token := range tokens {
		if isExcludedToken(token) {
			continue
		}
		token = s.cleanEmoji(token)
		token = strings.Trim(token, ".,!?-")
		token = strings.ToLower(token)
		tokenFrequency[strings.ToLower(token)]++
	}
	return tokenFrequency
}

// cosineSimilarity calculates the cosine similarity between two token frequency maps.
func (s *SpamFilter) cosineSimilarity(a, b map[string]int) float64 {
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

func (s *SpamFilter) stopWords(message string) bool {
	lowerCaseMessage := strings.ToLower(message)
	lowerCaseMessage = emojiPattern.ReplaceAllString(lowerCaseMessage, "")
	for _, word := range stopWords {
		if strings.Contains(lowerCaseMessage, strings.ToLower(word)) {
			log.Printf("[DEBUG] spam stop word %q", word)
			return true
		}
	}
	return false
}

func (s *SpamFilter) tooManyEmojis(message string, threshold int) (ok bool, count int) {
	matches := emojiPattern.FindAllString(message, -1)
	if len(matches) > threshold {
		log.Printf("[DEBUG] spam emojis, %d of %d", len(matches), threshold)
		return true, len(matches)
	}
	return false, len(matches)
}

func (s *SpamFilter) cleanEmoji(message string) string {
	return emojiPattern.ReplaceAllString(message, "")
}

// borrowed from https://stackoverflow.com/a/72255061
var emojiPattern = regexp.MustCompile(`[#*0-9]\x{FE0F}?\x{20E3}|©\x{FE0F}?|[®\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{2199}\x{21A9}\x{21AA}]\x{FE0F}?|[\x{231A}\x{231B}]|[\x{2328}\x{23CF}]\x{FE0F}?|[\x{23E9}-\x{23EC}]|[\x{23ED}-\x{23EF}]\x{FE0F}?|\x{23F0}|[\x{23F1}\x{23F2}]\x{FE0F}?|\x{23F3}|[\x{23F8}-\x{23FA}\x{24C2}\x{25AA}\x{25AB}\x{25B6}\x{25C0}\x{25FB}\x{25FC}]\x{FE0F}?|[\x{25FD}\x{25FE}]|[\x{2600}-\x{2604}\x{260E}\x{2611}]\x{FE0F}?|[\x{2614}\x{2615}]|\x{2618}\x{FE0F}?|\x{261D}[\x{FE0F}\x{1F3FB}-\x{1F3FF}]?|[\x{2620}\x{2622}\x{2623}\x{2626}\x{262A}\x{262E}\x{262F}\x{2638}-\x{263A}\x{2640}\x{2642}]\x{FE0F}?|[\x{2648}-\x{2653}]|[\x{265F}\x{2660}\x{2663}\x{2665}\x{2666}\x{2668}\x{267B}\x{267E}]\x{FE0F}?|\x{267F}|\x{2692}\x{FE0F}?|\x{2693}|[\x{2694}-\x{2697}\x{2699}\x{269B}\x{269C}\x{26A0}]\x{FE0F}?|\x{26A1}|\x{26A7}\x{FE0F}?|[\x{26AA}\x{26AB}]|[\x{26B0}\x{26B1}]\x{FE0F}?|[\x{26BD}\x{26BE}\x{26C4}\x{26C5}]|\x{26C8}\x{FE0F}?|\x{26CE}|[\x{26CF}\x{26D1}\x{26D3}]\x{FE0F}?|\x{26D4}|\x{26E9}\x{FE0F}?|\x{26EA}|[\x{26F0}\x{26F1}]\x{FE0F}?|[\x{26F2}\x{26F3}]|\x{26F4}\x{FE0F}?|\x{26F5}|[\x{26F7}\x{26F8}]\x{FE0F}?|\x{26F9}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{FE0F}\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{26FA}\x{26FD}]|\x{2702}\x{FE0F}?|\x{2705}|[\x{2708}\x{2709}]\x{FE0F}?|[\x{270A}\x{270B}][\x{1F3FB}-\x{1F3FF}]?|[\x{270C}\x{270D}][\x{FE0F}\x{1F3FB}-\x{1F3FF}]?|\x{270F}\x{FE0F}?|[\x{2712}\x{2714}\x{2716}\x{271D}\x{2721}]\x{FE0F}?|\x{2728}|[\x{2733}\x{2734}\x{2744}\x{2747}]\x{FE0F}?|[\x{274C}\x{274E}\x{2753}-\x{2755}\x{2757}]|\x{2763}\x{FE0F}?|\x{2764}(?:\x{200D}[\x{1F525}\x{1FA79}]|\x{FE0F}(?:\x{200D}[\x{1F525}\x{1FA79}])?)?|[\x{2795}-\x{2797}]|\x{27A1}\x{FE0F}?|[\x{27B0}\x{27BF}]|[\x{2934}\x{2935}\x{2B05}-\x{2B07}]\x{FE0F}?|[\x{2B1B}\x{2B1C}\x{2B50}\x{2B55}]|[\x{3030}\x{303D}\x{3297}\x{3299}]\x{FE0F}?|[\x{1F004}\x{1F0CF}]|[\x{1F170}\x{1F171}\x{1F17E}\x{1F17F}]\x{FE0F}?|[\x{1F18E}\x{1F191}-\x{1F19A}]|\x{1F1E6}[\x{1F1E8}-\x{1F1EC}\x{1F1EE}\x{1F1F1}\x{1F1F2}\x{1F1F4}\x{1F1F6}-\x{1F1FA}\x{1F1FC}\x{1F1FD}\x{1F1FF}]|\x{1F1E7}[\x{1F1E6}\x{1F1E7}\x{1F1E9}-\x{1F1EF}\x{1F1F1}-\x{1F1F4}\x{1F1F6}-\x{1F1F9}\x{1F1FB}\x{1F1FC}\x{1F1FE}\x{1F1FF}]|\x{1F1E8}[\x{1F1E6}\x{1F1E8}\x{1F1E9}\x{1F1EB}-\x{1F1EE}\x{1F1F0}-\x{1F1F5}\x{1F1F7}\x{1F1FA}-\x{1F1FF}]|\x{1F1E9}[\x{1F1EA}\x{1F1EC}\x{1F1EF}\x{1F1F0}\x{1F1F2}\x{1F1F4}\x{1F1FF}]|\x{1F1EA}[\x{1F1E6}\x{1F1E8}\x{1F1EA}\x{1F1EC}\x{1F1ED}\x{1F1F7}-\x{1F1FA}]|\x{1F1EB}[\x{1F1EE}-\x{1F1F0}\x{1F1F2}\x{1F1F4}\x{1F1F7}]|\x{1F1EC}[\x{1F1E6}\x{1F1E7}\x{1F1E9}-\x{1F1EE}\x{1F1F1}-\x{1F1F3}\x{1F1F5}-\x{1F1FA}\x{1F1FC}\x{1F1FE}]|\x{1F1ED}[\x{1F1F0}\x{1F1F2}\x{1F1F3}\x{1F1F7}\x{1F1F9}\x{1F1FA}]|\x{1F1EE}[\x{1F1E8}-\x{1F1EA}\x{1F1F1}-\x{1F1F4}\x{1F1F6}-\x{1F1F9}]|\x{1F1EF}[\x{1F1EA}\x{1F1F2}\x{1F1F4}\x{1F1F5}]|\x{1F1F0}[\x{1F1EA}\x{1F1EC}-\x{1F1EE}\x{1F1F2}\x{1F1F3}\x{1F1F5}\x{1F1F7}\x{1F1FC}\x{1F1FE}\x{1F1FF}]|\x{1F1F1}[\x{1F1E6}-\x{1F1E8}\x{1F1EE}\x{1F1F0}\x{1F1F7}-\x{1F1FB}\x{1F1FE}]|\x{1F1F2}[\x{1F1E6}\x{1F1E8}-\x{1F1ED}\x{1F1F0}-\x{1F1FF}]|\x{1F1F3}[\x{1F1E6}\x{1F1E8}\x{1F1EA}-\x{1F1EC}\x{1F1EE}\x{1F1F1}\x{1F1F4}\x{1F1F5}\x{1F1F7}\x{1F1FA}\x{1F1FF}]|\x{1F1F4}\x{1F1F2}|\x{1F1F5}[\x{1F1E6}\x{1F1EA}-\x{1F1ED}\x{1F1F0}-\x{1F1F3}\x{1F1F7}-\x{1F1F9}\x{1F1FC}\x{1F1FE}]|\x{1F1F6}\x{1F1E6}|\x{1F1F7}[\x{1F1EA}\x{1F1F4}\x{1F1F8}\x{1F1FA}\x{1F1FC}]|\x{1F1F8}[\x{1F1E6}-\x{1F1EA}\x{1F1EC}-\x{1F1F4}\x{1F1F7}-\x{1F1F9}\x{1F1FB}\x{1F1FD}-\x{1F1FF}]|\x{1F1F9}[\x{1F1E6}\x{1F1E8}\x{1F1E9}\x{1F1EB}-\x{1F1ED}\x{1F1EF}-\x{1F1F4}\x{1F1F7}\x{1F1F9}\x{1F1FB}\x{1F1FC}\x{1F1FF}]|\x{1F1FA}[\x{1F1E6}\x{1F1EC}\x{1F1F2}\x{1F1F3}\x{1F1F8}\x{1F1FE}\x{1F1FF}]|\x{1F1FB}[\x{1F1E6}\x{1F1E8}\x{1F1EA}\x{1F1EC}\x{1F1EE}\x{1F1F3}\x{1F1FA}]|\x{1F1FC}[\x{1F1EB}\x{1F1F8}]|\x{1F1FD}\x{1F1F0}|\x{1F1FE}[\x{1F1EA}\x{1F1F9}]|\x{1F1FF}[\x{1F1E6}\x{1F1F2}\x{1F1FC}]|\x{1F201}|\x{1F202}\x{FE0F}?|[\x{1F21A}\x{1F22F}\x{1F232}-\x{1F236}]|\x{1F237}\x{FE0F}?|[\x{1F238}-\x{1F23A}\x{1F250}\x{1F251}\x{1F300}-\x{1F320}]|[\x{1F321}\x{1F324}-\x{1F32C}]\x{FE0F}?|[\x{1F32D}-\x{1F335}]|\x{1F336}\x{FE0F}?|[\x{1F337}-\x{1F37C}]|\x{1F37D}\x{FE0F}?|[\x{1F37E}-\x{1F384}]|\x{1F385}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F386}-\x{1F393}]|[\x{1F396}\x{1F397}\x{1F399}-\x{1F39B}\x{1F39E}\x{1F39F}]\x{FE0F}?|[\x{1F3A0}-\x{1F3C1}]|\x{1F3C2}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F3C3}\x{1F3C4}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F3C5}\x{1F3C6}]|\x{1F3C7}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F3C8}\x{1F3C9}]|\x{1F3CA}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F3CB}\x{1F3CC}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{FE0F}\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F3CD}\x{1F3CE}]\x{FE0F}?|[\x{1F3CF}-\x{1F3D3}]|[\x{1F3D4}-\x{1F3DF}]\x{FE0F}?|[\x{1F3E0}-\x{1F3F0}]|\x{1F3F3}(?:\x{200D}(?:\x{26A7}\x{FE0F}?|\x{1F308})|\x{FE0F}(?:\x{200D}(?:\x{26A7}\x{FE0F}?|\x{1F308}))?)?|\x{1F3F4}(?:\x{200D}\x{2620}\x{FE0F}?|\x{E0067}\x{E0062}(?:\x{E0065}\x{E006E}\x{E0067}|\x{E0073}\x{E0063}\x{E0074}|\x{E0077}\x{E006C}\x{E0073})\x{E007F})?|[\x{1F3F5}\x{1F3F7}]\x{FE0F}?|[\x{1F3F8}-\x{1F407}]|\x{1F408}(?:\x{200D}\x{2B1B})?|[\x{1F409}-\x{1F414}]|\x{1F415}(?:\x{200D}\x{1F9BA})?|[\x{1F416}-\x{1F43A}]|\x{1F43B}(?:\x{200D}\x{2744}\x{FE0F}?)?|[\x{1F43C}-\x{1F43E}]|\x{1F43F}\x{FE0F}?|\x{1F440}|\x{1F441}(?:\x{200D}\x{1F5E8}\x{FE0F}?|\x{FE0F}(?:\x{200D}\x{1F5E8}\x{FE0F}?)?)?|[\x{1F442}\x{1F443}][\x{1F3FB}-\x{1F3FF}]?|[\x{1F444}\x{1F445}]|[\x{1F446}-\x{1F450}][\x{1F3FB}-\x{1F3FF}]?|[\x{1F451}-\x{1F465}]|[\x{1F466}\x{1F467}][\x{1F3FB}-\x{1F3FF}]?|\x{1F468}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D})?\x{1F468}|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}]|\x{1F466}(?:\x{200D}\x{1F466})?|\x{1F467}(?:\x{200D}[\x{1F466}\x{1F467}])?|[\x{1F468}\x{1F469}]\x{200D}(?:\x{1F466}(?:\x{200D}\x{1F466})?|\x{1F467}(?:\x{200D}[\x{1F466}\x{1F467}])?)|[\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}])|\x{1F3FB}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D})?\x{1F468}[\x{1F3FB}-\x{1F3FF}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F468}[\x{1F3FC}-\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FC}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D})?\x{1F468}[\x{1F3FB}-\x{1F3FF}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F468}[\x{1F3FB}\x{1F3FD}-\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FD}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D})?\x{1F468}[\x{1F3FB}-\x{1F3FF}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F468}[\x{1F3FB}\x{1F3FC}\x{1F3FE}\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FE}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D})?\x{1F468}[\x{1F3FB}-\x{1F3FF}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F468}[\x{1F3FB}-\x{1F3FD}\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FF}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D})?\x{1F468}[\x{1F3FB}-\x{1F3FF}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F468}[\x{1F3FB}-\x{1F3FE}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?)?|\x{1F469}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D})?[\x{1F468}\x{1F469}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}]|\x{1F466}(?:\x{200D}\x{1F466})?|\x{1F467}(?:\x{200D}[\x{1F466}\x{1F467}])?|\x{1F469}\x{200D}(?:\x{1F466}(?:\x{200D}\x{1F466})?|\x{1F467}(?:\x{200D}[\x{1F466}\x{1F467}])?)|[\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}])|\x{1F3FB}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}]|\x{1F48B}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}])|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FC}-\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FC}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}]|\x{1F48B}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}])|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FB}\x{1F3FD}-\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FD}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}]|\x{1F48B}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}])|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FB}\x{1F3FC}\x{1F3FE}\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FE}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}]|\x{1F48B}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}])|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FD}\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FF}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}]|\x{1F48B}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FF}])|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}[\x{1F468}\x{1F469}][\x{1F3FB}-\x{1F3FE}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?)?|\x{1F46A}|[\x{1F46B}-\x{1F46D}][\x{1F3FB}-\x{1F3FF}]?|\x{1F46E}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F46F}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?|[\x{1F470}\x{1F471}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F472}[\x{1F3FB}-\x{1F3FF}]?|\x{1F473}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F474}-\x{1F476}][\x{1F3FB}-\x{1F3FF}]?|\x{1F477}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F478}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F479}-\x{1F47B}]|\x{1F47C}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F47D}-\x{1F480}]|[\x{1F481}\x{1F482}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F483}[\x{1F3FB}-\x{1F3FF}]?|\x{1F484}|\x{1F485}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F486}\x{1F487}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F488}-\x{1F48E}]|\x{1F48F}[\x{1F3FB}-\x{1F3FF}]?|\x{1F490}|\x{1F491}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F492}-\x{1F4A9}]|\x{1F4AA}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F4AB}-\x{1F4FC}]|\x{1F4FD}\x{FE0F}?|[\x{1F4FF}-\x{1F53D}]|[\x{1F549}\x{1F54A}]\x{FE0F}?|[\x{1F54B}-\x{1F54E}\x{1F550}-\x{1F567}]|[\x{1F56F}\x{1F570}\x{1F573}]\x{FE0F}?|\x{1F574}[\x{FE0F}\x{1F3FB}-\x{1F3FF}]?|\x{1F575}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{FE0F}\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F576}-\x{1F579}]\x{FE0F}?|\x{1F57A}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F587}\x{1F58A}-\x{1F58D}]\x{FE0F}?|\x{1F590}[\x{FE0F}\x{1F3FB}-\x{1F3FF}]?|[\x{1F595}\x{1F596}][\x{1F3FB}-\x{1F3FF}]?|\x{1F5A4}|[\x{1F5A5}\x{1F5A8}\x{1F5B1}\x{1F5B2}\x{1F5BC}\x{1F5C2}-\x{1F5C4}\x{1F5D1}-\x{1F5D3}\x{1F5DC}-\x{1F5DE}\x{1F5E1}\x{1F5E3}\x{1F5E8}\x{1F5EF}\x{1F5F3}\x{1F5FA}]\x{FE0F}?|[\x{1F5FB}-\x{1F62D}]|\x{1F62E}(?:\x{200D}\x{1F4A8})?|[\x{1F62F}-\x{1F634}]|\x{1F635}(?:\x{200D}\x{1F4AB})?|\x{1F636}(?:\x{200D}\x{1F32B}\x{FE0F}?)?|[\x{1F637}-\x{1F644}]|[\x{1F645}-\x{1F647}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F648}-\x{1F64A}]|\x{1F64B}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F64C}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F64D}\x{1F64E}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F64F}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F680}-\x{1F6A2}]|\x{1F6A3}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F6A4}-\x{1F6B3}]|[\x{1F6B4}-\x{1F6B6}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F6B7}-\x{1F6BF}]|\x{1F6C0}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F6C1}-\x{1F6C5}]|\x{1F6CB}\x{FE0F}?|\x{1F6CC}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F6CD}-\x{1F6CF}]\x{FE0F}?|[\x{1F6D0}-\x{1F6D2}\x{1F6D5}-\x{1F6D7}\x{1F6DD}-\x{1F6DF}]|[\x{1F6E0}-\x{1F6E5}\x{1F6E9}]\x{FE0F}?|[\x{1F6EB}\x{1F6EC}]|[\x{1F6F0}\x{1F6F3}]\x{FE0F}?|[\x{1F6F4}-\x{1F6FC}\x{1F7E0}-\x{1F7EB}\x{1F7F0}]|\x{1F90C}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F90D}\x{1F90E}]|\x{1F90F}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F910}-\x{1F917}]|[\x{1F918}-\x{1F91F}][\x{1F3FB}-\x{1F3FF}]?|[\x{1F920}-\x{1F925}]|\x{1F926}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F927}-\x{1F92F}]|[\x{1F930}-\x{1F934}][\x{1F3FB}-\x{1F3FF}]?|\x{1F935}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F936}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F937}-\x{1F939}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F93A}|\x{1F93C}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?|[\x{1F93D}\x{1F93E}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F93F}-\x{1F945}\x{1F947}-\x{1F976}]|\x{1F977}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F978}-\x{1F9B4}]|[\x{1F9B5}\x{1F9B6}][\x{1F3FB}-\x{1F3FF}]?|\x{1F9B7}|[\x{1F9B8}\x{1F9B9}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F9BA}|\x{1F9BB}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F9BC}-\x{1F9CC}]|[\x{1F9CD}-\x{1F9CF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F9D0}|\x{1F9D1}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F384}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F9D1}|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}])|\x{1F3FB}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D}|)\x{1F9D1}[\x{1F3FC}-\x{1F3FF}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F384}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F9D1}[\x{1F3FB}-\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FC}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D}|)\x{1F9D1}[\x{1F3FB}\x{1F3FD}-\x{1F3FF}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F384}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F9D1}[\x{1F3FB}-\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FD}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D}|)\x{1F9D1}[\x{1F3FB}\x{1F3FC}\x{1F3FE}\x{1F3FF}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F384}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F9D1}[\x{1F3FB}-\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FE}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D}|)\x{1F9D1}[\x{1F3FB}-\x{1F3FD}\x{1F3FF}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F384}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F9D1}[\x{1F3FB}-\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?|\x{1F3FF}(?:\x{200D}(?:[\x{2695}\x{2696}\x{2708}]\x{FE0F}?|\x{2764}\x{FE0F}?\x{200D}(?:\x{1F48B}\x{200D}|)\x{1F9D1}[\x{1F3FB}-\x{1F3FE}]|[\x{1F33E}\x{1F373}\x{1F37C}\x{1F384}\x{1F393}\x{1F3A4}\x{1F3A8}\x{1F3EB}\x{1F3ED}\x{1F4BB}\x{1F4BC}\x{1F527}\x{1F52C}\x{1F680}\x{1F692}]|\x{1F91D}\x{200D}\x{1F9D1}[\x{1F3FB}-\x{1F3FF}]|[\x{1F9AF}-\x{1F9B3}\x{1F9BC}\x{1F9BD}]))?)?|[\x{1F9D2}\x{1F9D3}][\x{1F3FB}-\x{1F3FF}]?|\x{1F9D4}(?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|\x{1F9D5}[\x{1F3FB}-\x{1F3FF}]?|[\x{1F9D6}-\x{1F9DD}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?|[\x{1F3FB}-\x{1F3FF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?)?|[\x{1F9DE}\x{1F9DF}](?:\x{200D}[\x{2640}\x{2642}]\x{FE0F}?)?|[\x{1F9E0}-\x{1F9FF}\x{1FA70}-\x{1FA74}\x{1FA78}-\x{1FA7C}\x{1FA80}-\x{1FA86}\x{1FA90}-\x{1FAAC}\x{1FAB0}-\x{1FABA}\x{1FAC0}-\x{1FAC2}]|[\x{1FAC3}-\x{1FAC5}][\x{1F3FB}-\x{1F3FF}]?|[\x{1FAD0}-\x{1FAD9}\x{1FAE0}-\x{1FAE7}]|\x{1FAF0}[\x{1F3FB}-\x{1F3FF}]?|\x{1FAF1}(?:\x{1F3FB}(?:\x{200D}\x{1FAF2}[\x{1F3FC}-\x{1F3FF}])?|\x{1F3FC}(?:\x{200D}\x{1FAF2}[\x{1F3FB}\x{1F3FD}-\x{1F3FF}])?|\x{1F3FD}(?:\x{200D}\x{1FAF2}[\x{1F3FB}\x{1F3FC}\x{1F3FE}\x{1F3FF}])?|\x{1F3FE}(?:\x{200D}\x{1FAF2}[\x{1F3FB}-\x{1F3FD}\x{1F3FF}])?|\x{1F3FF}(?:\x{200D}\x{1FAF2}[\x{1F3FB}-\x{1F3FE}])?)?|[\x{1FAF2}-\x{1FAF6}][\x{1F3FB}-\x{1F3FF}]?`)
