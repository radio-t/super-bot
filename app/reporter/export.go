package reporter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/radio-t/gitter-rt-bot/app/bot"
	"github.com/radio-t/gitter-rt-bot/app/storage"
)

// Exporter performs conversion from log file to html
type Exporter struct {
	ExporterParams
	location      *time.Location
	fileRecipient FileRecipient
	converters    map[string]Converter
	storage       storage.Storage

	fileIDToURL map[string]string
}

// ExporterParams for locations
type ExporterParams struct {
	OutputRoot   string
	InputRoot    string
	TemplateFile string
	SuperUsers   SuperUser
}

// SuperUser knows which user is a superuser
type SuperUser interface {
	IsSuper(user string) bool
}

// NewExporter from params, initializes time.Location
func NewExporter(
	fileRecipient FileRecipient,
	converters map[string]Converter,
	storage storage.Storage,
	params ExporterParams,
) *Exporter {
	log.Printf("[INFO] exporter with %v", params)
	result := Exporter{
		ExporterParams: params,
		fileRecipient:  fileRecipient,
		converters:     converters,
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
func (e Exporter) Export(showNum int, yyyymmdd int) {
	from := fmt.Sprintf("%s/%s.log", e.InputRoot, time.Now().Format("20060102")) // current day by default
	if yyyymmdd != 0 {
		from = fmt.Sprintf("%s/%d.log", e.InputRoot, yyyymmdd)
	}
	to := fmt.Sprintf("%s/radio-t-%d.html", e.OutputRoot, showNum)

	messages, err := readMessages(from)
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

func (e Exporter) toHTML(messages []bot.Message, num int) string {

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
		e.maybeDownloadFiles(msg)

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
		"counter": func() func() int {
			i := -1
			return func() int {
				i++
				return i
			}
		},
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

func (e Exporter) maybeDownloadFiles(msg bot.Message) {
	switch {
	case msg.Picture != nil:
		err := e.maybeDownloadFile(msg.Picture.Image.Source)
		if err != nil {
			log.Printf("[ERROR] failed to get file URL for file %s: %v", msg.Picture.Image.Source.FileID, err)
		}

		for _, source := range (*msg.Picture).Image.Sources {
			err := e.maybeDownloadFile(source)
			if err != nil {
				log.Printf("[ERROR] failed to download file %s: %v", source.FileID, err)
				continue
			}
		}

		for _, source := range (*msg.Picture).Sources {
			err := e.maybeDownloadFile(source)
			if err != nil {
				log.Printf("[ERROR] failed to download file %s: %v", source.FileID, err)
				continue
			}
		}
	}
}

func (e Exporter) maybeDownloadFile(source bot.Source) error {
	if _, found := e.fileIDToURL[source.FileID]; found {
		// already downloaded
		return nil
	}

	if strings.Contains(source.FileID, ".") {
		// hacky way to handle WebP & TGS stickers
		// convertion to happens after image "FileID" download
		e.fileIDToURL[source.FileID] = e.storage.BuildLink(source.FileID)
		return nil
	}

	fileExists, err := e.storage.FileExists(source.FileID)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists alredy: %s", source.FileID)
	}

	if fileExists {
		e.fileIDToURL[source.FileID] = e.storage.BuildLink(source.FileID)
		return nil
	}

	body, err := e.fileRecipient.GetFile(source.FileID)
	if err != nil {
		return errors.Wrapf(err, "failed to get file body for %s", source.FileID)
	}
	defer body.Close()

	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return errors.Wrapf(err, "failed to read file body for %s", source.FileID)
	}

	fileURL, err := e.storage.CreateFile(source.FileID, bodyBytes)
	if err != nil {
		return err
	}

	e.fileIDToURL[source.FileID] = fileURL
	// hacky: need to pass fileURL to template
	// using this map in template FuncMap later

	converter, found := e.converters[source.Type]
	if !found {
		log.Printf("[DEBUG] no convertion will happen (converter for type \"%s\" not found)", source.Type)
		return nil
	}

	convertedBody, err := converter.Convert(bodyBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to convert file %s", source.FileID)
	}

	_, err = e.storage.CreateFile(source.FileID+"."+converter.Extension(), convertedBody)
	return err
}

func readMessages(path string) ([]bot.Message, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	messages := []bot.Message{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		msg := bot.Message{}
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("[ERROR] failed to unmarshal %s, error=%v", line, err)
			continue
		}
		if !filter(msg) {
			messages = append(messages, msg)
		}
	}
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
