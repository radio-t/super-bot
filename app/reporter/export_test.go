package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/radio-t/super-bot/app/bot"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	e, err := setup(nil, nil)
	assert.NoError(t, err)
	defer teardown()

	exporter := NewExporter(nil, nil, testExportParams)
	assert.Equal(t, exporter, e)
}

func TestExporter_Export(t *testing.T) {
	e, err := setup(nil, nil)
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
		input  bot.Message
		output bool
	}{
		{bot.Message{Text: " +1"}, true},
		{bot.Message{Text: " -1"}, true},
		{bot.Message{Text: ":+1:"}, true},
		{bot.Message{Text: ":-1:"}, true},
		{bot.Message{Text: "+1 blah"}, false},
		{bot.Message{Text: "blah +1 blah"}, false},
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
			msgs, err := readMessages(testFile, nil)
			if tt.fail {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.msgs, msgs)
		})
	}
}

func Test_readMessagesCheckBroadcastMessages(t *testing.T) {
	tbl := []struct {
		broadcastUsers SuperUserMock
		in             []bot.Message
		out            []bot.Message
	}{
		{
			SuperUserMock{},
			[]bot.Message{},
			[]bot.Message{},
		},
		{
			SuperUserMock{"radio-t-bot": true},
			[]bot.Message{
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "radio-t-bot"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
			},
			[]bot.Message{},
		},
		{
			SuperUserMock{"radio-t-bot": true},
			[]bot.Message{
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
			},
			[]bot.Message{
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
			},
		},
		{
			SuperUserMock{"radio-t-bot": true},
			[]bot.Message{
				{Text: bot.MsgBroadcastStarted + "\n_pong_", From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
			},
			[]bot.Message{
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
			},
		},
		{
			SuperUserMock{"radio-t-bot": true},
			[]bot.Message{
				{Text: "message-0", From: bot.User{Username: "user-0"}},
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-3", From: bot.User{Username: "user-3"}},
			},
			[]bot.Message{
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
			},
		},
		{
			SuperUserMock{"radio-t-bot": true},
			[]bot.Message{
				{Text: "message-0", From: bot.User{Username: "user-0"}},
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "radio-t-bot"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "radio-t-bot"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-3", From: bot.User{Username: "user-3"}},
			},
			[]bot.Message{
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
			},
		},
		{
			SuperUserMock{},
			[]bot.Message{
				{Text: "message-0", From: bot.User{Username: "user-0"}},
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-3", From: bot.User{Username: "user-3"}},
			},
			[]bot.Message{
				{Text: "message-0", From: bot.User{Username: "user-0"}},
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-3", From: bot.User{Username: "user-3"}},
			},
		},
		{
			SuperUserMock{"radio-t-bot": true, "umputun": true},
			[]bot.Message{
				{Text: "message-0", From: bot.User{Username: "user-0"}},
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "umputun"}},
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-3", From: bot.User{Username: "user-3"}},
			},
			[]bot.Message{
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
			},
		},
		{
			SuperUserMock{"radio-t-bot": true},
			[]bot.Message{
				{Text: bot.MsgBroadcastStarted, From: bot.User{Username: "radio-t-bot"}},
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
			},
			[]bot.Message{
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
			},
		},
		{
			SuperUserMock{"radio-t-bot": true},
			[]bot.Message{
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
				{Text: bot.MsgBroadcastFinished, From: bot.User{Username: "radio-t-bot"}},
			},
			[]bot.Message{
				{Text: "message-1", From: bot.User{Username: "user-1"}},
				{Text: "message-2", From: bot.User{Username: "user-2"}},
			},
		},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			err := createFile(testFile, tt.in)
			defer os.Remove(testFile)
			assert.NoError(t, err)

			msgs, err := readMessages(testFile, tt.broadcastUsers)
			assert.NoError(t, err)
			assert.Equal(t, tt.out, msgs)
		})
	}
}

