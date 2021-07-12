//
//  Copyright 2020 The AVFS authors
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

package dummyfs

import (
	"io/fs"

	"github.com/avfs/avfs"
)

// Chdir changes the current working directory to the file,
// which must be a directory.
// If there is an error, it will be of type *PathError.
func (f *DummyFile) Chdir() error {
	const op = "chdir"

	return &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Chmod changes the mode of the file to mode.
// If there is an error, it will be of type *PathError.
func (f *DummyFile) Chmod(mode fs.FileMode) error {
	const op = "chmod"

	return &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Chown changes the numeric uid and gid of the named file.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (f *DummyFile) Chown(uid, gid int) error {
	const op = "chown"

	return &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Close closes the DummyFile, rendering it unusable for I/O.
// On files that support SetDeadline, any pending I/O operations will
// be canceled and return immediately with an error.
func (f *DummyFile) Close() error {
	const op = "close"

	return &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Fd returns the integer Unix file descriptor referencing the open file.
// The file descriptor is valid only until f.Close is called or f is garbage collected.
// On Unix systems this will cause the SetDeadline methods to stop working.
func (f *DummyFile) Fd() uintptr {
	return 0
}

// Name returns the name of the file as presented to Open.
func (f *DummyFile) Name() string {
	return avfs.NotImplemented
}

// Read reads up to len(b) bytes from the DummyFile.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *DummyFile) Read(b []byte) (n int, err error) {
	const op = "read"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// ReadAt reads len(b) bytes from the DummyFile starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (f *DummyFile) ReadAt(b []byte, off int64) (n int, err error) {
	const op = "read"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
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
func (f *DummyFile) ReadDir(n int) ([]fs.DirEntry, error) {
	const op = "readdirent"

	return nil, &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Readdir reads the contents of the directory associated with file and
// returns a slice of up to n FileInfo values, as would be returned
// by Lstat, in directory order. Subsequent calls on the same file will yield
// further FileInfos.
//
// If n > 0, Readdir returns at most n FileInfo structures. In this case, if
// Readdir returns an empty slice, it will return a non-nil error
// explaining why. At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdir returns all the FileInfo from the directory in
// a single slice. In this case, if Readdir succeeds (reads all
// the way to the end of the directory), it returns the slice and a
// nil error. If it encounters an error before the end of the
// directory, Readdir returns the FileInfo read until that point
// and a non-nil error.
//
// Most clients are better served by the more efficient ReadDir method.
func (f *DummyFile) Readdir(n int) ([]fs.FileInfo, error) {
	const op = "readdirent"

	return nil, &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
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
func (f *DummyFile) Readdirnames(n int) (names []string, err error) {
	const op = "readdirent"

	return nil, &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Seek sets the offset for the next Read or Write on file to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means
// relative to the current offset, and 2 means relative to the end.
// It returns the new offset and an error, if any.
// The behavior of Seek on a file opened with O_APPEND is not specified.
func (f *DummyFile) Seek(offset int64, whence int) (ret int64, err error) {
	const op = "seek"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *DummyFile) Stat() (fs.FileInfo, error) {
	const op = "stat"

	return nil, &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy
// of recently written data to disk.
func (f *DummyFile) Sync() error {
	const op = "sync"

	return &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Truncate changes the size of the file.
// It does not change the I/O offset.
// If there is an error, it will be of type *PathError.
func (f *DummyFile) Truncate(size int64) error {
	const op = "truncate"

	return &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// Write writes len(b) bytes to the DummyFile.
// It returns the number of bytes written and an error, if any.
// Write returns a non-nil error when n != len(b).
func (f *DummyFile) Write(b []byte) (n int, err error) {
	const op = "write"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// WriteAt writes len(b) bytes to the DummyFile starting at byte offset off.
// It returns the number of bytes written and an error, if any.
// WriteAt returns a non-nil error when n != len(b).
func (f *DummyFile) WriteAt(b []byte, off int64) (n int, err error) {
	const op = "write"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: avfs.ErrPermDenied}
}

// WriteString is like Write, but writes the contents of string s rather than
// a slice of bytes.
func (f *DummyFile) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}
