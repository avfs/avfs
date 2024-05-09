//
//  Copyright 2024 The AVFS authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//  	http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package failfs

import (
	"io/fs"

	"github.com/avfs/avfs"
)

// Chdir changes the current working directory to the file,
// which must be a directory.
// If there is an error, it will be of type *PathError.
func (f *FailFile) Chdir() error {
	if f == nil {
		return fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "chdir", Path: name}
	vfs := f.vfs

	err := vfs.fail(avfs.FnFileChdir, &fp)
	if err != nil {
		return err
	}

	return f.baseFile.Chdir()
}

// Chmod changes the mode of the file to mode.
// If there is an error, it will be of type *PathError.
func (f *FailFile) Chmod(mode fs.FileMode) error {
	if f == nil {
		return fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "chmod", Path: name, Perm: mode}
	vfs := f.vfs

	err := vfs.fail(avfs.FnFileChmod, &fp)
	if err != nil {
		return err
	}

	return f.baseFile.Chmod(mode)
}

// Chown changes the numeric uid and gid of the named file.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (f *FailFile) Chown(uid, gid int) error {
	if f == nil {
		return fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "chown", Path: name, Uid: uid, Gid: gid}
	vfs := f.vfs

	err := vfs.fail(avfs.FnFileChown, &fp)
	if err != nil {
		return err
	}

	return f.baseFile.Chown(uid, gid)
}

// Close closes the File, rendering it unusable for I/O.
// On files that support SetDeadline, any pending I/O operations will
// be canceled and return immediately with an error.
func (f *FailFile) Close() error {
	if f == nil {
		return fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Path: name}
	vfs := f.vfs

	err := vfs.fail(avfs.FnFileClose, &fp)
	if err != nil {
		return err
	}

	return f.baseFile.Close()
}

// Fd returns the integer Unix file descriptor referencing the open file.
// The file descriptor is valid only until f.Close is called or f is garbage collected.
// On Unix systems this will cause the SetDeadline methods to stop working.
func (f *FailFile) Fd() uintptr {
	return f.baseFile.Fd()
}

// Name returns the link of the file as presented to Open.
func (f *FailFile) Name() string {
	if f.baseFile == nil {
		return ""
	}

	return f.baseFile.Name()
}

// Read reads up to len(b) bytes from the FailFile.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *FailFile) Read(b []byte) (n int, err error) {
	if f == nil {
		return 0, fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "read", Path: name}
	vfs := f.vfs

	err = vfs.fail(avfs.FnFileRead, &fp)
	if err != nil {
		return 0, err
	}

	return f.baseFile.Read(b)
}

// ReadAt reads len(b) bytes from the File starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (f *FailFile) ReadAt(b []byte, off int64) (n int, err error) {
	if f == nil {
		return 0, fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "read", Path: name}
	vfs := f.vfs

	err = vfs.fail(avfs.FnFileReadAt, &fp)
	if err != nil {
		return 0, err
	}

	return f.baseFile.ReadAt(b, off)
}

// ReadDir reads the contents of the directory associated with the file f
// and returns a slice of DirEntry values in directory order.
// Subsequent calls on the same file will yield later DirEntry records in the directory.
//
// If n > 0, ReadDir returns at most n DirEntry records.
// In this case, if ReadDir returns an empty slice, it will return an error explaining why.
// At the end of a directory, the error is io.EOF.
//
// If n <= 0, ReadDir returns all the DirEntry records remaining in the directory.
// When it succeeds, it returns a nil error (not io.EOF).
func (f *FailFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f == nil {
		return nil, fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "readdir", Path: name}
	vfs := f.vfs

	err := vfs.fail(avfs.FnFileReadDir, &fp)
	if err != nil {
		return nil, err
	}

	return f.baseFile.ReadDir(n)
}

// Readdirnames reads and returns a slice of names from the directory f.
//
// If n > 0, Readdirnames returns at most n names. In this case, if
// Readdirnames returns an empty slice, it will return a non-nil error
// explaining why. At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdirnames returns all the names from the directory in
// a single slice. In this case, if Readdirnames succeeds (reads all
// the way to the end of the directory), it returns the slice and a
// nil error. If it encounters an error before the end of the
// directory, Readdirnames returns the names read until that point and
// a non-nil error.
func (f *FailFile) Readdirnames(n int) (names []string, err error) {
	if f == nil {
		return nil, fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Path: name}
	vfs := f.vfs

	err = vfs.fail(avfs.FnFileReaddirnames, &fp)
	if err != nil {
		return nil, err
	}

	return f.baseFile.Readdirnames(n)
}

// Seek sets the offset for the next Read or Write on file to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means
// relative to the current offset, and 2 means relative to the end.
// It returns the new offset and an error, if any.
// The behavior of Seek on a file opened with O_APPEND is not specified.
func (f *FailFile) Seek(offset int64, whence int) (ret int64, err error) {
	if f == nil {
		return 0, fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "seek", Path: name, Flag: whence}
	vfs := f.vfs

	err = vfs.fail(avfs.FnFileSeek, &fp)
	if err != nil {
		return 0, err
	}

	return f.baseFile.Seek(offset, whence)
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *FailFile) Stat() (info fs.FileInfo, err error) {
	if f == nil {
		return nil, fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "stat", Path: name}
	vfs := f.vfs

	err = vfs.fail(avfs.FnFileStat, &fp)
	if err != nil {
		return nil, err
	}

	return f.baseFile.Stat()
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy
// of recently written data to disk.
func (f *FailFile) Sync() error {
	if f == nil {
		return fs.ErrInvalid
	}

	fp := FailParam{Op: "sync", Path: avfs.NotImplemented}
	vfs := f.vfs

	err := vfs.fail(avfs.FnFileSync, &fp)
	if err != nil {
		return err
	}

	return f.baseFile.Sync()
}

// Truncate changes the size of the file.
// It does not change the I/O offset.
// If there is an error, it will be of type *PathError.
func (f *FailFile) Truncate(size int64) error {
	if f == nil {
		return fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "truncate", Path: name, Size: size}
	vfs := f.vfs

	err := vfs.fail(avfs.FnFileTruncate, &fp)
	if err != nil {
		return err
	}

	return f.baseFile.Truncate(size)
}

// Write writes len(b) bytes to the File.
// It returns the number of bytes written and an error, if any.
// Write returns a non-nil error when n != len(b).
func (f *FailFile) Write(b []byte) (n int, err error) {
	if f == nil {
		return 0, fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "write", Path: name}
	vfs := f.vfs

	err = vfs.fail(avfs.FnFileWrite, &fp)
	if err != nil {
		return 0, err
	}

	return f.baseFile.Write(b)
}

// WriteAt writes len(b) bytes to the File starting at byte offset off.
// It returns the number of bytes written and an error, if any.
// WriteAt returns a non-nil error when n != len(b).
func (f *FailFile) WriteAt(b []byte, off int64) (n int, err error) {
	if f == nil {
		return 0, fs.ErrInvalid
	}

	name := f.Name()
	fp := FailParam{Op: "write", Path: name}
	vfs := f.vfs

	err = vfs.fail(avfs.FnFileWriteAt, &fp)
	if err != nil {
		return 0, err
	}

	return f.baseFile.WriteAt(b, off)
}

// WriteString is like Write, but writes the contents of string s rather than
// a slice of bytes.
func (f *FailFile) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}
