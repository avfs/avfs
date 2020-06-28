package fsutil_test

import (
	"testing"

	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/test"
)

func TestPath(t *testing.T) {
	fs, err := memfs.New(memfs.OptMainDirs())
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	cf := test.NewConfigFs(t, fs)
	cf.SuitePath()
}
