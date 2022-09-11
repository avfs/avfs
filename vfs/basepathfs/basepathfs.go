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

// Package basepathfs restricts all operations to a given path within a file system.
package basepathfs

import (
	"io/fs"
	"os"
	"time"

	"github.com/avfs/avfs"
)

// file system functions.

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (vfs *BasePathFS) Abs(path string) (string, error) {
	abs, _ := vfs.baseFS.Abs(vfs.toBasePath(path))

	return vfs.fromBasePath(abs), nil
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *BasePathFS) Base(path string) string {
	base := vfs.baseFS.Base(path)

	return vfs.fromBasePath(base)
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
func (vfs *BasePathFS) Clean(path string) string {
	return vfs.Utils.Clean(path)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Create(name string) (avfs.File, error) {
	return vfs.Utils.Create(vfs, name)
}

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (vfs *BasePathFS) CreateTemp(dir, pattern string) (avfs.File, error) {
	return vfs.Utils.CreateTemp(vfs, dir, pattern)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *BasePathFS) Dir(path string) string {
	return vfs.Utils.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *BasePathFS) EvalSymlinks(path string) (string, error) {
	op := "lstat"
	err := error(avfs.ErrPermDenied)

	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
		err = avfs.ErrWinAccessDenied
	}

	return "", &fs.PathError{Op: op, Path: path, Err: err}
}

// Getwd returns a rooted name link corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *BasePathFS) Getwd() (dir string, err error) {
	dir, err = vfs.baseFS.Getwd()

	return vfs.fromBasePath(dir), vfs.restoreError(err)
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
	matches, err = vfs.Utils.Glob(vfs, pattern)

	for i, m := range matches {
		matches[i] = vfs.fromBasePath(m)
	}

	return matches, err
}

// Idm returns the identity manager of the file system.
// If the file system does not have an identity manager, avfs.DummyIdm is returned.
func (vfs *BasePathFS) Idm() avfs.IdentityMgr {
	return vfs.baseFS.Idm()
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
	if name == "" {
		err := error(avfs.ErrNoSuchFileOrDir)
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinPathNotFound
		}

		return &fs.PathError{Op: "mkdir", Path: "", Err: err}
	}

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

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (vfs *BasePathFS) MkdirTemp(dir, prefix string) (name string, err error) {
	return vfs.Utils.MkdirTemp(vfs, dir, prefix)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Open(path string) (avfs.File, error) {
	return vfs.Utils.Open(vfs, path)
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
		vfs:      vfs,
		baseFile: f,
	}

	return bf, nil
}

// ReadDir reads the directory named by dirname and returns
// a list of directory entries sorted by filename.
func (vfs *BasePathFS) ReadDir(dirname string) ([]fs.DirEntry, error) {
	return vfs.Utils.ReadDir(vfs, dirname)
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func (vfs *BasePathFS) ReadFile(filename string) ([]byte, error) {
	return vfs.Utils.ReadFile(vfs, filename)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Readlink(name string) (string, error) {
	const op = "readlink"

	if vfs.OSType() == avfs.OsWindows {
		return "", &fs.PathError{Op: op, Path: name, Err: avfs.ErrWinNotReparsePoint}
	}

	return "", &fs.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
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

	return vfs.restoreError(err)
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

// SetUMask sets the file mode creation mask.
func (vfs *BasePathFS) SetUMask(mask fs.FileMode) {
	vfs.baseFS.SetUMask(mask)
}

// SetUser sets and returns the current user.
// If the user is not found, the returned error is of type UnknownUserError.
func (vfs *BasePathFS) SetUser(name string) (avfs.UserReader, error) {
	return vfs.baseFS.SetUser(name)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Stat(path string) (fs.FileInfo, error) {
	info, err := vfs.baseFS.Stat(vfs.toBasePath(path))

	return info, vfs.restoreError(err)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (vfs *BasePathFS) Sub(dir string) (avfs.VFS, error) {
	subFS, err := vfs.baseFS.Sub(vfs.toBasePath(dir))

	return subFS, vfs.restoreError(err)
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *BasePathFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	if vfs.OSType() == avfs.OsWindows {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrWinPrivilegeNotHeld}
	}

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrPermDenied}
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
func (vfs *BasePathFS) TempDir() string {
	return vfs.baseFS.TempDir()
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *BasePathFS) ToSlash(path string) string {
	return vfs.baseFS.ToSlash(path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *BasePathFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return vfs.baseFS.ToSysStat(info)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) Truncate(name string, size int64) error {
	err := vfs.baseFS.Truncate(vfs.toBasePath(name), size)

	return vfs.restoreError(err)
}

// UMask returns the file mode creation mask.
func (vfs *BasePathFS) UMask() fs.FileMode {
	return vfs.baseFS.UMask()
}

// User returns the current user.
func (vfs *BasePathFS) User() avfs.UserReader {
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
func (vfs *BasePathFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	err := vfs.Utils.WalkDir(vfs, root, fn)

	return vfs.restoreError(err)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func (vfs *BasePathFS) WriteFile(filename string, data []byte, perm fs.FileMode) error {
	err := vfs.Utils.WriteFile(vfs, filename, data, perm)

	return vfs.restoreError(err)
}
