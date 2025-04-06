package storage

import (
	"fmt"
	"os"
)

// Local implements Storage interface
// operating with the local filesystem (possibly mounted to the container)
type Local struct {
	filesPath  string
	publicPath string
}

// NewLocal creates new Local storage
func NewLocal(filesPath, publicPath string) (*Local, error) {
	if _, err := os.Stat(filesPath); os.IsNotExist(err) {
		err = os.MkdirAll(filesPath, 0755) // nolint
		if err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", filesPath, err)
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

		return false, fmt.Errorf("failed to check if file %s exists: %w", fileName, err)
	}

	return true, nil
}

// CreateFile creates file in `filesPath` directory with a given name and body
func (l *Local) CreateFile(fileName string, body []byte) (string, error) {
	err := os.WriteFile(l.filesPath+"/"+fileName, body, 0644) // nolint
	if err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", fileName, err)
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
