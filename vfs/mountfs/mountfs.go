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

// Package mountfs implements a file system composed of different file systems.
package mountfs

import (
	"io/fs"
	"os"
	"time"

	"github.com/avfs/avfs"
)

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls [Clean] on the result.
func (vfs *MountFS) Abs(path string) (string, error) {
	if vfs.IsAbs(path) {
		return vfs.Clean(path), nil
	}

	return vfs.Join(vfs.CurDir(), path), nil
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *MountFS) Base(path string) string {
	return avfs.Base(vfs, path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Chdir(dir string) error {
	mnt, vfsPath := vfs.pathToMount(dir)

	err := mnt.vfs.Chdir(vfsPath)
	if err != nil {
		return mnt.restoreError(err)
	}

	vfs.curMnt = mnt
	_ = vfs.SetCurDir(vfsPath)

	return nil
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
func (vfs *MountFS) Chmod(name string, mode fs.FileMode) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Chmod(vfsPath, mode)

	return mnt.restoreError(err)
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *MountFS) Chown(name string, uid, gid int) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Chown(vfsPath, uid, gid)

	return mnt.restoreError(err)
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Chtimes(name string, atime, mtime time.Time) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Chtimes(vfsPath, atime, mtime)

	return mnt.restoreError(err)
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
func (vfs *MountFS) Clean(path string) string {
	return avfs.Clean(vfs, path)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Create(name string) (avfs.File, error) {
	return avfs.Create(vfs, name)
}

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (vfs *MountFS) CreateTemp(dir, pattern string) (avfs.File, error) {
	return avfs.CreateTemp(vfs, dir, pattern)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *MountFS) Dir(path string) string {
	return avfs.Dir(vfs, path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *MountFS) EvalSymlinks(path string) (string, error) {
	op := "lstat"
	err := error(avfs.ErrPermDenied)

	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
		err = avfs.ErrWinAccessDenied
	}

	return "", &fs.PathError{Op: op, Path: path, Err: err}
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *MountFS) FromSlash(path string) string {
	return avfs.FromSlash(vfs, path)
}

// Getwd returns a rooted name link corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *MountFS) Getwd() (dir string, err error) {
	cd := vfs.curMnt.toAbsPath(vfs.CurDir())

	return cd, nil
}

// IsAbs reports whether the path is absolute.
func (vfs *MountFS) IsAbs(path string) bool {
	return avfs.IsAbs(vfs, path)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *MountFS) IsPathSeparator(c uint8) bool {
	return avfs.IsPathSeparator(vfs, c)
}

// Join joins any number of path elements into a single path,
// separating them with an OS specific Separator. Empty elements
// are ignored. The result is Cleaned. However, if the argument
// list is empty or all its elements are empty, Join returns
// an empty string.
// On Windows, the result will only be a UNC path if the first
// non-empty element is a UNC path.
func (vfs *MountFS) Join(elem ...string) string {
	return avfs.Join(vfs, elem...)
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (vfs *MountFS) Glob(pattern string) (matches []string, err error) {
	matches, err = avfs.Glob(vfs, pattern)

	/*
		TODO : restore path if possible or disable function
		for i, m := range matches {
			matches[i] = vfs.fromBasePath(m)
		}
	*/

	return matches, err
}

// Idm returns the identity manager of the file system.
// If the file system does not have an identity manager, avfs.DummyIdm is returned.
func (vfs *MountFS) Idm() avfs.IdentityMgr {
	return vfs.rootFS.Idm()
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *MountFS) Lchown(name string, uid, gid int) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Lchown(vfsPath, uid, gid)

	return mnt.restoreError(err)
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *MountFS) Link(oldname, newname string) error {
	const op = ""

	oldMnt, oldVfsPath := vfs.pathToMount(oldname)
	newMnt, newVfsPath := vfs.pathToMount(newname)

	if oldMnt != newMnt {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrCrossDevLink}
	}

	err := oldMnt.vfs.Link(oldVfsPath, newVfsPath)

	return oldMnt.restoreError(err)
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Lstat(path string) (fs.FileInfo, error) {
	mnt, vfsPath := vfs.pathToMount(path)
	info, err := mnt.vfs.Lstat(vfsPath)

	return info, mnt.restoreError(err)
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
func (vfs *MountFS) Match(pattern, name string) (matched bool, err error) {
	return avfs.Match(vfs, pattern, name)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Mkdir(name string, perm fs.FileMode) error {
	if name == "" {
		err := error(avfs.ErrNoSuchFileOrDir)
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinPathNotFound
		}

		return &fs.PathError{Op: "mkdir", Path: "", Err: err}
	}

	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Mkdir(vfsPath, perm)

	return mnt.restoreError(err)
}

// MkdirAll creates a directory named name,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If name is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *MountFS) MkdirAll(path string, perm fs.FileMode) error {
	absPath, _ := vfs.rootFS.Abs(path)
	mnt := vfs.rootMnt
	mntPos := 0

	pi := avfs.NewPathIterator(vfs, absPath)
	for pi.Next() {
		m, ok := vfs.mounts[pi.LeftPart()]
		if ok {
			vfsPath := absPath[mntPos:pi.End()]

			err := mnt.vfs.MkdirAll(vfsPath, perm)
			if err != nil {
				return mnt.restoreError(err)
			}

			mnt = m
			mntPos = pi.End()
		}
	}

	err := mnt.vfs.MkdirAll(absPath[mntPos:], perm)

	return mnt.restoreError(err)
}

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (vfs *MountFS) MkdirTemp(dir, pattern string) (string, error) {
	return avfs.MkdirTemp(vfs, dir, pattern)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Open(name string) (avfs.File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) OpenFile(name string, flag int, perm fs.FileMode) (avfs.File, error) {
	mnt, vfsPath := vfs.pathToMount(name)

	f, err := mnt.vfs.OpenFile(vfsPath, flag, perm)
	if err != nil {
		return f, mnt.restoreError(err)
	}

	mf := &MountFile{
		vfs:   vfs,
		mount: mnt,
		file:  f,
	}

	return mf, nil
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *MountFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return avfs.ReadDir(vfs, name)
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
func (vfs *MountFS) ReadFile(name string) ([]byte, error) {
	return avfs.ReadFile(vfs, name)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Readlink(name string) (string, error) {
	const op = "readlink"

	if vfs.OSType() == avfs.OsWindows {
		return "", &fs.PathError{Op: op, Path: name, Err: avfs.ErrWinNotReparsePoint}
	}

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
func (vfs *MountFS) Rel(basepath, targpath string) (string, error) {
	return avfs.Rel(vfs, basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Remove(name string) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Remove(vfsPath)

	return mnt.restoreError(err)
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) RemoveAll(path string) error {
	if path == "" {
		// fail silently to retain compatibility with previous behavior of RemoveAll.
		return nil
	}

	mnt, vfsPath := vfs.pathToMount(path)
	err := mnt.vfs.RemoveAll(vfsPath)

	return mnt.restoreError(err)
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *MountFS) Rename(oldname, newname string) error {
	const op = "rename"

	oldMnt, oldVfsPath := vfs.pathToMount(oldname)
	newMnt, newVfsPath := vfs.pathToMount(newname)

	if oldMnt != newMnt {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrCrossDevLink}
	}

	err := oldMnt.vfs.Rename(oldVfsPath, newVfsPath)

	return oldMnt.restoreError(err)
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (vfs *MountFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	return vfs.rootFS.SameFile(fi1, fi2)
}

// SetUMask sets the file mode creation mask.
func (vfs *MountFS) SetUMask(mask fs.FileMode) error {
	return vfs.rootFS.SetUMask(mask)
}

// SetUserByName sets the current user by name.
// If the user is not found, the returned error is of type UnknownUserError.
func (vfs *MountFS) SetUserByName(name string) error {
	return avfs.SetUserByName(vfs, name)
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *MountFS) Split(path string) (dir, file string) {
	return avfs.Split(vfs, path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Stat(path string) (fs.FileInfo, error) {
	mnt, vfsPath := vfs.pathToMount(path)
	info, err := mnt.vfs.Stat(vfsPath)

	return info, mnt.restoreError(err)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (vfs *MountFS) Sub(dir string) (avfs.VFS, error) {
	mnt, vfsPath := vfs.pathToMount(dir)
	info, err := mnt.vfs.Sub(vfsPath)

	return info, mnt.restoreError(err)
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *MountFS) Symlink(oldname, newname string) error {
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
func (vfs *MountFS) TempDir() string {
	return avfs.TempDir(vfs)
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *MountFS) ToSlash(path string) string {
	return avfs.ToSlash(vfs, path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *MountFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return vfs.rootFS.ToSysStat(info)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *MountFS) Truncate(name string, size int64) error {
	mnt, vfsPath := vfs.pathToMount(name)
	err := mnt.vfs.Truncate(vfsPath, size)

	return mnt.restoreError(err)
}

// UMask returns the file mode creation mask.
func (vfs *MountFS) UMask() fs.FileMode {
	return vfs.rootFS.UMask()
}

// User returns the current user.
func (vfs *MountFS) User() avfs.UserReader {
	return vfs.rootFS.User()
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
func (vfs *MountFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	err := avfs.WalkDir(vfs, root, fn)

	return vfs.rootMnt.restoreError(err)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func (vfs *MountFS) WriteFile(filename string, data []byte, perm fs.FileMode) error {
	err := avfs.WriteFile(vfs, filename, data, perm)

	return vfs.rootMnt.restoreError(err)
}
