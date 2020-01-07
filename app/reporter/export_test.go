package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/radio-t/gitter-rt-bot/app/bot"
	"github.com/stretchr/testify/assert"
)

var testFile = "test.log"
var msgs = []bot.Message{{Text: "1st"}, {Text: "2nd"}}
var testExportParams = ExporterParams{
	OutputRoot:   "output",
	InputRoot:    "input",
	TemplateFile: "output/template",
	SuperUsers:   SuperUserMock{},
}

func TestNewExporter(t *testing.T) {
	e, err := setup()
	assert.NoError(t, err)
	defer teardown()

	exporter := NewExporter(nil, nil, testExportParams)
	assert.Equal(t, exporter, e)
}

func TestExporter_Export(t *testing.T) {
	e, err := setup()
	assert.NoError(t, err)
	defer teardown()

	tbl := []struct {
		showNum  int
		yyyymmdd int
		output   string
	}{
		{1, 0, "output/radio-t-1.html"},
		{2, 20081012, "output/radio-t-2.html"},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			from := fmt.Sprintf("%s/%s.log", e.InputRoot, time.Now().Format("20060102"))
			if tt.yyyymmdd != 0 {
				from = fmt.Sprintf("%s/%d.log", e.InputRoot, tt.yyyymmdd)
			}
			err := createFile(from, msgs)
			assert.NoError(t, err)
			defer os.Remove(from)
			e.Export(tt.showNum, tt.yyyymmdd)
			assert.FileExists(t, tt.output)
		})
	}
}

func Test_filter(t *testing.T) {
	tbl := []struct {
		input  tbapi.Message
		output bool
	}{
		{tbapi.Message{Text: " +1"}, true},
		{tbapi.Message{Text: " -1"}, true},
		{tbapi.Message{Text: ":+1:"}, true},
		{tbapi.Message{Text: ":-1:"}, true},
		{tbapi.Message{Text: "+1 blah"}, false},
		{tbapi.Message{Text: "blah +1 blah"}, false},
	}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			filtered := filter(tt.input)
			assert.Equal(t, tt.output, filtered)
		})
	}
}

func Test_readMessages(t *testing.T) {
	tbl := []struct {
		createFile bool
		msgs       []bot.Message
		fail       bool
	}{
		{false, nil, true},
		{true, []bot.Message{}, false},
		{true, msgs, false},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tt.createFile {
				err := createFile(testFile, tt.msgs)
				defer os.Remove(testFile)
				assert.NoError(t, err)
			}
			msgs, err := readMessages(testFile)
			if tt.fail {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.msgs, msgs)
		})
	}
}

//setup creates Exporter with temp folders
func setup() (*Exporter, error) {
	err := os.MkdirAll(testExportParams.InputRoot, os.ModePerm)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(testExportParams.OutputRoot, os.ModePerm)
	if err != nil {
		return nil, err
	}

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return nil, err
	}

	f, err := os.Create(testExportParams.TemplateFile)
	if err != nil {
		return nil, err
	}
	f.Close()

	e := &Exporter{
		ExporterParams: testExportParams,
		location:       location,
	}

	return e, nil
}

// teardown remove all temp folders
func teardown() {
	_ = os.RemoveAll(testExportParams.InputRoot)
	_ = os.RemoveAll(testExportParams.OutputRoot)
}

// setup creates file and fill it with  messages
func createFile(filepath string, msgs []bot.Message) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, msg := range msgs {
		bdata, err := json.Marshal(&msg)
		if err != nil {
			return err
		}
		_, err = f.WriteString(string(bdata) + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

type SuperUserMock map[string]bool

// IsSuper checks if user in su list
func (s SuperUserMock) IsSuper(userName string) bool {
	return s[userName]
}