func Test_downloadFilesNeverCalledForTextMessages(t *testing.T) {
	msgs := []bot.Message{
		{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Text: "Message",
		},
	}

	fileRecipient := new(fileRecipientMock)
	storage := new(storageMock)

	e, err := setup(fileRecipient, storage)
	assert.NoError(t, err)
	defer teardown()

	err = createFile(e.InputRoot+"/20200111.log", msgs)
	assert.NoError(t, err)
	defer os.Remove(e.InputRoot + "/20200111.log")

	e.Export(684, 20200111)

	fileRecipient.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func Test_downloadFilesImage(t *testing.T) {
	msgs := []bot.Message{
		{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Image: &bot.Image{
				FileID: "FILE_ID",
			},
		},
	}

	fileRecipient := new(fileRecipientMock)
	fileRecipient.On("GetFile", "FILE_ID").Return(buffer("IMAGE"), nil).Once()

	storage := new(storageMock)
	storage.On("FileExists", "FILE_ID").Return(false, nil).Once()
	storage.On("CreateFile", "FILE_ID", []byte("IMAGE")).Return("684/FILE_ID", nil).Once()

	e, err := setup(fileRecipient, storage)
	assert.NoError(t, err)
	defer teardown()

	err = createFile(e.InputRoot+"/20200111.log", msgs)
	assert.NoError(t, err)
	defer os.Remove(e.InputRoot + "/20200111.log")

	e.Export(684, 20200111)

	fileRecipient.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestExporter_format(t *testing.T) {
	tbl := []struct {
		in       string
		entities *[]bot.Entity
		output   template.HTML
	}{
		{
			"text", nil, "text",
		},
		{
			"text",
			&[]bot.Entity{
				{
					Type:   "bold",
					Offset: 0,
					Length: 4,
				},
			},
			"<strong>text</strong>",
		},
		{
			"- У нас дыра в безопасности\n- Ну хоть что-то у нас в безопасности",
			nil,
			"- У нас дыра в безопасности<br>- Ну хоть что-то у нас в безопасности",
		},
		{
			"some text here",
			&[]bot.Entity{{Type: "italic", Offset: 5, Length: 4}},
			"some <em>text</em> here",
		},
		{
			"@chuhlomin тебя слишком много, отдохни...",
			&[]bot.Entity{{Type: "mention", Offset: 0, Length: 10}, {Type: "italic", Offset: 11, Length: 30}},
			"<a class=\"mention\" href=\"https://t.me/chuhlomin\">@chuhlomin</a> <em>тебя слишком много, отдохни...</em>",
		},
		{
			"Меня url заинтересовал... do.co",
			&[]bot.Entity{{Type: "url", Offset: 26, Length: 5}},
			"Меня url заинтересовал... <a href=\"https://do.co\">do.co</a>",
		},
		{
			"inline",
			&[]bot.Entity{{Type: "code", Offset: 0, Length: 6}},
			"<code>inline</code>",
		},
		{
			"code block here",
			&[]bot.Entity{{Type: "pre", Offset: 0, Length: 15}},
			"<pre>code block here</pre>",
		},
		{
			"some link: https://github.com",
			&[]bot.Entity{{Type: "url", Offset: 11, Length: 18}},
			"some link: <a href=\"https://github.com\">https://github.com</a>",
		},
		{
			"email: mail@domain.com\nphone: +1 (987) 654-32-10",
			&[]bot.Entity{{Type: "email", Offset: 7, Length: 15}, {Type: "phone_number", Offset: 30, Length: 18}},
			"email: <a href=\"mailto:mail@domain.com\">mail@domain.com</a><br>phone: <a href=\"tel:+19876543210\">+1 (987) 654-32-10</a>",
		},
		{
			"some text here",
			&[]bot.Entity{{Type: "underline", Offset: 5, Length: 4}},
			"some <u>text</u> here",
		},
		{
			"some text here",
			&[]bot.Entity{{Type: "strikethrough", Offset: 5, Length: 4}},
			"some <s>text</s> here",
		},
		{
			"here is some link",
			&[]bot.Entity{{Type: "text_link", Offset: 13, Length: 4, URL: "https://github.com/"}},
			"here is some <a href=\"https://github.com/\">link</a>",
		},
		{
			"code\nwith\nline breaks",
			&[]bot.Entity{{Type: "pre", Offset: 0, Length: 21}},
			"<pre>code<br>with<br>line breaks</pre>",
		},
		{
			"Hello \u003cscript type='application/javascript'\u003ealert('xss');\u003c/script\u003e World",
			&[]bot.Entity{{Type: "bold", Offset: 0, Length: 5}, {Type: "italic", Offset: 67, Length: 5}},
			"<strong>Hello</strong> &lt;script type=&#39;application/javascript&#39;&gt;alert(&#39;xss&#39;);&lt;/script&gt; <em>World</em>",
		},
		{
			"say! /say /who когда? /когда /how /доколе правила? ping пинг кто? who? /кто when? доколе? /ping /пинг rules? /правила /when как? how? /как /rules news! новости! /news /новости анекдот! анкедот! joke! chuck! /анекдот /joke /chuck so! /so ddg! ?? /ddg search! /search",
			&[]bot.Entity{{Offset: 0, Length: 265, Type: "italic"}, {Offset: 5, Length: 4, Type: "bot_command"}, {Offset: 15, Length: 5, Type: "bot_command"}, {Offset: 21, Length: 4, Type: "bot_command"}, {Offset: 42, Length: 6, Type: "bot_command"}, {Offset: 89, Length: 4, Type: "bot_command"}, {Offset: 119, Length: 5, Type: "bot_command"}, {Offset: 161, Length: 5, Type: "bot_command"}, {Offset: 216, Length: 5, Type: "bot_command"}, {Offset: 222, Length: 6, Type: "bot_command"}, {Offset: 233, Length: 3, Type: "bot_command"}, {Offset: 245, Length: 4, Type: "bot_command"}, {Offset: 258, Length: 7, Type: "bot_command"}},
			"<em>say! /say /who когда? /когда /how /доколе правила? ping пинг кто? who? /кто when? доколе? /ping /пинг rules? /правила /when как? how? /как /rules news! новости! /news /новости анекдот! анкедот! joke! chuck! /анекдот /joke /chuck so! /so ddg! ?? /ddg search! /search</em>",
		},
		{
			"must show say.data",
			&[]bot.Entity{{Type: "bold", Offset: 0, Length: 18}, {Type: "url", Offset: 10, Length: 8}},
			"<strong>must show say.data</strong>",
		},
		{
			"must show say.data",
			&[]bot.Entity{{Type: "bold", Offset: 200, Length: 18}}, // to cause panic
			"must show say.data",
		},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.output, format(tt.in, tt.entities))
		})
	}
}

