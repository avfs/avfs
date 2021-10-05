//
//  Copyright 2021 The AVFS authors
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

package avfs

import (
	"io/fs"
	"os"
	"time"
)

// DummyFS represents the file system.
type DummyFS struct {
	utils Utils
}

// DummyFile represents an open file descriptor.
type DummyFile struct{}

// DummySysStat implements SysStater interface returned by fs.FileInfo.Sys().
type DummySysStat struct{}

// DummyFS configuration functions.

// NewDummyFS creates a new NewDummyFS file system.
func NewDummyFS() *DummyFS {
	vfs := &DummyFS{utils: NewUtils(OsLinux)}

	return vfs
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *DummyFS) Features() Feature {
	return 0
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (vfs *DummyFS) HasFeature(feature Feature) bool {
	return false
}

// Name returns the name of the fileSystem.
func (vfs *DummyFS) Name() string {
	return ""
}

// OSType returns the operating system type of the file system.
func (vfs *DummyFS) OSType() OSType {
	return vfs.utils.osType
}

// Type returns the type of the fileSystem or Identity manager.
func (vfs *DummyFS) Type() string {
	return "DummyFS"
}

// DummyFS file system functions.

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (vfs *DummyFS) Abs(path string) (string, error) {
	return vfs.utils.Abs(vfs, path)
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *DummyFS) Base(path string) string {
	return vfs.utils.Base(path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Chdir(dir string) error {
	const op = "chdir"

	return &fs.PathError{Op: op, Path: dir, Err: ErrPermDenied}
}

// Chmod changes the mode of the named file to mode.
// If the file is a symbolic link, it changes the mode of the link's target.
// If there is an error, it will be of type *PathError.
//
// A different subset of the mode bits are used, depending on the
// operating system.
//
// On Unix, the mode's permission bits, ModeSetuid, ModeSetgid, and
// ModeSticky are used.
//
// On Windows, only the 0200 bit (owner writable) of mode is used; it
// controls whether the file's read-only attribute is set or cleared.
// The other bits are currently unused. For compatibility with Go 1.12
// and earlier, use a non-zero mode. Use mode 0400 for a read-only
// file and 0600 for a readable+writable file.
//
// On Plan 9, the mode's permission bits, ModeAppend, ModeExclusive,
// and ModeTemporary are used.
func (vfs *DummyFS) Chmod(name string, mode fs.FileMode) error {
	const op = "chmod"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *DummyFS) Chown(name string, uid, gid int) error {
	const op = "chown"

	return &fs.PathError{Op: op, Path: name, Err: ErrOpNotPermitted}
}

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Chroot(path string) error {
	const op = "chroot"

	return &fs.PathError{Op: op, Path: path, Err: ErrOpNotPermitted}
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Chtimes(name string, atime, mtime time.Time) error {
	const op = "chtimes"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// Clean returns the shortest path name equivalent to path
// by purely lexical processing. It applies the following rules
// iteratively until no further processing can be done:
//
//	1. Replace multiple Separator elements with a single one.
//	2. Eliminate each . path name element (the current directory).
//	3. Eliminate each inner .. path name element (the parent directory)
//	   along with the non-.. element that precedes it.
//	4. Eliminate .. elements that begin a rooted path:
//	   that is, replace "/.." by "/" at the beginning of a path,
//	   assuming Separator is '/'.
//
// The returned path ends in a slash only if it represents a root directory,
// such as "/" on Unix or `C:\` on Windows.
//
// Finally, any occurrences of slash are replaced by Separator.
//
// If the result of this process is an empty string, Clean
// returns the string ".".
//
// See also Rob Pike, ``Lexical File Names in Plan 9 or
// Getting Dot-Dot Right,''
// https://9p.io/sys/doc/lexnames.html
func (vfs *DummyFS) Clean(path string) string {
	return vfs.utils.Clean(path)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Create(name string) (File, error) {
	return vfs.utils.Create(vfs, name)
}

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (vfs *DummyFS) CreateTemp(dir, pattern string) (File, error) {
	return vfs.utils.CreateTemp(vfs, dir, pattern)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *DummyFS) Dir(path string) string {
	return vfs.utils.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *DummyFS) EvalSymlinks(path string) (string, error) {
	const op = "lstat"

	return "", &fs.PathError{Op: op, Path: path, Err: ErrPermDenied}
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *DummyFS) FromSlash(path string) string {
	return vfs.utils.FromSlash(path)
}

// Getwd returns a rooted path name corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *DummyFS) Getwd() (dir string, err error) {
	const op = "getwd"

	return "", &fs.PathError{Op: op, Path: dir, Err: ErrPermDenied}
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (vfs *DummyFS) Glob(pattern string) (matches []string, err error) {
	return vfs.utils.Glob(vfs, pattern)
}

// Idm returns the identity manager of the file system.
// if the file system does not have an identity manager, avfs.DummyIdm is returned.
func (vfs *DummyFS) Idm() IdentityMgr {
	return NotImplementedIdm
}

// IsAbs reports whether the path is absolute.
func (vfs *DummyFS) IsAbs(path string) bool {
	return vfs.utils.IsAbs(path)
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (vfs *DummyFS) IsExist(err error) bool {
	return vfs.utils.IsExist(err)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (vfs *DummyFS) IsNotExist(err error) bool {
	return vfs.utils.IsNotExist(err)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *DummyFS) IsPathSeparator(c uint8) bool {
	return vfs.utils.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (vfs *DummyFS) Join(elem ...string) string {
	return vfs.utils.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *DummyFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	return &fs.PathError{Op: op, Path: name, Err: ErrOpNotPermitted}
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *DummyFS) Link(oldname, newname string) error {
	const op = "link"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: ErrPermDenied}
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Lstat(name string) (fs.FileInfo, error) {
	const op = "lstat"

	return nil, &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// Match reports whether name matches the shell file name pattern.
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-Separator characters
//		'?'         matches any single non-Separator character
//		'[' [ '^' ] { character-range } ']'
//		            character class (must be non-empty)
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c
//		lo '-' hi   matches character c for lo <= c <= hi
//
// Match requires pattern to match all of name, not just a substring.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
//
// On Windows, escaping is disabled. Instead, '\\' is treated as
// path separator.
//
func (vfs *DummyFS) Match(pattern, name string) (matched bool, err error) {
	return vfs.utils.Match(pattern, name)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Mkdir(name string, perm fs.FileMode) error {
	const op = "mkdir"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// MkdirAll creates a directory named path,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *DummyFS) MkdirAll(path string, perm fs.FileMode) error {
	const op = "mkdir"

	return &fs.PathError{Op: op, Path: path, Err: ErrPermDenied}
}

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (vfs *DummyFS) MkdirTemp(dir, pattern string) (string, error) {
	return vfs.utils.MkdirTemp(vfs, dir, pattern)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Open(name string) (File, error) {
	return vfs.utils.Open(vfs, name)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned DummyFile can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	const op = "open"

	return &DummyFile{}, &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// PathSeparator return the OS-specific path separator.
func (vfs *DummyFS) PathSeparator() uint8 {
	return vfs.utils.pathSeparator
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *DummyFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return vfs.utils.ReadDir(vfs, name)
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
func (vfs *DummyFS) ReadFile(name string) ([]byte, error) {
	return vfs.utils.ReadFile(vfs, name)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Readlink(name string) (string, error) {
	const op = "readlink"

	return "", &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (vfs *DummyFS) Rel(basepath, targpath string) (string, error) {
	return vfs.utils.Rel(basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Remove(name string) error {
	const op = "remove"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) RemoveAll(path string) error {
	const op = "removeall"

	return &fs.PathError{Op: op, Path: path, Err: ErrPermDenied}
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *DummyFS) Rename(oldname, newname string) error {
	const op = "rename"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: ErrPermDenied}
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (vfs *DummyFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	return false
}

// SetUMask sets the file mode creation mask.
func (vfs *DummyFS) SetUMask(mask fs.FileMode) {
}

// SetUser sets and returns the current user.
// If the user is not found, the returned error is of type UnknownUserError.
func (vfs *DummyFS) SetUser(name string) (UserReader, error) {
	return nil, ErrPermDenied
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *DummyFS) Split(path string) (dir, file string) {
	return vfs.utils.Split(path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Stat(name string) (fs.FileInfo, error) {
	const op = "stat"

	return nil, &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *DummyFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: ErrPermDenied}
}

// TempDir returns the default directory to use for temporary files.
//
// On Unix systems, it returns $TMPDIR if non-empty, else /tmp.
// On Windows, it uses GetTempPath, returning the first non-empty
// value from %TMP%, %TEMP%, %USERPROFILE%, or the Windows directory.
// On Plan 9, it returns /tmp.
//
// The directory is neither guaranteed to exist nor have accessible
// permissions.
func (vfs *DummyFS) TempDir() string {
	return vfs.utils.TempDir(vfs)
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *DummyFS) ToSlash(path string) string {
	return vfs.utils.ToSlash(path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *DummyFS) ToSysStat(info fs.FileInfo) SysStater {
	return &DummySysStat{}
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *DummyFS) Truncate(name string, size int64) error {
	const op = "truncate"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// UMask returns the file mode creation mask.
func (vfs *DummyFS) UMask() fs.FileMode {
	return 0o022
}

// User returns the current user.
// if the file system does not have a current user, the user avfs.NotImplementedUser is returned.
func (vfs *DummyFS) User() UserReader {
	return NotImplementedUser
}

// Utils returns the file utils of the current file system.
func (vfs *DummyFS) Utils() Utils {
	return vfs.utils
}

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root.
//
// All errors that arise visiting files and directories are filtered by fn:
// see the fs.WalkDirFunc documentation for details.
//
// The files are walked in lexical order, which makes the output deterministic
// but requires WalkDir to read an entire directory into memory before proceeding
// to walk that directory.
//
// WalkDir does not follow symbolic links.
func (vfs *DummyFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return vfs.utils.WalkDir(vfs, root, fn)
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
func (vfs *DummyFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return vfs.utils.WriteFile(vfs, name, data, perm)
}

// DummyFile functions.

// Chdir changes the current working directory to the file,
// which must be a directory.
// If there is an error, it will be of type *PathError.
func (f *DummyFile) Chdir() error {
	const op = "chdir"

	return &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// Chmod changes the mode of the file to mode.
// If there is an error, it will be of type *PathError.
func (f *DummyFile) Chmod(mode fs.FileMode) error {
	const op = "chmod"

	return &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// Chown changes the numeric uid and gid of the named file.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (f *DummyFile) Chown(uid, gid int) error {
	const op = "chown"

	return &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// Close closes the DummyFile, rendering it unusable for I/O.
// On files that support SetDeadline, any pending I/O operations will
// be canceled and return immediately with an error.
func (f *DummyFile) Close() error {
	const op = "close"

	return &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// Fd returns the integer Unix file descriptor referencing the open file.
// The file descriptor is valid only until f.Close is called or f is garbage collected.
// On Unix systems this will cause the SetDeadline methods to stop working.
func (f *DummyFile) Fd() uintptr {
	return 0
}

// Name returns the name of the file as presented to Open.
func (f *DummyFile) Name() string {
	return NotImplemented
}

// Read reads up to len(b) bytes from the DummyFile.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *DummyFile) Read(b []byte) (n int, err error) {
	const op = "read"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// ReadAt reads len(b) bytes from the DummyFile starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (f *DummyFile) ReadAt(b []byte, off int64) (n int, err error) {
	const op = "read"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
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

	return nil, &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
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

	return nil, &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// Seek sets the offset for the next Read or Write on file to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means
// relative to the current offset, and 2 means relative to the end.
// It returns the new offset and an error, if any.
// The behavior of Seek on a file opened with O_APPEND is not specified.
func (f *DummyFile) Seek(offset int64, whence int) (ret int64, err error) {
	const op = "seek"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *DummyFile) Stat() (fs.FileInfo, error) {
	const op = "stat"

	return nil, &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy
// of recently written data to disk.
func (f *DummyFile) Sync() error {
	const op = "sync"

	return &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// Truncate changes the size of the file.
// It does not change the I/O offset.
// If there is an error, it will be of type *PathError.
func (f *DummyFile) Truncate(size int64) error {
	const op = "truncate"

	return &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// Write writes len(b) bytes to the DummyFile.
// It returns the number of bytes written and an error, if any.
// Write returns a non-nil error when n != len(b).
func (f *DummyFile) Write(b []byte) (n int, err error) {
	const op = "write"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// WriteAt writes len(b) bytes to the DummyFile starting at byte offset off.
// It returns the number of bytes written and an error, if any.
// WriteAt returns a non-nil error when n != len(b).
func (f *DummyFile) WriteAt(b []byte, off int64) (n int, err error) {
	const op = "write"

	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: ErrPermDenied}
}

// WriteString is like Write, but writes the contents of string s rather than
// a slice of bytes.
func (f *DummyFile) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}

// Gid returns the group id.
func (sst *DummySysStat) Gid() int {
	return NotImplementedUser.Gid()
}

// Uid returns the user id.
func (sst *DummySysStat) Uid() int {
	return NotImplementedUser.Uid()
}

// Nlink returns the number of hard links.
func (sst *DummySysStat) Nlink() uint64 {
	return 1
}
