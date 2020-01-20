package reporter

import (
	"compress/gzip"
	"io"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

// Converter knows how to convert file from one format to another
type Converter interface {
	Convert(fileID string) (err error)
}

// WebPConverter can convert WebP (used for Telegram stickers) into JPG
// See https://developers.google.com/speed/webp
type WebPConverter struct {
	storage Storage
}

// NewWebPConverter creates new WebPConverter
func NewWebPConverter(storage Storage) Converter {
	return &WebPConverter{storage: storage}
}

// Convert converts WebP image into JPG image
// Requires `dwebp` binary in PATH
// See http://downloads.webmproject.org/releases/webp/
func (w *WebPConverter) Convert(fileID string) error {
	cmd := exec.Command(
		"dwebp",
		w.storage.BuildPath(fileID),
		"-o",
		w.storage.BuildPath(fileID+".png"),
	)
	err := cmd.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start dwebp execution")
	}

	err = cmd.Wait()
	if err != nil {
		return errors.Wrap(err, "failed to execute dwebp")
	}

	return nil
}

// TGSConverter can convert TGS (used for Telegram Animated Stickers) into GIF
// TGS file is a GZIP archive of Lottie JSON file.
// See https://github.com/airbnb/lottie-web
// https://core.telegram.org/animated_stickers
type TGSConverter struct {
	storage Storage
}

// NewTGSConverter creates new TGSConverter
func NewTGSConverter(storage Storage) Converter {
	return &TGSConverter{storage: storage}
}

// Convert converts TGS file bytes into animated GIF
func (tgs *TGSConverter) Convert(fileID string) error {
	in, err := os.Open(tgs.storage.BuildPath(fileID))
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", tgs.storage.BuildPath(fileID))
	}
	defer in.Close()

	out, err := os.Create(tgs.storage.BuildPath(fileID + ".json"))
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", tgs.storage.BuildPath(fileID))
	}
	defer out.Close()

	reader, err := gzip.NewReader(in)
	if err != nil {
		return errors.Wrapf(err, "failed to create reader for file %s", tgs.storage.BuildPath(fileID))
	}
	defer reader.Close()

	_, err = io.Copy(out, reader)
	if err != nil {
		return errors.Wrapf(err, "failed to copy reader for file %s", tgs.storage.BuildPath(fileID))
	}

	return nil
}

// OGGConverter can convert OGG Audio (used for Voice messages) into MP3
// Requires `dwebp` binary in PATH
type OGGConverter struct {
	storage Storage
}

// NewOGGConverter creates new OGGConverter
func NewOGGConverter(storage Storage) Converter {
	return &OGGConverter{storage: storage}
}

// Convert converts OGG audio bytes to MP3
func (ogg *OGGConverter) Convert(fileID string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-i",
		ogg.storage.BuildPath(fileID),
		ogg.storage.BuildPath(fileID+".mp3"),
	)
	err := cmd.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start ffmpeg execution")
	}

	err = cmd.Wait()
	if err != nil {
		return errors.Wrap(err, "failed to execute ffmpeg")
	}

	return nil
}
