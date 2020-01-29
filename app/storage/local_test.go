package storage

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestNewLocalCreatesPathIfNotExists(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	_, err = NewLocal(path.Join(tmp, "no_dir"), "")
	require.NoError(t, err)
}

func TestNewLocal(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	_, err = NewLocal(tmp, "")
	require.NoError(t, err)
}

func TestFileExists(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	s, err := NewLocal(tmp, "")
	require.NoError(t, err)
	_, err = s.CreateFile("fn", []byte{})
	require.NoError(t, err)

	exists, err := s.FileExists("fn")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = s.FileExists("no_file")
	require.NoError(t, err)
	require.False(t, exists)
}
