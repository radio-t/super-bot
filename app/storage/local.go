package storage

import (
	"io/ioutil"
	"os"
)

type Local struct {
	filesPath  string
	publicPath string
}

func NewLocal(filesPath string, publicPath string) (*Local, error) {
	if _, err := os.Stat(filesPath); os.IsNotExist(err) {
		err = os.MkdirAll(filesPath, 0755)
		if err != nil {
			return nil, err
		}
	}

	return &Local{
		filesPath:  filesPath,
		publicPath: publicPath,
	}, nil
}

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

func (l *Local) SaveFile(fileName string, body []byte) (string, error) {
	err := ioutil.WriteFile(l.filesPath+"/"+fileName, body, 0644)
	if err != nil {
		return "", err
	}

	return l.BuildLink(fileName), nil
}

func (l *Local) BuildLink(fileName string) string {
	return l.publicPath + "/" + fileName
}
