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

// Package osfs implements a file system using functions from os and path/filepath packages.
//
// Most functions are just calls to the original ones from os and filepath packages.
package osfs

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/osidm"
)

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls [Clean] on the result.
func (vfs *OsFS) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *OsFS) Base(path string) string {
	return filepath.Base(path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Chdir(dir string) error {
	return os.Chdir(dir)
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
func (vfs *OsFS) Chmod(name string, mode fs.FileMode) error {
	return os.Chmod(name, mode)
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *OsFS) Chown(name string, uid, gid int) error {
	const op = "chown"

	if !vfs.HasFeature(avfs.FeatIdentityMgr) && vfs.OSType() != avfs.OsWindows {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrOpNotPermitted}
	}

	return os.Chown(name, uid, gid)
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying filesystem may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Chtimes(name string, atime, mtime time.Time) error {
	return os.Chtimes(name, atime, mtime)
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
func (vfs *OsFS) Clean(path string) string {
	return filepath.Clean(path)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Create(name string) (avfs.File, error) {
	return os.Create(name)
}

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by [TempDir].
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (vfs *OsFS) CreateTemp(dir, pattern string) (avfs.File, error) {
	return os.CreateTemp(dir, pattern)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls [Clean] on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *OsFS) Dir(path string) string {
	return filepath.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls [Clean] on the result.
func (vfs *OsFS) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *OsFS) FromSlash(path string) string {
	return filepath.FromSlash(path)
}

// Getwd returns a rooted path name corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *OsFS) Getwd() (dir string, err error) {
	return os.Getwd()
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in [Match]. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is [ErrBadPattern], when pattern
// is malformed.
func (vfs *OsFS) Glob(pattern string) (matches []string, err error) {
	return filepath.Glob(pattern)
}

// IsAbs reports whether the path is absolute.
func (vfs *OsFS) IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *OsFS) IsPathSeparator(c uint8) bool {
	return os.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path,
// separating them with an OS specific Separator. Empty elements
// are ignored. The result is Cleaned. However, if the argument
// list is empty or all its elements are empty, Join returns
// an empty string.
// On Windows, the result will only be a UNC path if the first
// non-empty element is a UNC path.
func (vfs *OsFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *OsFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	if !vfs.HasFeature(avfs.FeatIdentityMgr) && vfs.OSType() != avfs.OsWindows {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrOpNotPermitted}
	}

	return os.Lchown(name, uid, gid)
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *OsFS) Link(oldname, newname string) error {
	return os.Link(oldname, newname)
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Lstat(name string) (fs.FileInfo, error) {
	return os.Lstat(name)
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
// The only possible returned error is [ErrBadPattern], when pattern
// is malformed.
//
// On Windows, escaping is disabled. Instead, '\\' is treated as
// path separator.
func (vfs *OsFS) Match(pattern, name string) (matched bool, err error) {
	return filepath.Match(pattern, name)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Mkdir(name string, perm fs.FileMode) error {
	return os.Mkdir(name, perm)
}

// MkdirAll creates a directory named path,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *OsFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by [TempDir].
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (vfs *OsFS) MkdirTemp(dir, pattern string) (name string, err error) {
	return os.MkdirTemp(dir, pattern)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Open(name string) (avfs.File, error) {
	return os.Open(name)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) OpenFile(name string, flag int, perm fs.FileMode) (avfs.File, error) {
	return os.OpenFile(name, flag, perm)
}

// OSType returns the operating system type of the file system.
func (vfs *OsFS) OSType() avfs.OSType {
	return avfs.CurrentOSType()
}

// PathSeparator return the OS-specific path separator.
func (vfs *OsFS) PathSeparator() uint8 {
	return os.PathSeparator
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *OsFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
func (vfs *OsFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Readlink(name string) (string, error) {
	return os.Readlink(name)
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// [Join](basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls [Clean] on the result.
func (vfs *OsFS) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

// Remove removes the named file or directory.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Remove(name string) error {
	return os.Remove(name)
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
func (vfs *OsFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *OsFS) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's [Stat].
// It returns false in other cases.
func (vfs *OsFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	return os.SameFile(fi1, fi2)
}

// SetUMask sets the file mode creation mask.
func (vfs *OsFS) SetUMask(mask fs.FileMode) error {
	return avfs.SetUMask(mask)
}

// SetUser sets the current user.
// If the user can't be changed an error is returned.
func (vfs *OsFS) SetUser(user avfs.UserReader) error {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return avfs.ErrPermDenied
	}

	return osidm.SetUser(user)
}

// SetUserByName sets and returns the current user.
// If the user is not found, the returned error is of type UnknownUserError.
func (vfs *OsFS) SetUserByName(name string) error {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return avfs.ErrPermDenied
	}

	return osidm.SetUserByName(name)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *OsFS) Split(path string) (dir, file string) {
	return filepath.Split(path)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (vfs *OsFS) Sub(dir string) (avfs.VFS, error) {
	const op = "sub"

	// TODO:

	return nil, &fs.PathError{Op: op, Path: dir, Err: vfs.err.PermDenied}
}

// Symlink creates newname as a symbolic link to oldname.
// On Windows, a symlink to a non-existent oldname creates a file symlink;
// if oldname is later created as a directory the symlink will not work.
// If there is an error, it will be of type *LinkError.
func (vfs *OsFS) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
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
func (vfs *OsFS) TempDir() string {
	return os.TempDir()
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *OsFS) ToSlash(path string) string {
	return filepath.ToSlash(path)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
func (vfs *OsFS) Truncate(name string, size int64) error {
	return os.Truncate(name, size)
}

// UMask returns the file mode creation mask.
func (vfs *OsFS) UMask() fs.FileMode {
	return avfs.UMask()
}

// User returns the current user.
func (vfs *OsFS) User() avfs.UserReader {
	return osidm.User()
}

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root.
//
// All errors that arise visiting files and directories are filtered by fn:
// see the [fs.WalkDirFunc] documentation for details.
//
// The files are walked in lexical order, which makes the output deterministic
// but requires WalkDir to read an entire directory into memory before proceeding
// to walk that directory.
//
// WalkDir does not follow symbolic links.
func (vfs *OsFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
func (vfs *OsFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}