// setup creates Exporter with temp folders
func setup(fileRecipient FileRecipient, storage Storage) (*Exporter, error) {
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
		fileRecipient:  fileRecipient,
		storage:        storage,
		fileIDToURL:    map[string]string{},
	}

	return e, nil
}

// teardown removes all temp folders
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

type fileRecipientMock struct {
	mock.Mock
}

func (fdm *fileRecipientMock) GetFile(fileID string) (io.ReadCloser, error) {
	args := fdm.Called(fileID)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

type storageMock struct {
	mock.Mock
}

func (sm *storageMock) FileExists(fileName string) (bool, error) {
	args := sm.Called(fileName)
	return args.Bool(0), args.Error(1)
}

func (sm *storageMock) CreateFile(fileName string, body []byte) (string, error) {
	args := sm.Called(fileName, body)
	return args.String(0), args.Error(1)
}

func (sm *storageMock) BuildLink(fileName string) string {
	args := sm.Called(fileName)
	return args.String(0)
}

func (sm *storageMock) BuildPath(fileName string) string {
	args := sm.Called(fileName)
	return args.String(0)
}

// closingBuffer used in mocks to represent resp.Body, implements io.ReadCloser
type closingBuffer struct {
	*bytes.Buffer
}

func (cb *closingBuffer) Close() error {
	return nil
}

// buffer increases readability of tests
func buffer(content string) io.ReadCloser {
	return &closingBuffer{bytes.NewBufferString(content)}
}
