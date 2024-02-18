//
//  Copyright 2022 The AVFS authors
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

package mountfs

import (
	"io/fs"
)

// Chdir changes the current working directory to the file,
// which must be a directory.
// If there is an error, it will be of type *PathError.
func (f *MountFile) Chdir() error {
	err := f.file.Chdir()
	if err != nil {
		return f.mount.restoreError(err)
	}

	f.vfs.curMnt = f.mount
	_ = f.vfs.SetCurDir(f.file.Name())

	return nil
}

// Chmod changes the mode of the file to mode.
// If there is an error, it will be of type *PathError.
func (f *MountFile) Chmod(mode fs.FileMode) error {
	err := f.file.Chmod(mode)

	return f.mount.restoreError(err)
}

// Chown changes the numeric uid and gid of the named file.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (f *MountFile) Chown(uid, gid int) error {
	err := f.file.Chown(uid, gid)

	return f.mount.restoreError(err)
}

// Close closes the File, rendering it unusable for I/O.
// On files that support SetDeadline, any pending I/O operations will
// be canceled and return immediately with an error.
func (f *MountFile) Close() error {
	err := f.file.Close()

	return f.mount.restoreError(err)
}

// Fd returns the integer Unix file descriptor referencing the open file.
// The file descriptor is valid only until f.Close is called or f is garbage collected.
// On Unix systems this will cause the SetDeadline methods to stop working.
func (f *MountFile) Fd() uintptr {
	return f.file.Fd()
}

// Name returns the link of the file as presented to Open.
func (f *MountFile) Name() string {
	return f.mount.toAbsPath(f.file.Name())
}

// Read reads up to len(b) bytes from the MemFile.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *MountFile) Read(b []byte) (n int, err error) {
	n, err = f.file.Read(b)

	return n, f.mount.restoreError(err)
}

// ReadAt reads len(b) bytes from the File starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (f *MountFile) ReadAt(b []byte, off int64) (n int, err error) {
	n, err = f.file.ReadAt(b, off)

	return n, f.mount.restoreError(err)
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
func (f *MountFile) ReadDir(n int) ([]fs.DirEntry, error) {
	de, err := f.file.ReadDir(n)

	return de, f.mount.restoreError(err)
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
func (f *MountFile) Readdirnames(n int) (names []string, err error) {
	names, err = f.file.Readdirnames(n)

	return names, f.mount.restoreError(err)
}

// Seek sets the offset for the next Read or Write on file to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means
// relative to the current offset, and 2 means relative to the end.
// It returns the new offset and an error, if any.
// The behavior of Seek on a file opened with O_APPEND is not specified.
func (f *MountFile) Seek(offset int64, whence int) (ret int64, err error) {
	ret, err = f.file.Seek(offset, whence)

	return ret, f.mount.restoreError(err)
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *MountFile) Stat() (fs.FileInfo, error) {
	info, err := f.file.Stat()

	return info, f.mount.restoreError(err)
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy
// of recently written data to disk.
func (f *MountFile) Sync() error {
	err := f.file.Sync()

	return f.mount.restoreError(err)
}

// Truncate changes the size of the file.
// It does not change the I/O offset.
// If there is an error, it will be of type *PathError.
func (f *MountFile) Truncate(size int64) error {
	err := f.file.Truncate(size)

	return f.mount.restoreError(err)
}

// Write writes len(b) bytes to the File.
// It returns the number of bytes written and an error, if any.
// Write returns a non-nil error when n != len(b).
func (f *MountFile) Write(b []byte) (n int, err error) {
	n, err = f.file.Write(b)

	return n, f.mount.restoreError(err)
}

// WriteAt writes len(b) bytes to the File starting at byte offset off.
// It returns the number of bytes written and an error, if any.
// WriteAt returns a non-nil error when n != len(b).
func (f *MountFile) WriteAt(b []byte, off int64) (n int, err error) {
	n, err = f.file.WriteAt(b, off)

	return n, f.mount.restoreError(err)
}

// WriteString is like Write, but writes the contents of string s rather than
// a slice of bytes.
func (f *MountFile) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}
