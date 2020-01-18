package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/radio-t/gitter-rt-bot/app/bot"
	"github.com/radio-t/gitter-rt-bot/app/storage"

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
	e, err := setup(nil, nil, nil)
	assert.NoError(t, err)
	defer teardown()

	exporter := NewExporter(nil, nil, nil, testExportParams)
	assert.Equal(t, exporter, e)
}

func TestExporter_Export(t *testing.T) {
	e, err := setup(nil, nil, nil)
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
			msgs, err := readMessages(testFile)
			if tt.fail {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.msgs, msgs)
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

	e, err := setup(fileRecipient, nil, storage)
	assert.NoError(t, err)
	defer teardown()

	err = createFile(e.InputRoot+"/20200111.log", msgs)
	assert.NoError(t, err)
	defer os.Remove(e.InputRoot + "/20200111.log")

	e.Export(684, 20200111)

	fileRecipient.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func Test_downloadFilesPhoto(t *testing.T) {
	msgs := []bot.Message{
		{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Picture: &bot.Picture{
				Image: bot.Image{
					FileID: "FILE_ID_1",
					Sources: []bot.Source{
						bot.Source{
							FileID: "FILE_ID_1",
						},
						bot.Source{
							FileID: "FILE_ID_2",
						},
					},
				},
				Thumbnail: &bot.Source{
					FileID: "FILE_ID_1",
				},
			},
		},
	}

	fileRecipient := new(fileRecipientMock)
	fileRecipient.On("GetFile", "FILE_ID_1").Return(buffer("IMAGE_1"), nil).Once()
	fileRecipient.On("GetFile", "FILE_ID_2").Return(buffer("IMAGE_2"), nil)

	storage := new(storageMock)
	storage.On("FileExists", "FILE_ID_1").Return(false, nil).Once()
	storage.On("FileExists", "FILE_ID_2").Return(false, nil)
	storage.On("CreateFile", "FILE_ID_1", []byte("IMAGE_1")).Return("684/FILE_ID_1", nil).Once()
	storage.On("CreateFile", "FILE_ID_2", []byte("IMAGE_2")).Return("684/FILE_ID_2", nil)

	e, err := setup(fileRecipient, nil, storage)
	assert.NoError(t, err)
	defer teardown()

	err = createFile(e.InputRoot+"/20200111.log", msgs)
	assert.NoError(t, err)
	defer os.Remove(e.InputRoot + "/20200111.log")

	e.Export(684, 20200111)

	fileRecipient.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func Test_downloadFilesSticker(t *testing.T) {
	msgs := []bot.Message{
		{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Picture: &bot.Picture{
				Class: "sticker",
				Image: bot.Image{
					FileID: "FILE_ID",
					Alt:    "😀",
					Type:   "webp",
				},
				Sources: []bot.Source{
					bot.Source{
						FileID: "FILE_ID",
						Type:   "webp",
					},
					bot.Source{
						FileID: "FILE_ID.png",
						Type:   "png",
					},
				},
				Thumbnail: &bot.Source{
					FileID: "FILE_ID_THUMB",
					Width:  128,
					Height: 128,
				},
			},
		},
	}

	fileRecipient := new(fileRecipientMock)
	fileRecipient.On("GetFile", "FILE_ID").Return(buffer("IMAGE"), nil).Once()
	fileRecipient.On("GetFile", "FILE_ID_THUMB").Return(buffer("THUMB"), nil).Once()

	converter := new(converterMock)
	converter.On("Convert", "FILE_ID").Return(nil)
	converter.On("Convert", "FILE_ID_THUMB").Return(nil)

	storage := new(storageMock)
	storage.On("FileExists", "FILE_ID").Return(false, nil).Once()
	storage.On("CreateFile", "FILE_ID", []byte("IMAGE")).Return("684/FILE_ID", nil).Once()
	storage.On("BuildLink", "FILE_ID.png").Return("684/FILE_ID.png", nil)

	storage.On("FileExists", "FILE_ID_THUMB").Return(false, nil)
	storage.On("CreateFile", "FILE_ID_THUMB", []byte("THUMB")).Return("684/FILE_ID_THUMB", nil).Once()

	e, err := setup(fileRecipient, map[string]Converter{"webp": converter}, storage)
	assert.NoError(t, err)
	defer teardown()

	err = createFile(e.InputRoot+"/20200111.log", msgs)
	assert.NoError(t, err)
	defer os.Remove(e.InputRoot + "/20200111.log")

	e.Export(684, 20200111)

	fileRecipient.AssertExpectations(t)
	converter.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func Test_downloadFilesAnimatedSticker(t *testing.T) {
	msgs := []bot.Message{
		{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Picture: &bot.Picture{
				Class: "animated-sticker",
				Image: bot.Image{
					FileID: "FILE_ID.json",
					Width:  512,
					Height: 512,
					Type:   "json",
					Alt:    "👻",
				},
				Thumbnail: &bot.Source{
					FileID: "FILE_ID_THUMB",
					Width:  128,
					Height: 128,
				},
				Sources: []bot.Source{
					bot.Source{
						FileID: "FILE_ID",
						Type:   "tgs",
						Size:   2278,
					},
					bot.Source{
						FileID: "FILE_ID.json",
						Type:   "json",
					},
				},
			},
		},
	}

	fileRecipient := new(fileRecipientMock)
	fileRecipient.On("GetFile", "FILE_ID").Return(buffer("STICKER"), nil).Once()
	fileRecipient.On("GetFile", "FILE_ID_THUMB").Return(buffer("STICKER_THUMB"), nil).Once()

	converterTGS := new(converterMock)
	converterTGS.On("Convert", "FILE_ID").Return(nil)

	converterWebP := new(converterMock)
	converterWebP.On("Convert", "FILE_ID_THUMB").Return(nil)

	storage := new(storageMock)
	storage.On("FileExists", "FILE_ID").Return(false, nil)
	storage.On("CreateFile", "FILE_ID", []byte("STICKER")).Return("684/FILE_ID", nil)
	storage.On("BuildLink", "FILE_ID.json").Return("684/FILE_ID.json", nil)

	storage.On("FileExists", "FILE_ID_THUMB").Return(false, nil)
	storage.On("CreateFile", "FILE_ID_THUMB", []byte("STICKER_THUMB")).Return("684/FILE_ID_THUMB", nil)

	e, err := setup(
		fileRecipient,
		map[string]Converter{
			"tgs":  converterTGS,
			"webp": converterWebP,
		},
		storage,
	)
	assert.NoError(t, err)
	defer teardown()

	err = createFile(e.InputRoot+"/20200111.log", msgs)
	assert.NoError(t, err)
	defer os.Remove(e.InputRoot + "/20200111.log")

	e.Export(684, 20200111)

	fileRecipient.AssertExpectations(t)
	converterTGS.AssertExpectations(t)
	converterWebP.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func Test_downloadFilesDocument(t *testing.T) {
	msgs := []bot.Message{
		{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Document: &bot.Document{
				FileID:   "FILE_ID",
				MimeType: "image/jpeg",
				Size:     101780,
				Thumbnail: &bot.Source{
					FileID: "FILE_ID_THUMB",
					Width:  300,
					Height: 300,
					Size:   49827,
				},
			},
		},
	}

	fileRecipient := new(fileRecipientMock)
	fileRecipient.On("GetFile", "FILE_ID").Return(buffer("DOCUMENT"), nil).Once()
	fileRecipient.On("GetFile", "FILE_ID_THUMB").Return(buffer("THUMB"), nil).Once()

	storage := new(storageMock)
	storage.On("FileExists", "FILE_ID").Return(false, nil)
	storage.On("CreateFile", "FILE_ID", []byte("DOCUMENT")).Return("684/FILE_ID", nil)

	storage.On("FileExists", "FILE_ID_THUMB").Return(false, nil)
	storage.On("CreateFile", "FILE_ID_THUMB", []byte("THUMB")).Return("684/FILE_ID_THUMB", nil)

	e, err := setup(fileRecipient, nil, storage)
	assert.NoError(t, err)
	defer teardown()

	err = createFile(e.InputRoot+"/20200111.log", msgs)
	assert.NoError(t, err)
	defer os.Remove(e.InputRoot + "/20200111.log")

	e.Export(684, 20200111)

	fileRecipient.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func Test_downloadFilesAnimation(t *testing.T) {
	msgs := []bot.Message{
		{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Animation: &bot.Animation{
				FileID:   "FILE_ID",
				FileName: "giphy.mp4",
				MimeType: "video/mp4",
				Size:     199710,
				Duration: 2,
				Width:    480,
				Height:   266,
				Thumbnail: &bot.Source{
					FileID: "FILE_ID_THUMB",
					Width:  90,
					Height: 50,
					Size:   2158,
				},
			},
		},
	}

	fileRecipient := new(fileRecipientMock)
	fileRecipient.On("GetFile", "FILE_ID").Return(buffer("ANIMATION"), nil).Once()
	fileRecipient.On("GetFile", "FILE_ID_THUMB").Return(buffer("THUMB"), nil).Once()

	storage := new(storageMock)
	storage.On("FileExists", "FILE_ID").Return(false, nil)
	storage.On("CreateFile", "FILE_ID", []byte("ANIMATION")).Return("684/FILE_ID", nil)

	storage.On("FileExists", "FILE_ID_THUMB").Return(false, nil)
	storage.On("CreateFile", "FILE_ID_THUMB", []byte("THUMB")).Return("684/FILE_ID_THUMB", nil)

	e, err := setup(fileRecipient, nil, storage)
	assert.NoError(t, err)
	defer teardown()

	err = createFile(e.InputRoot+"/20200111.log", msgs)
	assert.NoError(t, err)
	defer os.Remove(e.InputRoot + "/20200111.log")

	e.Export(684, 20200111)

	fileRecipient.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func Test_downloadFilesVoice(t *testing.T) {
	msgs := []bot.Message{
		{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Voice: &bot.Voice{
				Duration: 1,
				Sources: []bot.Source{
					{
						FileID: "FILE_ID",
						Type:   "audio/ogg",
						Size:   6394,
					},
					{
						FileID: "FILE_ID.mp3",
						Type:   "audio/mp3",
					},
				},
			},
		},
	}

	fileRecipient := new(fileRecipientMock)
	fileRecipient.On("GetFile", "FILE_ID").Return(buffer("VOICE"), nil)

	converter := new(converterMock)
	converter.On("Convert", "FILE_ID").Return(nil)

	storage := new(storageMock)
	storage.On("FileExists", "FILE_ID").Return(false, nil)
	storage.On("CreateFile", "FILE_ID", []byte("VOICE")).Return("684/FILE_ID", nil)
	storage.On("BuildLink", "FILE_ID.mp3").Return("684/FILE_ID.mp3", nil)

	e, err := setup(fileRecipient, map[string]Converter{"audio/ogg": converter}, storage)
	assert.NoError(t, err)
	defer teardown()

	err = createFile(e.InputRoot+"/20200111.log", msgs)
	assert.NoError(t, err)
	defer os.Remove(e.InputRoot + "/20200111.log")

	e.Export(684, 20200111)

	fileRecipient.AssertExpectations(t)
	converter.AssertExpectations(t)
	storage.AssertExpectations(t)
}

//setup creates Exporter with temp folders
func setup(fileRecipient FileRecipient, converters map[string]Converter, storage storage.Storage) (*Exporter, error) {
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
		converters:     converters,
		storage:        storage,
		fileIDToURL:    map[string]string{},
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

type converterMock struct {
	mock.Mock
}

func (cm *converterMock) Convert(fileID string) error {
	args := cm.Called(fileID)
	return args.Error(0)
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
