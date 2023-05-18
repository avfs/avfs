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

// Package rofs provides a read only file system on top of any other Avfs file system.
package rofs

import (
	"io/fs"
	"os"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/dummyidm"
)

// file system functions.

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (vfs *RoFS) Abs(path string) (string, error) {
	return vfs.baseFS.Abs(path)
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *RoFS) Base(path string) string {
	return vfs.baseFS.Base(path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Chdir(dir string) error {
	return vfs.baseFS.Chdir(dir)
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
func (vfs *RoFS) Chmod(name string, mode fs.FileMode) error {
	const op = "chmod"

	return &fs.PathError{Op: op, Path: name, Err: vfs.errPermDenied}
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *RoFS) Chown(name string, uid, gid int) error {
	const op = "chown"

	return &fs.PathError{Op: op, Path: name, Err: vfs.errOpNotPermitted}
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Chtimes(name string, atime, mtime time.Time) error {
	const op = "chtimes"

	return &fs.PathError{Op: op, Path: name, Err: vfs.errPermDenied}
}

// Clean returns the shortest path name equivalent to path
// by purely lexical processing. It applies the following rules
// iteratively until no further processing can be done:
//
//  1. Replace multiple Separator elements with a single one.
//  2. Eliminate each . path name element (the current directory).
//  3. Eliminate each inner .. path name element (the parent directory)
//     along with the non-.. element that precedes it.
//  4. Eliminate .. elements that begin a rooted path:
//     that is, replace "/.." by "/" at the beginning of a path,
//     assuming Separator is '/'.
//
// The returned path ends in a slash only if it represents a root directory,
// such as "/" on Unix or `C:\` on Windows.
//
// Finally, any occurrences of slash are replaced by Separator.
//
// If the result of this process is an empty string, Clean
// returns the string ".".
//
// See also Rob Pike, “Lexical File Names in Plan 9 or
// Getting Dot-Dot Right,”
// https://9p.io/sys/doc/lexnames.html
func (vfs *RoFS) Clean(path string) string {
	return vfs.baseFS.Clean(path)
}

// Create creates the named file with mode 0666 (before umask), truncating
// it if it already exists. If successful, methods on the returned
// File can be used for I/O; the associated file descriptor has mode
// O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Create(name string) (avfs.File, error) {
	const op = "open"

	return &RoFile{}, &fs.PathError{Op: op, Path: name, Err: vfs.errPermDenied}
}

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (vfs *RoFS) CreateTemp(dir, pattern string) (avfs.File, error) {
	const op = "createtemp"

	return &RoFile{}, &fs.PathError{Op: op, Path: dir, Err: vfs.errPermDenied}
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *RoFS) Dir(path string) string {
	return vfs.baseFS.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *RoFS) EvalSymlinks(path string) (string, error) {
	return vfs.baseFS.EvalSymlinks(path)
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *RoFS) FromSlash(path string) string {
	return vfs.baseFS.FromSlash(path)
}

// Getwd returns a rooted path name corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *RoFS) Getwd() (dir string, err error) {
	return vfs.baseFS.Getwd()
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (vfs *RoFS) Glob(pattern string) (matches []string, err error) {
	return vfs.baseFS.Glob(pattern)
}

// Idm returns the identity manager of the file system.
// If the file system does not have an identity manager, avfs.DummyIdm is returned.
func (vfs *RoFS) Idm() avfs.IdentityMgr {
	return dummyidm.NotImplementedIdm
}

// IsAbs reports whether the path is absolute.
func (vfs *RoFS) IsAbs(path string) bool {
	return vfs.baseFS.IsAbs(path)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *RoFS) IsPathSeparator(c uint8) bool {
	return vfs.baseFS.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (vfs *RoFS) Join(elem ...string) string {
	return vfs.baseFS.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *RoFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	return &fs.PathError{Op: op, Path: name, Err: vfs.errOpNotPermitted}
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *RoFS) Link(oldname, newname string) error {
	const op = "link"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.errPermDenied}
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Lstat(name string) (fs.FileInfo, error) {
	return vfs.baseFS.Lstat(name)
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
func (vfs *RoFS) Match(pattern, name string) (matched bool, err error) {
	return vfs.baseFS.Match(pattern, name)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Mkdir(name string, perm fs.FileMode) error {
	const op = "mkdir"

	return &fs.PathError{Op: op, Path: name, Err: vfs.errPermDenied}
}

// MkdirAll creates a directory named path,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *RoFS) MkdirAll(path string, perm fs.FileMode) error {
	const op = "mkdir"

	return &fs.PathError{Op: op, Path: path, Err: vfs.errPermDenied}
}

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (vfs *RoFS) MkdirTemp(dir, prefix string) (name string, err error) {
	const op = "mkdirtemp"

	return "", &fs.PathError{Op: op, Path: dir, Err: vfs.errPermDenied}
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Open(name string) (avfs.File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) OpenFile(name string, flag int, perm fs.FileMode) (avfs.File, error) {
	const op = "open"

	if flag != os.O_RDONLY {
		return &RoFile{}, &fs.PathError{Op: op, Path: name, Err: vfs.errPermDenied}
	}

	fBase, err := vfs.baseFS.OpenFile(name, os.O_RDONLY, 0)

	f := &RoFile{baseFile: fBase, vfs: vfs}

	return f, err
}

// PathSeparator return the OS-specific path separator.
func (vfs *RoFS) PathSeparator() uint8 {
	return vfs.baseFS.PathSeparator()
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *RoFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return vfs.baseFS.ReadDir(name)
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func (vfs *RoFS) ReadFile(filename string) ([]byte, error) {
	return vfs.baseFS.ReadFile(filename)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Readlink(name string) (string, error) {
	return vfs.baseFS.Readlink(name)
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (vfs *RoFS) Rel(basepath, targpath string) (string, error) {
	return vfs.baseFS.Rel(basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Remove(name string) error {
	const op = "remove"

	return &fs.PathError{Op: op, Path: name, Err: vfs.errPermDenied}
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) RemoveAll(path string) error {
	const op = "removeall"

	return &fs.PathError{Op: op, Path: path, Err: vfs.errPermDenied}
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *RoFS) Rename(oldname, newname string) error {
	const op = "rename"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.errPermDenied}
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (vfs *RoFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	return vfs.baseFS.SameFile(fi1, fi2)
}

// SetUMask sets the file mode creation mask.
// Setting Umask is disabled for read only file systems.
func (vfs *RoFS) SetUMask(mask fs.FileMode) {
	vfs.baseFS.SetUMask(mask)
}

// SetUser sets and returns the current user.
// If the user is not found, the returned error is of type UnknownUserError.
func (vfs *RoFS) SetUser(name string) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *RoFS) Split(path string) (dir, file string) {
	return vfs.baseFS.Split(path)
}

// SplitAbs splits an absolute path immediately preceding the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, splitPath returns an empty dir
// and file set to path.
// The returned values have the property that path = dir + PathSeparator + file.
func (vfs *RoFS) SplitAbs(path string) (dir, file string) {
	return vfs.baseFS.SplitAbs(path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Stat(name string) (fs.FileInfo, error) {
	return vfs.baseFS.Stat(name)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (vfs *RoFS) Sub(dir string) (avfs.VFS, error) {
	return vfs.baseFS.Sub(dir)
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *RoFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	err := vfs.errPermDenied
	if vfs.OSType() == avfs.OsWindows {
		err = avfs.ErrWinPrivilegeNotHeld
	}

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: err}
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
func (vfs *RoFS) TempDir() string {
	return vfs.baseFS.TempDir()
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *RoFS) ToSlash(path string) string {
	return vfs.baseFS.ToSlash(path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *RoFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return vfs.baseFS.ToSysStat(info)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) Truncate(name string, size int64) error {
	const op = "truncate"

	return &fs.PathError{Op: op, Path: name, Err: vfs.errPermDenied}
}

// UMask returns the file mode creation mask.
func (vfs *RoFS) UMask() fs.FileMode {
	return vfs.baseFS.UMask()
}

// User returns the current user.
func (vfs *RoFS) User() avfs.UserReader {
	return vfs.baseFS.User()
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
func (vfs *RoFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return vfs.baseFS.WalkDir(root, fn)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func (vfs *RoFS) WriteFile(filename string, data []byte, perm fs.FileMode) error {
	const op = "open"

	return &fs.PathError{Op: op, Path: filename, Err: vfs.errPermDenied}
}
