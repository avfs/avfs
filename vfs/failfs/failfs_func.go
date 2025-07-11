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
func ReadOnlyFunc(vfs avfs.VFSBase, fn avfs.FnVFS, fp *FailParam) error {
	errForOS := avfs.ErrorsFor(vfs.OSType())

	switch fn {
	case avfs.FnOpenFile:
		if fp.Flag != avfs.O_DIRECTORY && fp.Flag != os.O_RDONLY {
			return &fs.PathError{Op: fp.Op, Path: fp.Path, Err: errForOS.PermDenied}
		}

		return nil
	case avfs.FnChmod, avfs.FnFileChmod, avfs.FnFileChown, avfs.FnChtimes, avfs.FnCreateTemp,
		avfs.FnFileSync, avfs.FnFileTruncate, avfs.FnFileWrite, avfs.FnFileWriteAt,
		avfs.FnMkdir, avfs.FnMkdirAll, avfs.FnMkdirTemp, avfs.FnRemove, avfs.FnRemoveAll, avfs.FnTruncate:
		return &fs.PathError{Op: fp.Op, Path: fp.Path, Err: errForOS.PermDenied}
	case avfs.FnChown, avfs.FnLchown:
		return &fs.PathError{Op: fp.Op, Path: fp.Path, Err: errForOS.OpNotPermitted}
	case avfs.FnLink, avfs.FnRename, avfs.FnSymlink:
		return &os.LinkError{Op: fp.Op, Old: fp.Path, New: fp.NewPath, Err: errForOS.PermDenied}
	default:
		return nil
	}
}
