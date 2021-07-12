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

// Package basepathfs restricts all operations to a given path within an file system.
package basepathfs

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfsutils"
)

// file system functions.

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (vfs *BasePathFS) Abs(path string) (string, error) {
	absPath, _ := vfs.baseFS.Abs(path)

	return vfs.fromBasePath(absPath), nil
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *BasePathFS) Base(path string) string {
	return vfsutils.Base(path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Chdir(dir string) error {
	err := vfs.baseFS.Chdir(vfs.toBasePath(dir))

	return vfs.restoreError(err)
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
func (vfs *BasePathFS) Chmod(name string, mode fs.FileMode) error {
	err := vfs.baseFS.Chmod(vfs.toBasePath(name), mode)

	return vfs.restoreError(err)
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *BasePathFS) Chown(name string, uid, gid int) error {
	err := vfs.baseFS.Chown(vfs.toBasePath(name), uid, gid)

	return vfs.restoreError(err)
}

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Chroot(path string) error {
	const op = "chroot"

	return &fs.PathError{Op: op, Path: path, Err: avfs.ErrOpNotPermitted}
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Chtimes(name string, atime, mtime time.Time) error {
	err := vfs.baseFS.Chtimes(vfs.toBasePath(name), atime, mtime)

	return vfs.restoreError(err)
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
func (vfs *BasePathFS) Clean(path string) string {
	return vfsutils.Clean(path)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Create(name string) (avfs.File, error) {
	return vfs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *BasePathFS) Dir(path string) string {
	return vfsutils.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *BasePathFS) EvalSymlinks(path string) (string, error) {
	const op = "lstat"

	return "", &fs.PathError{Op: op, Path: path, Err: avfs.ErrPermDenied}
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *BasePathFS) FromSlash(path string) string {
	return vfs.baseFS.FromSlash(path)
}

// GetTempDir returns the default directory to use for temporary files.
//
// On Unix systems, it returns $TMPDIR if non-empty, else /tmp.
// On Windows, it uses GetTempPath, returning the first non-empty
// value from %TMP%, %TEMP%, %USERPROFILE%, or the Windows directory.
// On Plan 9, it returns /tmp.
//
// The directory is neither guaranteed to exist nor have accessible
// permissions.
func (vfs *BasePathFS) GetTempDir() string {
	return vfs.baseFS.GetTempDir()
}

// GetUMask returns the file mode creation mask.
func (vfs *BasePathFS) GetUMask() fs.FileMode {
	return vfs.baseFS.GetUMask()
}

// Getwd returns a rooted name link corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *BasePathFS) Getwd() (dir string, err error) {
	cwd, err := vfs.baseFS.Getwd()
	if err != nil {
		return "", err
	}

	return vfs.fromBasePath(cwd), nil
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (vfs *BasePathFS) Glob(pattern string) (matches []string, err error) {
	return vfs.baseFS.Glob(vfs.toBasePath(pattern))
}

// IsAbs reports whether the path is absolute.
func (vfs *BasePathFS) IsAbs(path string) bool {
	return vfsutils.IsAbs(path)
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (vfs *BasePathFS) IsExist(err error) bool {
	return vfsutils.IsExist(err)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (vfs *BasePathFS) IsNotExist(err error) bool {
	return vfsutils.IsNotExist(err)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *BasePathFS) IsPathSeparator(c uint8) bool {
	return vfsutils.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (vfs *BasePathFS) Join(elem ...string) string {
	return vfsutils.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *BasePathFS) Lchown(name string, uid, gid int) error {
	err := vfs.baseFS.Lchown(vfs.toBasePath(name), uid, gid)

	return vfs.restoreError(err)
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *BasePathFS) Link(oldname, newname string) error {
	err := vfs.baseFS.Link(vfs.toBasePath(oldname), vfs.toBasePath(newname))

	return vfs.restoreError(err)
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Lstat(path string) (fs.FileInfo, error) {
	info, err := vfs.baseFS.Lstat(vfs.toBasePath(path))

	return info, vfs.restoreError(err)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Mkdir(name string, perm fs.FileMode) error {
	err := vfs.baseFS.Mkdir(vfs.toBasePath(name), perm)

	return vfs.restoreError(err)
}

// MkdirAll creates a directory named name,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If name is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *BasePathFS) MkdirAll(path string, perm fs.FileMode) error {
	err := vfs.baseFS.MkdirAll(vfs.toBasePath(path), perm)

	return vfs.restoreError(err)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Open(path string) (avfs.File, error) {
	return vfs.OpenFile(path, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) OpenFile(name string, flag int, perm fs.FileMode) (avfs.File, error) {
	f, err := vfs.baseFS.OpenFile(vfs.toBasePath(name), flag, perm)
	if err != nil {
		return f, vfs.restoreError(err)
	}

	bf := &BasePathFile{
		bpFS:     vfs,
		baseFile: f,
	}

	return bf, nil
}

// ReadDir reads the directory named by dirname and returns
// a list of directory entries sorted by filename.
func (vfs *BasePathFS) ReadDir(dirname string) ([]fs.FileInfo, error) {
	return vfsutils.ReadDir(vfs, dirname)
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func (vfs *BasePathFS) ReadFile(filename string) ([]byte, error) {
	return vfsutils.ReadFile(vfs, filename)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Readlink(name string) (string, error) {
	const op = "readlink"

	return "", &fs.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (vfs *BasePathFS) Rel(basepath, targpath string) (string, error) {
	return vfsutils.Rel(basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Remove(name string) error {
	err := vfs.baseFS.Remove(vfs.toBasePath(name))

	return vfs.restoreError(err)
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) RemoveAll(path string) error {
	err := vfs.baseFS.RemoveAll(vfs.toBasePath(path))
	if err != nil {
		return vfs.restoreError(err)
	}

	return nil
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *BasePathFS) Rename(oldname, newname string) error {
	err := vfs.baseFS.Rename(vfs.toBasePath(oldname), vfs.toBasePath(newname))

	return vfs.restoreError(err)
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (vfs *BasePathFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	return vfs.baseFS.SameFile(fi1, fi2)
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *BasePathFS) Split(path string) (dir, file string) {
	return vfsutils.Split(vfs, path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Stat(path string) (fs.FileInfo, error) {
	info, err := vfs.baseFS.Stat(vfs.toBasePath(path))

	return info, vfs.restoreError(err)
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *BasePathFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrPermDenied}
}

// TempDir creates a new temporary directory in the directory dir
// with a link beginning with prefix and returns the name of the
// new directory. If dir is the empty string, GetTempDir uses the
// default directory for temporary files (see os.GetTempDir).
// Multiple programs calling GetTempDir simultaneously
// will not choose the same directory. It is the caller's responsibility
// to removeNodes the directory when no longer needed.
func (vfs *BasePathFS) TempDir(dir, prefix string) (name string, err error) {
	return vfsutils.TempDir(vfs, dir, prefix)
}

// TempFile creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting *os.File.
// The filename is generated by taking pattern and adding a random
// string to the end. If pattern includes a "*", the random string
// replaces the last "*".
// If dir is the empty string, TempFile uses the default directory
// for temporary files (see os.TempDir).
// Multiple programs calling TempFile simultaneously
// will not choose the same file. The caller can use f.Type()
// to find the pathname of the file. It is the caller's responsibility
// to removeNodes the file when no longer needed.
func (vfs *BasePathFS) TempFile(dir, pattern string) (f avfs.File, err error) {
	return vfsutils.TempFile(vfs, dir, pattern)
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *BasePathFS) ToSlash(path string) string {
	return vfs.baseFS.ToSlash(path)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Truncate(name string, size int64) error {
	err := vfs.baseFS.Truncate(vfs.toBasePath(name), size)

	return vfs.restoreError(err)
}

// UMask sets the file mode creation mask.
func (vfs *BasePathFS) UMask(mask fs.FileMode) {
	vfs.baseFS.UMask(mask)
}

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.
func (vfs *BasePathFS) Walk(root string, walkFn filepath.WalkFunc) error {
	err := vfs.baseFS.Walk(vfs.toBasePath(root), func(path string, info fs.FileInfo, err error) error {
		return walkFn(vfs.fromBasePath(path), info, err)
	})

	return vfs.restoreError(err)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func (vfs *BasePathFS) WriteFile(filename string, data []byte, perm fs.FileMode) error {
	return vfsutils.WriteFile(vfs, filename, data, perm)
}
