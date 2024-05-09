package failfs

import (
	"io/fs"
	"os"

	"github.com/avfs/avfs"
)

// OkFunc is a FailFunc that never fails.
func OkFunc(_ avfs.VFSBase, _ avfs.FnVFS, _ *FailParam) error {
	return nil
}

// ReadOnlyFunc is a FailFunc that fails on writes.
func ReadOnlyFunc(_ avfs.VFSBase, fn avfs.FnVFS, fp *FailParam) error {
	switch fn {
	case avfs.FnOpenFile:
		if fp.Flag != os.O_RDONLY {
			return &fs.PathError{Op: fp.Op, Path: fp.Path, Err: avfs.ErrPermDenied}
		}

		return nil
	case avfs.FnChmod, avfs.FnFileChmod, avfs.FnFileChown, avfs.FnChtimes, avfs.FnCreateTemp,
		avfs.FnFileSync, avfs.FnFileTruncate, avfs.FnFileWrite, avfs.FnFileWriteAt,
		avfs.FnMkdir, avfs.FnMkdirAll, avfs.FnMkdirTemp, avfs.FnRemove, avfs.FnRemoveAll, avfs.FnTruncate:
		return &fs.PathError{Op: fp.Op, Path: fp.Path, Err: avfs.ErrPermDenied}
	case avfs.FnChown, avfs.FnLchown:
		return &fs.PathError{Op: fp.Op, Path: fp.Path, Err: avfs.ErrOpNotPermitted}
	case avfs.FnLink, avfs.FnRename, avfs.FnSymlink:
		return &os.LinkError{Op: fp.Op, Old: fp.Path, New: fp.NewPath, Err: avfs.ErrPermDenied}
	default:
		return nil
	}
}
