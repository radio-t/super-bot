package reporter

import (
	"bytes"
	"io/ioutil"
	"os/exec"

	"github.com/pkg/errors"
)

// Converter knows how to convert file from one format to another
type Converter interface {
	Convert(in []byte) (out []byte, err error)
	Extension() string
}

// WebPConverter can convert WebP (used for Telegram stickers) into JPG
// See https://developers.google.com/speed/webp
type WebPConverter struct{}

// NewWebPConverter creates new WebPConverter
func NewWebPConverter() Converter {
	return &WebPConverter{}
}

// Convert converts WebP image into JPG image
// Requires dwebp binary in PATH
// See http://downloads.webmproject.org/releases/webp/
func (w *WebPConverter) Convert(in []byte) (out []byte, err error) {
	var b bytes.Buffer

	cmd := exec.Command("dwebp", "-o", "-", "--", "-")
	cmd.Stdin = bytes.NewBuffer(in)
	cmd.Stdout = &b
	err = cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start dwebp execution")
	}

	err = cmd.Wait()
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute dwebp")
	}

	out, err = ioutil.ReadAll(&b)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read JSON (WebP to JSON conversion)")
	}

	return
}

// Extension returnes new file extension for converted file
func (w *WebPConverter) Extension() string {
	return "jpg"
}

// TODO: converter for TGS animated stickers?
