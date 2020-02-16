package reporter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	"github.com/radio-t/super-bot/app/bot"
)

// Exporter performs conversion from log file to html
type Exporter struct {
	ExporterParams
	location      *time.Location
	fileRecipient FileRecipient
	storage       Storage

	fileIDToURL map[string]string
}

// ExporterParams for locations
type ExporterParams struct {
	OutputRoot     string
	InputRoot      string
	TemplateFile   string
	BotUsername    string
	SuperUsers     SuperUser
	BroadcastUsers SuperUser // Users who can send "bot.MsgBroadcastStarted" and "bot.MsgBroadcastStarted" messages.
	// it maybe just bot, or bot + some or all SuperUsers.
	// Cannot use SuperUsers field for same purpose becase they used to mark messages as "from host" in template
}

// SuperUser knows which user is a superuser
type SuperUser interface {
	IsSuper(user string) bool
}

// Storage knows how to: create file, check for file existence
// and build a public-accessible link (relative in our case)
type Storage interface {
	FileExists(fileName string) (bool, error)
	CreateFile(fileName string, body []byte) (string, error)
	BuildLink(fileName string) string
	BuildPath(fileName string) string
}

// NewExporter from params, initializes time.Location
func NewExporter(fileRecipient FileRecipient, storage Storage, params ExporterParams) *Exporter {
	log.Printf("[INFO] exporter with %v", params)
	result := Exporter{
		ExporterParams: params,
		fileRecipient:  fileRecipient,
		storage:        storage,
		fileIDToURL:    map[string]string{},
	}

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalf("[ERROR] can't load location, error %v", err)
	}
	result.location = location
	return &result
}

// Export to html with showNum
func (e *Exporter) Export(showNum int, yyyymmdd int) error {
	from := fmt.Sprintf("%s/%s.log", e.InputRoot, time.Now().Format("20060102")) // current day by default
	if yyyymmdd != 0 {
		from = fmt.Sprintf("%s/%d.log", e.InputRoot, yyyymmdd)
	}
	to := fmt.Sprintf("%s/radio-t-%d.html", e.OutputRoot, showNum)

	messages, err := readMessages(from, e.ExporterParams.BroadcastUsers)
	if err != nil {
		return errors.Wrapf(err, "failed to read messages from %s", from)
	}

	fh, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666) // nolint
	if err != nil {
		return errors.Wrapf(err, "failed to open destination file %s", to)
	}

	defer func() {
		if err := fh.Close(); err != nil {
			log.Printf("[WARN] failed to close %s, %v", fh.Name(), err)
		}
	}()

	h, err := e.toHTML(messages, showNum)
	if err != nil {
		return errors.Wrapf(err, "can't export #%d", showNum)
	}

	if _, err = fh.WriteString(h); err != nil {
		return errors.Wrapf(err, "failed to write HTML to file %s", to)
	}

	log.Printf("[INFO] exported %d lines to %s", len(messages), to)
	return nil
}

func (e *Exporter) toHTML(messages []bot.Message, num int) (string, error) {

	type Record struct {
		Time   string
		Msg    bot.Message
		IsHost bool
		IsBot  bool
	}

	type Data struct {
		Num     int
		Records []Record
	}

	data := Data{Num: num}
	for _, msg := range messages {

		if msg.Image != nil {
			if err := e.maybeDownloadFile(msg.Image.FileID); err != nil {
				log.Printf("[WARN] failed to download, %v", err)
			}
		}

		data.Records = append(
			data.Records,
			Record{
				Time:   msg.Sent.In(e.location).Format("15:04:05"),
				Msg:    msg,
				IsHost: e.SuperUsers.IsSuper(msg.From.Username),
				IsBot:  msg.From.Username == e.BotUsername,
			},
		)
	}

	funcMap := template.FuncMap{
		"fileURL": func(fileID string) string {
			if url, found := e.fileIDToURL[fileID]; found {
				return url
			}
			return ""
		},
		"timestampHuman": e.timestampHuman,
		"format":         format,
	}
	name := e.TemplateFile[strings.LastIndex(e.TemplateFile, "/")+1:]
	t, err := template.New(name).Funcs(funcMap).ParseFiles(e.TemplateFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse template")
	}

	var h bytes.Buffer
	if err := t.ExecuteTemplate(&h, name, data); err != nil {
		return "", errors.Wrap(err, "failed to process template")
	}
	return h.String(), nil
}

func (e *Exporter) timestampHuman(t time.Time) string {
	return t.In(e.location).Format("15:04:05")
}

func (e *Exporter) maybeDownloadFile(fileID string) error {
	if fileID == "" {
		return nil
	}

	if _, found := e.fileIDToURL[fileID]; found {
		// already downloaded
		return nil
	}

	fileExists, err := e.storage.FileExists(fileID)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file %s exists", fileID)
	}

	if fileExists {
		e.fileIDToURL[fileID] = e.storage.BuildLink(fileID)
		return nil
	}

	log.Printf("[DEBUG] downloading file %s", fileID)
	body, err := e.fileRecipient.GetFile(fileID)
	if err != nil {
		return errors.Wrapf(err, "failed to get file body for %s", fileID)
	}
	defer body.Close()

	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return errors.Wrapf(err, "failed to read file body for %s", fileID)
	}
	log.Printf("[DEBUG] downloaded file %s", fileID)

	fileURL, err := e.storage.CreateFile(fileID, bodyBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", fileID)
	}

	e.fileIDToURL[fileID] = fileURL
	// hacky: need to pass fileURL to template
	// using this map in template FuncMap later
	return nil
}

