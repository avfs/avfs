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
	"path/filepath"
	"time"
)

// BaseFS file system functions.

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (vfs *BaseFS) Abs(path string) (string, error) {

}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *BaseFS) Base(path string) string {

}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Chdir(dir string) error {
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
func (vfs *BaseFS) Chmod(name string, mode fs.FileMode) error {
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
func (vfs *BaseFS) Chown(name string, uid, gid int) error {
	const op = "chown"

	return &fs.PathError{Op: op, Path: name, Err: ErrOpNotPermitted}
}

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Chroot(path string) error {
	const op = "chroot"

	return &fs.PathError{Op: op, Path: path, Err: ErrOpNotPermitted}
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Chtimes(name string, atime, mtime time.Time) error {
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
func (vfs *BaseFS) Clean(path string) string {

}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Create(name string) (File, error) {
	return vfs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (vfs *BaseFS) CreateTemp(dir, pattern string) (File, error) {

}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *BaseFS) Dir(path string) string {

}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *BaseFS) EvalSymlinks(path string) (string, error) {
	const op = "lstat"

	return "", &fs.PathError{Op: op, Path: path, Err: ErrPermDenied}
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *BaseFS) FromSlash(path string) string {
	return filepath.FromSlash(path)
}

// GetUMask returns the file mode creation mask.
func (vfs *BaseFS) GetUMask() fs.FileMode {
	return 0o022
}

// Getwd returns a rooted path name corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *BaseFS) Getwd() (dir string, err error) {
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
func (vfs *BaseFS) Glob(pattern string) (matches []string, err error) {

}

// IsAbs reports whether the path is absolute.
func (vfs *BaseFS) IsAbs(path string) bool {

}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (vfs *BaseFS) IsExist(err error) bool {

}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (vfs *BaseFS) IsNotExist(err error) bool {

}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *BaseFS) IsPathSeparator(c uint8) bool {

}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (vfs *BaseFS) Join(elem ...string) string {

}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *BaseFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	return &fs.PathError{Op: op, Path: name, Err: ErrOpNotPermitted}
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *BaseFS) Link(oldname, newname string) error {
	const op = "link"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: ErrPermDenied}
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Lstat(name string) (fs.FileInfo, error) {
	const op = "lstat"

	return nil, &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Mkdir(name string, perm fs.FileMode) error {
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
func (vfs *BaseFS) MkdirAll(path string, perm fs.FileMode) error {
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
func (vfs *BaseFS) MkdirTemp(dir, prefix string) (name string, err error) {

}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Open(name string) (File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned DummyFile can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	const op = "open"

	return &BaseFile{}, &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *BaseFS) ReadDir(name string) ([]fs.DirEntry, error) {

}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Readlink(name string) (string, error) {
	const op = "readlink"

	return "", &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func (vfs *BaseFS) ReadFile(filename string) ([]byte, error) {

}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (vfs *BaseFS) Rel(basepath, targpath string) (string, error) {

}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Remove(name string) error {
	const op = "remove"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) RemoveAll(path string) error {
	const op = "removeall"

	return &fs.PathError{Op: op, Path: path, Err: ErrPermDenied}
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *BaseFS) Rename(oldname, newname string) error {
	const op = "rename"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: ErrPermDenied}
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (vfs *BaseFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	return false
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *BaseFS) Split(path string) (dir, file string) {

}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Stat(name string) (fs.FileInfo, error) {
	const op = "stat"

	return nil, &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *BaseFS) Symlink(oldname, newname string) error {
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
func (vfs *BaseFS) TempDir() string {
	return TmpDir
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *BaseFS) ToSlash(path string) string {
	return filepath.ToSlash(path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *BaseFS) ToSysStat(info fs.FileInfo) SysStater {
	return &BaseSysStat{}
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Truncate(name string, size int64) error {
	const op = "truncate"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// UMask sets the file mode creation mask.
func (vfs *BaseFS) UMask(mask fs.FileMode) {
	_ = mask
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
func (vfs *BaseFS) WalkDir(root string, fn fs.WalkDirFunc) error {

}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func (vfs *BaseFS) WriteFile(filename string, data []byte, perm fs.FileMode) error {

}