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

package orefafs

import (
	"io"
	"io/fs"
	"math"
	"time"

	"github.com/avfs/avfs"
)

// Chdir changes the current working directory to the file,
// which must be a directory.
// If there is an error, it will be of type *PathError.
func (f *OrefaFile) Chdir() error {
	const op = "chdir"

	if f == nil {
		return fs.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return fs.ErrInvalid
	}

	if f.nd == nil {
		return &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	if !f.nd.mode.IsDir() {
		return &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrNotADirectory}
	}

	f.vfs.curDir = f.name

	return nil
}

// Chmod changes the mode of the file to mode.
// If there is an error, it will be of type *PathError.
func (f *OrefaFile) Chmod(mode fs.FileMode) error {
	const op = "chmod"

	if f == nil {
		return fs.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.name == "" {
		return fs.ErrInvalid
	}

	if f.nd == nil {
		return &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	f.nd.setMode(mode)

	return nil
}

// Chown changes the numeric uid and gid of the named file.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (f *OrefaFile) Chown(uid, gid int) error {
	const op = "chown"

	if f == nil {
		return fs.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.name == "" {
		return fs.ErrInvalid
	}

	if f.nd == nil {
		return &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	f.nd.setOwner(uid, gid)

	return nil
}

// Close closes the File, rendering it unusable for I/O.
// On files that support SetDeadline, any pending I/O operations will
// be canceled and return immediately with an error.
func (f *OrefaFile) Close() error {
	const op = "close"

	if f == nil {
		return fs.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.nd == nil {
		if f.name == "" {
			return fs.ErrInvalid
		}

		return &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	f.dirEntries = nil
	f.dirNames = nil
	f.nd = nil

	return nil
}

// Fd returns the integer Unix file descriptor referencing the open file.
// The file descriptor is valid only until f.Close is called or f is garbage collected.
// On Unix systems this will cause the SetDeadline methods to stop working.
func (f *OrefaFile) Fd() uintptr {
	return uintptr(math.MaxUint64)
}

// Name returns the link of the file as presented to Open.
func (f *OrefaFile) Name() string {
	return f.name
}

// Read reads up to len(b) bytes from the OrefaFile.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *OrefaFile) Read(b []byte) (n int, err error) {
	const op = "read"

	if f == nil {
		return 0, fs.ErrInvalid
	}

	f.mu.RLock()

	if f.name == "" {
		f.mu.RUnlock()

		return 0, fs.ErrInvalid
	}

	if f.nd == nil {
		f.mu.RUnlock()

		return 0, &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() {
		f.mu.RUnlock()

		return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrIsADirectory}
	}

	if f.permMode&avfs.PermRead == 0 {
		f.mu.RUnlock()

		return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrBadFileDesc}
	}

	f.mu.RUnlock()

	nd.mu.RLock()
	n = copy(b, nd.data[f.at:])
	nd.mu.RUnlock()

	f.mu.Lock()
	f.at += int64(n)
	f.mu.Unlock()

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

// ReadAt reads len(b) bytes from the File starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (f *OrefaFile) ReadAt(b []byte, off int64) (n int, err error) {
	const op = "read"

	if f == nil {
		return 0, fs.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return 0, fs.ErrInvalid
	}

	if f.nd == nil {
		return 0, &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() {
		return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrIsADirectory}
	}

	if off < 0 {
		return 0, &fs.PathError{Op: "readat", Path: f.name, Err: avfs.ErrNegativeOffset}
	}

	if f.permMode&avfs.PermRead == 0 {
		return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrBadFileDesc}
	}

	nd.mu.RLock()
	defer nd.mu.RUnlock()

	if int(off) > len(nd.data) {
		return 0, io.EOF
	}

	n = copy(b, nd.data[off:])
	if n < len(b) {
		return n, io.EOF
	}

	return n, nil
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
func (f *OrefaFile) ReadDir(n int) ([]fs.DirEntry, error) {
	const op = "readdirent"

	if f == nil {
		return nil, fs.ErrInvalid
	}

	f.mu.RLock()

	if f.name == "" {
		f.mu.RUnlock()

		return nil, fs.ErrInvalid
	}

	if f.nd == nil {
		f.mu.RUnlock()

		return nil, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrFileClosing}
	}

	nd := f.nd
	if !nd.mode.IsDir() {
		f.mu.RUnlock()

		return nil, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrNotADirectory}
	}

	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	if n <= 0 || f.dirEntries == nil {
		nd.mu.RLock()
		de := nd.dirEntries()
		nd.mu.RUnlock()

		f.dirIndex = 0

		if n <= 0 {
			f.dirEntries = nil

			return de, nil
		}

		f.dirEntries = de
	}

	start := f.dirIndex
	if start >= len(f.dirEntries) {
		f.dirIndex = 0
		f.dirEntries = nil

		return nil, io.EOF
	}

	end := start + n
	if end > len(f.dirEntries) {
		end = len(f.dirEntries)
	}

	f.dirIndex = end

	return f.dirEntries[start:end], nil
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
func (f *OrefaFile) Readdirnames(n int) (names []string, err error) {
	const op = "readdirent"

	if f == nil {
		return nil, fs.ErrInvalid
	}

	f.mu.RLock()

	if f.name == "" {
		f.mu.RUnlock()

		return nil, fs.ErrInvalid
	}

	if f.nd == nil {
		f.mu.RUnlock()

		return nil, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrFileClosing}
	}

	nd := f.nd
	if !nd.mode.IsDir() {
		f.mu.RUnlock()

		return nil, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrNotADirectory}
	}

	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	if n <= 0 || f.dirNames == nil {
		nd.mu.RLock()
		names := nd.dirNames()
		nd.mu.RUnlock()

		f.dirIndex = 0

		if n <= 0 {
			f.dirNames = nil

			return names, nil
		}

		f.dirNames = names
	}

	start := f.dirIndex
	if start >= len(f.dirNames) {
		f.dirIndex = 0
		f.dirNames = nil

		return nil, io.EOF
	}

	end := start + n
	if end > len(f.dirNames) {
		end = len(f.dirNames)
	}

	f.dirIndex = end

	return f.dirNames[start:end], nil
}