func readMessages(path string, broadcastUsers SuperUser) ([]bot.Message, error) {
	file, err := os.Open(path) // nolint
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("[WARN] can't close %s", file.Name())
		}
	}()

	messages := []bot.Message{}
	var (
		currentIndex           uint
		broadcastStartedIndex  uint
		broadcastFinishedIndex uint
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		msg := bot.Message{}
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("[ERROR] failed to unmarshal %s, error=%v", line, err)
			continue
		}

		if broadcastUsers != nil && broadcastUsers.IsSuper(msg.From.Username) {
			// if received message from bot/user who can send "broadcast" messages
			if strings.Contains(msg.Text, bot.MsgBroadcastStarted) {
				if broadcastStartedIndex == 0 {
					// record first occurrence of MsgBroadcastFinished
					broadcastStartedIndex = currentIndex
				}
				continue
			}

			if strings.Contains(msg.Text, bot.MsgBroadcastFinished) {
				// record last occurrence of MsgBroadcastFinished
				broadcastFinishedIndex = currentIndex
				continue
			}
		}

		if filter(msg) {
			continue
		}
		messages = append(messages, msg)
		currentIndex++
	}

	if broadcastStartedIndex == 0 {
		log.Print(`[WARN] "BroadcastStarted" message not found, exporting messages from the beginning`)
	}
	if broadcastFinishedIndex == 0 && len(messages) > 0 {
		log.Print(`[WARN] "BroadcastFinished" message not found, exporting messages till the end`)
		broadcastFinishedIndex = uint(len(messages))
	}

	messages = messages[broadcastStartedIndex:broadcastFinishedIndex]

	return messages, scanner.Err()
}

func filter(msg bot.Message) bool {
	contains := func(s []string, e string) bool {
		e = strings.TrimSpace(strings.ToLower(e))
		for _, a := range s {
			if strings.ToLower(a) == e {
				return true
			}
		}
		return false
	}
	return contains([]string{"+1", "-1", ":+1:", ":-1:"}, msg.Text)
}

func format(text string, entities *[]bot.Entity) (out template.HTML) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] failed to format %q, %#v", text, entities)
		}
	}()

	out = template.HTML(strings.ReplaceAll(html.EscapeString(text), "\n", "<br>")) // nolint
	if entities == nil {
		return out
	}

	runes := []rune(text)
	result := ""
	pos := 0

	for _, entity := range *entities {
		if entity.Offset < pos {
			// current code does not support nested entities
			continue
		}

		before, after := getDecoration(
			entity,
			runes[entity.Offset:entity.Offset+entity.Length],
		)

		if before == "" && after == "" {
			continue
		}

		result += html.EscapeString(string(runes[pos:entity.Offset])) +
			before +
			html.EscapeString(string(runes[entity.Offset:entity.Offset+entity.Length])) +
			after

		pos = entity.Offset + entity.Length
	}

	result += html.EscapeString(string(runes[pos:]))

	return template.HTML(strings.ReplaceAll(result, "\n", "<br>")) // nolint
}

// getDecoration returns pair of HTML tags (decorations) for Telegram Entity
func getDecoration(entity bot.Entity, body []rune) (string, string) {
	switch entity.Type {
	case "bold":
		return "<strong>", "</strong>"

	case "italic":
		return "<em>", "</em>"

	case "underline":
		return "<u>", "</u>"

	case "strikethrough":
		return "<s>", "</s>"

	case "code":
		return "<code>", "</code>"

	case "pre":
		return "<pre>", "</pre>"

	case "text_link":
		return fmt.Sprintf("<a href=\"%s\">", entity.URL), "</a>"

	case "url":
		urlRaw := string(body)

		// fix links without scheme so they will be non-relative in browser
		u, err := url.Parse(urlRaw)
		if err != nil {
			log.Printf("[ERROR] failed parse URL %s", urlRaw)
		} else {
			if u.Scheme == "" {
				u.Scheme = "https"
				urlRaw = u.String()
			}
		}

		return fmt.Sprintf("<a href=\"%s\">", urlRaw), "</a>"

	case "mention":
		return fmt.Sprintf("<a class=\"mention\" href=\"https://t.me/%s\">", string(body[1:])), "</a>"
		// body[1:] because first symbol in mention is "@" it's not needed for link

	case "email":
		return fmt.Sprintf("<a href=\"mailto:%s\">", string(body)), "</a>"

	case "phone_number":
		return fmt.Sprintf("<a href=\"tel:%s\">", cleanPhoneNumber(string(body))), "</a>"

	// intentionally ignored:
	case "text_mention": // for users without usernames
	case "bot_command": // "/start@jobs_bot"
	case "hashtag": // "#hashtag"
	case "cashtag": // "$USD"
	}

	return "", ""
}

func cleanPhoneNumber(phoneNumber string) string {
	reg := regexp.MustCompile(`[^\d+]`)
	return reg.ReplaceAllString(phoneNumber, "")
}
