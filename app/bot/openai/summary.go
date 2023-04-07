package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-pkgz/lcw"
	"github.com/go-pkgz/syncs"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate moq --out mocks/openai_summary.go --pkg mocks --skip-ensure . openAISummary:OpenAISummary
//go:generate moq --out mocks/remark.go --pkg mocks --skip-ensure . remarkCommentsGetter:RemarkClient
//go:generate moq --out mocks/ukeeper.go --pkg mocks --skip-ensure . uKeeperGetter:UKeeperClient

// Summarizer is a helper for summarizing articles by links
type Summarizer struct {
	openAISummary openAISummary
	remark        remarkCommentsGetter
	uKeeper       uKeeperGetter

	// Cache is using to optimize `Summary` calls (generating summary by link)
	// In debug mode cache is dumped to local file
	cache lcw.LoadingCache

	threads int
	debug   bool
}

type uKeeperGetter interface {
	Get(link string) (title, content string, err error)
}

type remarkCommentsGetter interface {
	GetTopComments(remarkLink string) (comments []string, links []string, err error)
}

type openAISummary interface {
	Summary(text string) (response string, err error)
}

// NewSummarizer creates new summarizer object
// If debug is true, it loads cache from file
func NewSummarizer(openAISummary openAISummary, remark remarkCommentsGetter, uKeeper uKeeperGetter, threads int, debug bool) Summarizer {
	cache, _ := lcw.NewExpirableCache(lcw.MaxKeys(100), lcw.TTL(time.Hour*12))

	if debug {
		err := loadBackup(cache)
		if err != nil {
			log.Printf("[WARN] can't apply backup: %v", err)
		}
	}

	return Summarizer{
		openAISummary: openAISummary,
		remark:        remark,
		uKeeper:       uKeeper,
		cache:         cache,
		threads:       threads,
		debug:         debug,
	}
}

// GetSummariesByMessage returns summary for the link in the message
// If this is a remark link, then it returns summaries for all comments
func (s Summarizer) GetSummariesByMessage(message string) (messages []string, err error) {
	link, err := getLink(message)
	if err != nil {
		return []string{link}, err
	}

	// if link is a remark link on radio-t.com, then we need to links by comments
	if strings.Contains(link, "radio-t.com") {
		err = matchRemarkLink(link)
		if err != nil {
			return []string{link}, err
		}
		return s.GetSummariesByRemarkLink(link)
	}

	summary, err := s.Summary(link)
	if err != nil {
		return []string{link}, err
	}
	messages = append(messages, summary)

	return messages, nil
}

func getLink(message string) (link string, err error) {
	re := regexp.MustCompile(`https?://[^\s"'<>]+`)
	link = re.FindString(message)

	log.Printf("[DEBUG] Link found: %s", link)

	if link == "" {
		return "", fmt.Errorf("no link found in message: %s", message)
	}

	return link, nil
}

func matchRemarkLink(remarkLink string) error {
	re := regexp.MustCompile(`https?://radio-t\.com/p/[^\s"'<>]+/prep-\d+/`)
	if re.MatchString(remarkLink) {
		return nil
	}

	return fmt.Errorf("radio-t link doesn't fit to format: %s", remarkLink) // ignore radio-t.com links
}

// GetSummariesByRemarkLink returns summaries for all comments by the remark link
func (s Summarizer) GetSummariesByRemarkLink(remarkLink string) (messages []string, err error) {
	comments, links, err := s.remark.GetTopComments(remarkLink)
	if err != nil {
		return []string{}, fmt.Errorf("can't get comments: %w", err)
	}

	var mutex = &sync.Mutex{}
	withLock := func(f func()) {
		mutex.Lock()
		defer mutex.Unlock()
		f()
	}

	messages = make([]string, len(comments))
	swg := syncs.NewSizedGroup(s.threads)
	for i, c := range comments {
		taskNum := i
		withLock(func() {
			messages[taskNum] += fmt.Sprintf("[%d/%d] %s", i+1, len(comments), c)
		})

		link := links[taskNum]
		log.Printf("[DEBUG] Get summary %d %s", i, link)
		if link == "" {
			continue
		}

		swg.Go(func(ctx context.Context) {
			summary, err := s.Summary(link)
			if err != nil {
				log.Printf("[WARN] can't get summary for %s: %v", link, err)
				withLock(func() {
					messages[taskNum] += fmt.Sprintf("\n\nError: <pre>%v</pre>", err)
				})
				return
			}

			withLock(func() {
				messages[taskNum] += fmt.Sprintf("\n\n%s", summary)
			})
		})
	}
	swg.Wait()

	return messages, nil
}

// Summary returns summary for link
// It uses cache for links that was already summarized
// If debug is true, it saves cache to file
// Important: this isn't thread safe
func (s Summarizer) Summary(link string) (summary string, err error) {
	item, err := s.cache.Get(link, func() (interface{}, error) { return s.summaryInternal(link) })
	if err != nil {
		return "", err
	}

	if s.debug {
		err := saveBackup(s.cache)
		if err != nil {
			log.Printf("[WARN] can't save backup: %v", err)
		}
	}

	return item.(summaryItem).render(), nil
}

func (s Summarizer) summaryInternal(link string) (item summaryItem, err error) {
	log.Printf("[DEBUG] summary for link:%s", link)

	title, content, err := s.uKeeper.Get(link)
	if err != nil {
		return summaryItem{}, fmt.Errorf("can't get content for %s: %w", link, err)
	}

	res, err := s.openAISummary.Summary(title + " - " + content)
	if err != nil {
		return summaryItem{}, fmt.Errorf("can't get summary for %s: %w", link, err)
	}

	result := summaryItem{
		Title:   title,
		Content: res,
	}

	return result, nil
}

type summaryItem struct {
	Title   string `json:"Title"`
	Content string `json:"Content"`
}

// render telegram message
func (s summaryItem) render() string {
	if s.isEmpty() {
		return ""
	}

	title := tbapi.EscapeText(tbapi.ModeHTML, s.Title)
	content := tbapi.EscapeText(tbapi.ModeHTML, s.Content)
	return fmt.Sprintf("<b>%s</b>\n\n%s", title, content)
}

func (s summaryItem) isEmpty() bool {
	return s.Title == "" || s.Content == ""
}

// loadBackup loads cache from local file in debug mode
func loadBackup(cache lcw.LoadingCache) error {
	type LocalCache struct {
		Summaries map[string]summaryItem `json:"Summaries"`
	}

	lc := LocalCache{
		Summaries: make(map[string]summaryItem),
	}

	data, err := os.ReadFile("cache_openai.json")
	if err != nil {
		return fmt.Errorf("can't open cache file, %v", err)
	}
	if err := json.Unmarshal(data, &lc); err != nil {
		return fmt.Errorf("can't unmarshal cache file, %v", err)
	}

	for k := range lc.Summaries {
		_, _ = cache.Get(k, func() (interface{}, error) {
			return lc.Summaries[k], nil
		})
	}

	return nil
}

// saveBackup saves cache to local file in debug mode
func saveBackup(cache lcw.LoadingCache) error {
	type LocalCache struct {
		Summaries map[string]summaryItem `json:"Summaries"`
	}

	lc := LocalCache{
		Summaries: make(map[string]summaryItem),
	}

	for _, k := range cache.Keys() {
		v, _ := cache.Get(k, func() (interface{}, error) {
			return nil, nil
		})

		if v != nil {
			lc.Summaries[k] = v.(summaryItem)
		}
	}

	data, err := json.Marshal(lc)
	if err != nil {
		return err
	}

	return os.WriteFile("cache_openai.json", data, 0o600)
}