// Seek sets the offset for the next Read or Write on file to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means
// relative to the current offset, and 2 means relative to the end.
// It returns the new offset and an error, if any.
// The behavior of Seek on a file opened with O_APPEND is not specified.
func (f *OrefaFile) Seek(offset int64, whence int) (ret int64, err error) {
	const op = "seek"

	if f == nil {
		return 0, fs.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.name == "" {
		return 0, fs.ErrInvalid
	}

	if f.nd == nil {
		return 0, &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() {
		return 0, nil
	}

	nd.mu.RLock()
	size := int64(len(nd.data))
	nd.mu.RUnlock()

	switch whence {
	case io.SeekStart:
		if offset < 0 {
			return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrInvalidArgument}
		}

		f.at = offset
	case io.SeekCurrent:
		if f.at+offset < 0 {
			return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrInvalidArgument}
		}

		f.at += offset
	case io.SeekEnd:
		if size+offset < 0 {
			return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrInvalidArgument}
		}

		f.at = size + offset
	default:
		return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrInvalidArgument}
	}

	return f.at, nil
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *OrefaFile) Stat() (fs.FileInfo, error) {
	const op = "stat"

	if f == nil {
		return nil, fs.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return &OrefaInfo{}, fs.ErrInvalid
	}

	if f.nd == nil {
		return &OrefaInfo{}, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrFileClosing}
	}

	_, name := f.vfs.split(f.name)
	fst := f.nd.fillStatFrom(name)

	return fst, nil
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy
// of recently written data to disk.
func (f *OrefaFile) Sync() error {
	const op = "sync"

	if f == nil {
		return fs.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return fs.ErrInvalid
	}

	if f.nd == nil {
		return &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	return nil
}

// Truncate changes the size of the file.
// It does not change the I/O offset.
// If there is an error, it will be of type *PathError.
func (f *OrefaFile) Truncate(size int64) error {
	const op = "truncate"

	if f == nil {
		return fs.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return fs.ErrInvalid
	}

	if f.nd == nil {
		return &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() || f.permMode&avfs.PermWrite == 0 {
		return &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrInvalidArgument}
	}

	if size < 0 {
		return &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrInvalidArgument}
	}

	nd.mu.Lock()

	nd.truncate(size)
	nd.mtime = time.Now().UnixNano()

	nd.mu.Unlock()

	return nil
}

