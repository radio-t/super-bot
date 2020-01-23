package reporter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/radio-t/gitter-rt-bot/app/bot"
)

// magic number used to trim text in reply
// 43 is the number that Telegram clients are using
const maxQuoteLength = 43

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
	SuperUsers     SuperUser
	BroadcastUsers SuperUser // Users who can send "bot.MsgBroadcastStarted" and "bot.MsgBroadcastStarted" messages.
	// it maybe just bot, or bot + some or all SuperUsers.
	// Cannot use SuperUsers field for same purpose becase they used to mark messages as "from host" in template
}

// SuperUser knows which user is a superuser
type SuperUser interface {
	IsSuper(user string) bool
}

// Storage knows how to: create file, check for file existance
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
func (e *Exporter) Export(showNum int, yyyymmdd int) {
	from := fmt.Sprintf("%s/%s.log", e.InputRoot, time.Now().Format("20060102")) // current day by default
	if yyyymmdd != 0 {
		from = fmt.Sprintf("%s/%d.log", e.InputRoot, yyyymmdd)
	}
	to := fmt.Sprintf("%s/radio-t-%d.html", e.OutputRoot, showNum)

	messages, err := readMessages(from, e.ExporterParams.BroadcastUsers)
	if err != nil {
		log.Fatalf("[ERROR] failed to read messages from %s, %v", from, err)
	}
	fh, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("[ERROR] failed to export, %v", err)
	}
	defer fh.Close()

	fh.WriteString(e.toHTML(messages, showNum))
	log.Printf("[INFO] exported %d lines to %s", len(messages), to)

}

func (e *Exporter) toHTML(messages []bot.Message, num int) string {

	type Record struct {
		Time   string
		IsHost bool
		Msg    bot.Message
	}

	type Data struct {
		Num     int
		Records []Record
	}

	data := Data{Num: num}
	for _, msg := range messages {
		if msg.Image != nil {
			e.maybeDownloadFile(msg.Image.FileID)
		}

		data.Records = append(
			data.Records,
			Record{
				Time:   msg.Sent.In(e.location).Format("15:04:05"),
				IsHost: e.SuperUsers.IsSuper(msg.From.Username),
				Msg:    msg,
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
		log.Fatalf("[ERROR] failed to parse, %v", err)
	}

	var html bytes.Buffer
	if err := t.ExecuteTemplate(&html, name, data); err != nil {
		log.Fatalf("[ERROR] failed, error %v", err)
	}
	return html.String()
}

func (e *Exporter) timestampHuman(t time.Time) string {
	return t.In(e.location).Format("15:04:05")
}

func (e *Exporter) maybeDownloadFile(fileID string) {
	if fileID == "" {
		return
	}

	if _, found := e.fileIDToURL[fileID]; found {
		// already downloaded
		return
	}

	fileExists, err := e.storage.FileExists(fileID)
	if err != nil {
		log.Printf("[ERROR] failed to check if file exists alredy: %s: %v", fileID, err)
		return
	}

	if fileExists {
		e.fileIDToURL[fileID] = e.storage.BuildLink(fileID)
		return
	}

	log.Printf("[DEBUG] downloading file %s", fileID)
	body, err := e.fileRecipient.GetFile(fileID)
	if err != nil {
		log.Printf("[ERROR] failed to get file body for %s: %v", fileID, err)
		return
	}
	defer body.Close()

	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		log.Printf("[ERROR] failed to read file body for %s: %v", fileID, err)
		return
	}
	log.Printf("[DEBUG] downloaded file %s", fileID)

	fileURL, err := e.storage.CreateFile(fileID, bodyBytes)
	if err != nil {
		log.Printf("[ERROR] failed to create file %s: %v", fileID, err)
		return
	}

	e.fileIDToURL[fileID] = fileURL
	// hacky: need to pass fileURL to template
	// using this map in template FuncMap later

	return
}

func readMessages(path string, broadcastUsers SuperUser) ([]bot.Message, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

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
			switch msg.Text {
			case bot.MsgBroadcastStarted:
				if broadcastStartedIndex == 0 {
					// record first occurance of MsgBroadcastFinished
					// "+1" to exclude "MsgBroadcastStarted"
					broadcastStartedIndex = currentIndex
				}
				continue
			case bot.MsgBroadcastFinished:
				// record last occurance of MsgBroadcastFinished
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
		log.Println("[WARN] \"BroadcastStarted\" message was not found, exporting messages from the beginning")
	}
	if broadcastFinishedIndex == 0 && len(messages) > 0 {
		log.Println("[WARN] \"BroadcastFinished\" message was not found, exporting messages till the end")
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

func format(text string, entities *[]bot.Entity) template.HTML {
	if entities == nil {
		return template.HTML(html.EscapeString(text))
	}

	runes := []rune(text)
	result := ""
	pos := 0

	for _, entity := range *entities {
		before, after := getDecoration(
			entity,
			runes[entity.Offset:entity.Offset+entity.Length],
		)

		result += html.EscapeString(string(runes[pos:entity.Offset])) +
			before +
			html.EscapeString(string(runes[entity.Offset:entity.Offset+entity.Length])) +
			after

		pos = entity.Offset + entity.Length
	}

	result += html.EscapeString(string(runes[pos:]))

	return template.HTML(strings.ReplaceAll(result, "\n", "<br>"))
}

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
		return fmt.Sprintf("<a href=\"%s\">", string(body)), "</a>"

	case "mention":
		return fmt.Sprintf("<a href=\"https://t.me/%s\">", string(body[1:])), "</a>"
		// body[1:] because first symbol in mention is "@" it's not needed for link

	case "email":
		return fmt.Sprintf("<a href=\"mailto:%s\">", string(body)), "</a>"

	case "phone_number":
		return fmt.Sprintf("<a href=\"tel:%s\">", cleanPhoneNumber(string(body))), "</a>"

	// intentially ignored:
	case "text_mention": // for users without usernames
	case "bot_command": // "/start@jobs_bot"
	case "hashtag": // "#hashtag"
	case "cashtag": // "$USD"
	}

	return "", ""
}

func cleanPhoneNumber(phoneNumber string) string {
	reg := regexp.MustCompile("[^\\d+]")
	return reg.ReplaceAllString(phoneNumber, "")
}
