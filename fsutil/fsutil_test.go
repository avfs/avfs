package fsutil_test

import (
	"testing"

	"github.com/avfs/avfs/fs/memfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
)

// TestPath
func TestPath(t *testing.T) {
	fs, err := memfs.New(memfs.OptMainDirs(), memfs.OptIdm(memidm.New()))
	if err != nil {
		t.Fatalf("memfs.New : want error to be nil, got %v", err)
	}

	cf := test.NewConfigFs(t, fs)
	cf.SuitePath()
}