// Write writes len(b) bytes to the File.
// It returns the number of bytes written and an error, if any.
// Write returns a non-nil error when n != len(b).
func (f *OrefaFile) Write(b []byte) (n int, err error) {
	const op = "write"

	if f == nil {
		return 0, fs.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return 0, fs.ErrInvalid
	}

	if f.nd == nil {
		return 0, &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() || f.permMode&avfs.PermWrite == 0 {
		return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrBadFileDesc}
	}

	nd.mu.Lock()

	n = copy(nd.data[f.at:], b)
	if n < len(b) {
		nd.data = append(nd.data, b[n:]...)
		n = len(b)
	}

	nd.mtime = time.Now().UnixNano()

	nd.mu.Unlock()

	f.at += int64(n)

	return n, nil
}

// WriteAt writes len(b) bytes to the File starting at byte offset off.
// It returns the number of bytes written and an error, if any.
// WriteAt returns a non-nil error when n != len(b).
func (f *OrefaFile) WriteAt(b []byte, off int64) (n int, err error) {
	const op = "write"

	if f == nil {
		return 0, fs.ErrInvalid
	}

	if off < 0 {
		return 0, &fs.PathError{Op: "writeat", Path: f.name, Err: avfs.ErrNegativeOffset}
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return 0, fs.ErrInvalid
	}

	if f.nd == nil {
		return 0, &fs.PathError{Op: op, Path: f.name, Err: fs.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() || f.permMode&avfs.PermWrite == 0 {
		return 0, &fs.PathError{Op: op, Path: f.name, Err: avfs.ErrBadFileDesc}
	}

	nd.mu.Lock()

	diff := off + int64(len(b)) - nd.size()
	if diff > 0 {
		nd.data = append(nd.data, make([]byte, diff)...)
	}

	n = copy(nd.data[off:], b)

	nd.mtime = time.Now().UnixNano()

	nd.mu.Unlock()

	return n, nil
}

// WriteString is like Write, but writes the contents of string s rather than
// a slice of bytes.
func (f *OrefaFile) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}

// OrefaInfo is the implementation fs.DirEntry (returned by ReadDir) and fs.FileInfo (returned by Stat and Lstat).

// Info returns the FileInfo for the file or subdirectory described by the entry.
// The returned FileInfo may be from the time of the original directory read
// or from the time of the call to Info. If the file has been removed or renamed
// since the directory read, Info may return an error satisfying errors.Is(err, ErrNotExist).
// If the entry denotes a symbolic link, Info reports the information about the link itself,
// not the link's target.
func (info *OrefaInfo) Info() (fs.FileInfo, error) {
	return info, nil
}

// IsDir is the abbreviation for Mode().IsDir().
func (info *OrefaInfo) IsDir() bool {
	return info.mode.IsDir()
}

// Mode returns the file mode bits.
func (info *OrefaInfo) Mode() fs.FileMode {
	return info.mode
}

// ModTime returns the modification time.
func (info *OrefaInfo) ModTime() time.Time {
	return time.Unix(0, info.mtime)
}

// Name returns the base name of the file.
func (info *OrefaInfo) Name() string {
	return info.name
}

// Size returns the length in bytes for regular files; system-dependent for others.
func (info *OrefaInfo) Size() int64 {
	return info.size
}

// Sys returns the underlying data source (can return nil).
func (info *OrefaInfo) Sys() interface{} {
	return info
}

// Type returns the type bits for the entry.
// The type bits are a subset of the usual FileMode bits, those returned by the FileMode.Type method.
func (info *OrefaInfo) Type() fs.FileMode {
	return info.mode & fs.ModeType
}

// Gid returns the group id.
func (info *OrefaInfo) Gid() int {
	return info.gid
}

// Uid returns the user id.
func (info *OrefaInfo) Uid() int {
	return info.uid
}

// Nlink returns the number of hard links.
func (info *OrefaInfo) Nlink() uint64 {
	return uint64(info.nlink)
}
