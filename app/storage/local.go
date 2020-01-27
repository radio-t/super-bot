package storage

import (
	"io/ioutil"
	"os"
)

// Local implements Storage interface
// operating with the local filesystem (possibly mounted to the container)
type Local struct {
	filesPath  string
	publicPath string
}

// NewLocal creates new Local storage
func NewLocal(filesPath string, publicPath string) (*Local, error) {
	if _, err := os.Stat(filesPath); os.IsNotExist(err) {
		err = os.MkdirAll(filesPath, 0755) // nolint
		if err != nil {
			return nil, err
		}
	}

	return &Local{filesPath: filesPath, publicPath: publicPath}, nil
}

// FileExists checks if file exists in `filesPath` directory
func (l *Local) FileExists(fileName string) (bool, error) {
	_, err := os.Stat(l.filesPath + "/" + fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// CreateFile creates file in `filesPath` directory with a given name and body
func (l *Local) CreateFile(fileName string, body []byte) (string, error) {
	err := ioutil.WriteFile(l.filesPath+"/"+fileName, body, 0644)
	if err != nil {
		return "", err
	}

	return l.BuildLink(fileName), nil
}

// BuildLink builds public-accessible link to file
func (l *Local) BuildLink(fileName string) string {
	return l.publicPath + "/" + fileName
}

// BuildPath builds local path to file
func (l *Local) BuildPath(fileName string) string {
	return l.filesPath + "/" + fileName
}
