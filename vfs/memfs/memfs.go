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

// Package memfs implements an in memory file system.
//
// it supports several features : symbolic links, hard links, Chroot  ....
package memfs

import (
	"io/fs"
	"os"
	"sync/atomic"
	"time"

	"github.com/avfs/avfs"
)

// file system functions.

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (vfs *MemFS) Abs(path string) (string, error) {
	return vfs.utils.Abs(vfs, path)
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *MemFS) Base(path string) string {
	return vfs.utils.Base(path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Chdir(dir string) error {
	const op = "chdir"

	_, child, absPath, _, _, err := vfs.searchNode(dir, slmLstat)
	if err != avfs.ErrFileExists {
		return &fs.PathError{Op: op, Path: dir, Err: err}
	}

	c, ok := child.(*dirNode)
	if !ok {
		return &fs.PathError{Op: op, Path: dir, Err: avfs.ErrNotADirectory}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.checkPermission(avfs.PermLookup, vfs.user) {
		return &fs.PathError{Op: op, Path: dir, Err: avfs.ErrPermDenied}
	}

	vfs.curDir = absPath

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
func (vfs *MemFS) Chmod(name string, mode fs.FileMode) error {
	const op = "chmod"

	_, child, _, _, _, err := vfs.searchNode(name, slmEval)
	if err != avfs.ErrFileExists || child == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	child.Lock()
	defer child.Unlock()

	err = child.setMode(mode, vfs.user)
	if err != nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	return nil
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *MemFS) Chown(name string, uid, gid int) error {
	const op = "chown"

	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsAdmin() {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrOpNotPermitted}
	}

	_, child, _, _, _, err := vfs.searchNode(name, slmEval)
	if err != avfs.ErrFileExists || child == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	child.Lock()
	child.setOwner(uid, gid)
	child.Unlock()

	return nil
}

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Chroot(path string) error {
	const op = "chroot"

	if !vfs.user.IsAdmin() {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrOpNotPermitted}
	}

	_, child, _, _, _, err := vfs.searchNode(path, slmEval)
	if err != avfs.ErrFileExists || child == nil {
		return &fs.PathError{Op: op, Path: path, Err: err}
	}

	c, ok := child.(*dirNode)
	if !ok {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrNotADirectory}
	}

	vfs.rootNode = c

	return nil
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Chtimes(name string, atime, mtime time.Time) error {
	const op = "chtimes"

	_, child, _, _, _, err := vfs.searchNode(name, slmLstat)
	if err != avfs.ErrFileExists || child == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	child.Lock()
	defer child.Unlock()

	err = child.setModTime(mtime, vfs.user)
	if err != nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	return nil
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
func (vfs *MemFS) Clean(path string) string {
	return vfs.utils.Clean(path)
}

// Clone returns a shallow copy of the current file system.
func (vfs *MemFS) Clone() avfs.VFS {
	newFs := *vfs

	return &newFs
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Create(name string) (avfs.File, error) {
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
func (vfs *MemFS) CreateTemp(dir, pattern string) (avfs.File, error) {
	return vfs.utils.CreateTemp(vfs, dir, pattern)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *MemFS) Dir(path string) string {
	return vfs.utils.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *MemFS) EvalSymlinks(path string) (string, error) {
	const op = "lstat"

	_, _, absPath, _, end, err := vfs.searchNode(path, slmEval)
	if err != avfs.ErrFileExists {
		return "", &fs.PathError{Op: op, Path: absPath[:end], Err: avfs.ErrNoSuchFileOrDir}
	}

	return absPath, nil
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *MemFS) FromSlash(path string) string {
	return vfs.utils.FromSlash(path)
}

// Getwd returns a rooted name link corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *MemFS) Getwd() (dir string, err error) {
	dir = vfs.curDir

	return dir, nil
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (vfs *MemFS) Glob(pattern string) (matches []string, err error) {
	return vfs.utils.Glob(vfs, pattern)
}

// Idm returns the identity manager of the file system.
// if the file system does not have an identity manager, avfs.DummyIdm is returned.
func (vfs *MemFS) Idm() avfs.IdentityMgr {
	return vfs.memAttrs.idm
}

// IsAbs reports whether the path is absolute.
func (vfs *MemFS) IsAbs(path string) bool {
	return vfs.utils.IsAbs(path)
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (vfs *MemFS) IsExist(err error) bool {
	return vfs.utils.IsExist(err)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (vfs *MemFS) IsNotExist(err error) bool {
	return vfs.utils.IsNotExist(err)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *MemFS) IsPathSeparator(c uint8) bool {
	return vfs.utils.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (vfs *MemFS) Join(elem ...string) string {
	return vfs.utils.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *MemFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsAdmin() {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrOpNotPermitted}
	}

	_, child, _, _, _, err := vfs.searchNode(name, slmLstat)
	if err != avfs.ErrFileExists || child == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	child.Lock()
	child.setOwner(uid, gid)
	child.Unlock()

	return nil
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *MemFS) Link(oldname, newname string) error {
	const op = "link"

	_, oChild, _, _, _, oerr := vfs.searchNode(oldname, slmLstat)
	if oerr != avfs.ErrFileExists || oChild == nil {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: oerr}
	}

	nParent, _, absPath, start, end, nerr := vfs.searchNode(newname, slmLstat)
	if nParent == nil || nerr != avfs.ErrNoSuchFileOrDir {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: nerr}
	}

	nParent.mu.Lock()
	defer nParent.mu.Unlock()

	if !nParent.checkPermission(avfs.PermWrite, vfs.user) {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrOpNotPermitted}
	}

	c, ok := oChild.(*fileNode)
	if !ok {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrOpNotPermitted}
	}

	part := absPath[start:end]

	c.mu.Lock()
	nParent.addChild(part, c)
	c.nlink++
	c.mu.Unlock()

	return nil
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Lstat(path string) (fs.FileInfo, error) {
	const op = "lstat"

	_, child, absPath, start, end, err := vfs.searchNode(path, slmLstat)
	if err != avfs.ErrFileExists || child == nil {
		return nil, &fs.PathError{Op: op, Path: path, Err: err}
	}

	part := absPath[start:end]
	fst := child.fillStatFrom(part)

	return fst, nil
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
func (vfs *MemFS) Match(pattern, name string) (matched bool, err error) {
	return vfs.utils.Match(pattern, name)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Mkdir(name string, perm fs.FileMode) error {
	const op = "mkdir"

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: avfs.ErrNoSuchFileOrDir}
	}

	parent, _, absPath, start, end, err := vfs.searchNode(name, slmEval)
	if err != avfs.ErrNoSuchFileOrDir || end != len(absPath) || parent == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if !parent.checkPermission(avfs.PermWrite|avfs.PermLookup, vfs.user) {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
	}

	part := absPath[start:end]
	if parent.child(part) != nil {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrFileExists}
	}

	_ = vfs.createDir(parent, part, perm)

	return nil
}

// MkdirAll creates a directory named name,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If name is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *MemFS) MkdirAll(path string, perm fs.FileMode) error {
	const op = "mkdir"

	parent, child, absPath, start, end, err := vfs.searchNode(path, slmEval)
	if err == avfs.ErrFileExists {
		if _, ok := child.(*dirNode); !ok {
			return &fs.PathError{Op: op, Path: path, Err: avfs.ErrNotADirectory}
		}

		return nil
	}

	if parent == nil || err == avfs.ErrNotADirectory {
		return &fs.PathError{Op: op, Path: absPath[:end], Err: err}
	}

	if err != avfs.ErrNoSuchFileOrDir {
		return &fs.PathError{Op: op, Path: path, Err: err}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if !parent.checkPermission(avfs.PermWrite|avfs.PermLookup, vfs.user) {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrPermDenied}
	}

	ut := vfs.utils
	for dn, isLast := parent, len(absPath) <= 1; !isLast; start = end + 1 {
		end, isLast = ut.SegmentPath(absPath, start)

		part := absPath[start:end]
		if dn.child(part) != nil {
			return nil
		}

		dn = vfs.createDir(dn, part, perm)
	}

	return nil
}

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (vfs *MemFS) MkdirTemp(dir, pattern string) (string, error) {
	return vfs.utils.MkdirTemp(vfs, dir, pattern)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Open(name string) (avfs.File, error) {
	return vfs.utils.Open(vfs, name)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) OpenFile(name string, flag int, perm fs.FileMode) (avfs.File, error) {
	const op = "open"

	f := &MemFile{
		memFS: vfs,
		name:  name,
	}

	if flag == os.O_RDONLY || flag&os.O_RDWR != 0 {
		f.permMode = avfs.PermRead
	}

	if flag&(os.O_APPEND|os.O_CREATE|os.O_RDWR|os.O_TRUNC|os.O_WRONLY) != 0 {
		f.permMode |= avfs.PermWrite
	}

	parent, child, absPath, start, end, err := vfs.searchNode(name, slmEval)
	if err != avfs.ErrFileExists && err != avfs.ErrNoSuchFileOrDir {
		return &MemFile{}, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if err == avfs.ErrNoSuchFileOrDir {
		if end < len(absPath) || parent == nil {
			return nil, &fs.PathError{Op: op, Path: name, Err: err}
		}

		if flag&os.O_CREATE == 0 {
			return &MemFile{}, &fs.PathError{Op: op, Path: name, Err: err}
		}

		parent.mu.Lock()
		defer parent.mu.Unlock()

		if f.permMode&avfs.PermWrite == 0 || !parent.checkPermission(avfs.PermWrite|avfs.PermLookup, vfs.user) {
			return &MemFile{}, &fs.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
		}

		part := absPath[start:end]

		child = parent.child(part)
		if child == nil {
			child = vfs.createFile(parent, part, perm)
			f.nd = child

			return f, nil
		}
	}

	switch c := child.(type) {
	case *fileNode:
		c.mu.Lock()
		defer c.mu.Unlock()

		if !c.checkPermission(f.permMode, vfs.user) {
			return &MemFile{}, &fs.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
		}

		if flag&(os.O_CREATE|os.O_EXCL) == os.O_CREATE|os.O_EXCL {
			return &MemFile{}, &fs.PathError{Op: op, Path: name, Err: avfs.ErrFileExists}
		}

		if flag&os.O_TRUNC != 0 {
			c.truncate(0)
		}

		if flag&os.O_APPEND != 0 {
			f.at = c.size()
		}

	case *dirNode:
		c.mu.Lock()
		defer c.mu.Unlock()

		if f.permMode&avfs.PermWrite != 0 {
			return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: avfs.ErrIsADirectory}
		}

		if !c.checkPermission(f.permMode, vfs.user) {
			return &MemFile{}, &fs.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
		}

	default:
		return &MemFile{}, fs.ErrInvalid
	}

	f.nd = child

	return f, nil
}

// PathSeparator return the OS-specific path separator.
func (vfs *MemFS) PathSeparator() uint8 {
	return vfs.utils.PathSeparator()
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *MemFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return vfs.utils.ReadDir(vfs, name)
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
func (vfs *MemFS) ReadFile(name string) ([]byte, error) {
	return vfs.utils.ReadFile(vfs, name)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Readlink(name string) (string, error) {
	const op = "readlink"

	_, child, _, _, _, err := vfs.searchNode(name, slmLstat)
	if err != avfs.ErrFileExists {
		return "", &fs.PathError{Op: op, Path: name, Err: err}
	}

	sl, ok := child.(*symlinkNode)
	if !ok {
		return "", &fs.PathError{Op: op, Path: name, Err: avfs.ErrInvalidArgument}
	}

	return sl.link, nil
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (vfs *MemFS) Rel(basepath, targpath string) (string, error) {
	return vfs.utils.Rel(basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Remove(name string) error {
	const op = "remove"

	parent, child, absPath, start, end, err := vfs.searchNode(name, slmLstat)
	if err != avfs.ErrFileExists || parent == nil || child == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if !parent.checkPermission(avfs.PermWrite, vfs.user) {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
	}

	child.Lock()
	defer child.Unlock()

	if c, ok := child.(*dirNode); ok {
		if len(c.children) != 0 {
			return &fs.PathError{Op: op, Path: name, Err: avfs.ErrDirNotEmpty}
		}
	}

	part := absPath[start:end]
	if parent.child(part) == nil {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
	}

	parent.removeChild(part)
	child.delete()

	return nil
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) RemoveAll(path string) error {
	const op = "unlinkat"

	if path == "" {
		// fail silently to retain compatibility with previous behavior of RemoveAll.
		return nil
	}

	parent, child, absPath, start, end, err := vfs.searchNode(path, slmLstat)
	if vfs.IsNotExist(err) || parent == nil {
		return nil
	}

	if err != avfs.ErrFileExists {
		return &fs.PathError{Op: op, Path: path, Err: err}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	ok := parent.checkPermission(avfs.PermWrite|avfs.PermLookup, vfs.user)
	if !ok {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrPermDenied}
	}

	defer child.delete()

	if c, ok := child.(*dirNode); ok {
		err := vfs.removeAll(c)
		if err != nil {
			return &fs.PathError{Op: op, Path: path, Err: avfs.ErrPermDenied}
		}
	}

	part := absPath[start:end]
	parent.removeChild(part)

	return nil
}

func (vfs *MemFS) removeAll(dn *dirNode) error {
	dn.mu.Lock()
	defer dn.mu.Unlock()

	ok := dn.checkPermission(avfs.PermWrite|avfs.PermLookup, vfs.user)
	if !ok {
		return avfs.ErrPermDenied
	}

	for _, child := range dn.children {
		if c, ok := child.(*dirNode); ok {
			err := vfs.removeAll(c)
			if err != nil {
				return err
			}
		}

		child.delete()
	}

	return nil
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *MemFS) Rename(oldpath, newpath string) error {
	const op = "rename"

	oParent, oChild, oAbsPath, oStart, oEnd, oErr := vfs.searchNode(oldpath, slmLstat)
	if oErr != avfs.ErrFileExists || oParent == nil {
		return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: oErr}
	}

	nParent, nChild, nAbsPath, nStart, nEnd, nErr := vfs.searchNode(newpath, slmLstat)
	if !(nErr == avfs.ErrFileExists || nErr == avfs.ErrNoSuchFileOrDir) || nParent == nil {
		return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: nErr}
	}

	oParent.mu.Lock()
	defer oParent.mu.Unlock()

	if nParent != oParent {
		nParent.mu.Lock()
		defer nParent.mu.Unlock()
	}

	if !oParent.checkPermission(avfs.PermWrite, vfs.user) {
		return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: avfs.ErrPermDenied}
	}

	if !nParent.checkPermission(avfs.PermWrite, vfs.user) {
		return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: avfs.ErrPermDenied}
	}

	if oAbsPath == nAbsPath {
		return nil
	}

	nPart := nAbsPath[nStart:nEnd]
	oPart := oAbsPath[oStart:oEnd]

	switch oChild.(type) {
	case *dirNode:
		if nErr != avfs.ErrNoSuchFileOrDir {
			return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: nErr}
		}

	case *fileNode:
		if nChild == nil {
			break
		}

		switch nc := nChild.(type) {
		case *fileNode:
			nc.delete()
		default:
			return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: avfs.ErrFileExists}
		}
	default:
		return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: avfs.ErrPermDenied}
	}

	nParent.addChild(nPart, oChild)
	oParent.removeChild(oPart)

	return nil
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (vfs *MemFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	fs1, ok1 := fi1.(*MemInfo)
	if !ok1 {
		return false
	}

	fs2, ok2 := fi2.(*MemInfo)
	if !ok2 {
		return false
	}

	return fs1.id == fs2.id
}

// SetUMask sets the file mode creation mask.
func (vfs *MemFS) SetUMask(mask fs.FileMode) {
	atomic.StoreInt32(&vfs.memAttrs.umask, int32(mask))
}

// SetUser sets and returns the current user.
// If the user is not found, the returned error is of type UnknownUserError.
func (vfs *MemFS) SetUser(name string) (avfs.UserReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	if vfs.user.Name() == name {
		return vfs.user, nil
	}

	user, err := vfs.memAttrs.idm.LookupUser(name)
	if err != nil {
		return nil, err
	}

	vfs.user = user
	vfs.curDir = vfs.utils.HomeDirUser(user.Name())

	return user, nil
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *MemFS) Split(path string) (dir, file string) {
	return vfs.utils.Split(path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Stat(path string) (fs.FileInfo, error) {
	const op = "stat"

	_, child, absPath, start, end, err := vfs.searchNode(path, slmStat)
	if err != avfs.ErrFileExists || child == nil {
		return nil, &fs.PathError{Op: op, Path: path, Err: err}
	}

	part := absPath[start:end]
	fst := child.fillStatFrom(part)

	return fst, nil
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *MemFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	parent, _, absPath, start, end, nerr := vfs.searchNode(newname, slmLstat)
	if nerr != avfs.ErrNoSuchFileOrDir || parent == nil {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: nerr}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if !parent.checkPermission(avfs.PermWrite, vfs.user) {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrPermDenied}
	}

	link := vfs.Clean(oldname)
	part := absPath[start:end]

	vfs.createSymlink(parent, part, link)

	return nil
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
func (vfs *MemFS) TempDir() string {
	return vfs.utils.TempDir(vfs)
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *MemFS) ToSlash(path string) string {
	return vfs.utils.ToSlash(path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *MemFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return info.Sys().(avfs.SysStater)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Truncate(name string, size int64) error {
	const op = "truncate"

	parent, child, _, _, _, err := vfs.searchNode(name, slmEval)
	if err != avfs.ErrFileExists || parent == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	c, ok := child.(*fileNode)
	if !ok {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrIsADirectory}
	}

	if size < 0 {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrInvalidArgument}
	}

	c.mu.Lock()
	c.truncate(size)
	c.mu.Unlock()

	return nil
}

// UMask returns the file mode creation mask.
func (vfs *MemFS) UMask() fs.FileMode {
	u := atomic.LoadInt32(&vfs.memAttrs.umask)

	return fs.FileMode(u)
}

// User returns the current user.
// if the file system does not have a current user, the user avfs.NotImplementedUser is returned.
func (vfs *MemFS) User() avfs.UserReader {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return avfs.NotImplementedUser
	}

	return vfs.user
}

// Utils returns the file utils of the current file system.
func (vfs *MemFS) Utils() avfs.Utils {
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
func (vfs *MemFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return vfs.utils.WalkDir(vfs, root, fn)
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
func (vfs *MemFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return vfs.utils.WriteFile(vfs, name, data, perm)
}
