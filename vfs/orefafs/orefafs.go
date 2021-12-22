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

// Package orefafs implements an Afero like in memory file system.
package orefafs

import (
	"io/fs"
	"os"
	"strings"
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
func (vfs *OrefaFS) Abs(path string) (string, error) {
	return vfs.utils.Abs(vfs, path)
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *OrefaFS) Base(path string) string {
	return vfs.utils.Base(path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Chdir(dir string) error {
	const op = "chdir"

	absPath, _ := vfs.Abs(dir)

	vfs.mu.RLock()
	nd, ok := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: dir, Err: vfs.err.NoSuchFile}
	}

	if !nd.mode.IsDir() {
		err := vfs.err.NotADirectory
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinDirNameInvalid
		}

		return &fs.PathError{Op: op, Path: dir, Err: err}
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
func (vfs *OrefaFS) Chmod(name string, mode fs.FileMode) error {
	const op = "chmod"

	absPath, _ := vfs.Abs(name)

	vfs.mu.RLock()
	nd, ok := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
	}

	nd.mu.Lock()
	nd.setMode(mode)
	nd.mu.Unlock()

	return nil
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *OrefaFS) Chown(name string, uid, gid int) error {
	const op = "chown"

	if !vfs.HasFeature(avfs.FeatChownUser) {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.OpNotPermitted}
	}

	absPath, _ := vfs.Abs(name)

	vfs.mu.RLock()
	nd, ok := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
	}

	nd.mu.Lock()
	nd.setOwner(uid, gid)
	nd.mu.Unlock()

	return nil
}

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Chroot(path string) error {
	const op = "chroot"

	return &fs.PathError{Op: op, Path: path, Err: vfs.err.OpNotPermitted}
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Chtimes(name string, atime, mtime time.Time) error {
	const op = "chtimes"

	absPath, _ := vfs.Abs(name)

	vfs.mu.RLock()
	nd, ok := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
	}

	nd.mu.Lock()
	nd.setModTime(mtime)
	nd.mu.Unlock()

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
func (vfs *OrefaFS) Clean(path string) string {
	return vfs.utils.Clean(path)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Create(name string) (avfs.File, error) {
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
func (vfs *OrefaFS) CreateTemp(dir, pattern string) (avfs.File, error) {
	return vfs.utils.CreateTemp(vfs, dir, pattern)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *OrefaFS) Dir(path string) string {
	return vfs.utils.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *OrefaFS) EvalSymlinks(path string) (string, error) {
	op := "lstat"
	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
	}

	return "", &fs.PathError{Op: op, Path: path, Err: vfs.err.PermDenied}
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *OrefaFS) FromSlash(path string) string {
	return vfs.utils.FromSlash(path)
}

// Getwd returns a rooted name link corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *OrefaFS) Getwd() (dir string, err error) {
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
func (vfs *OrefaFS) Glob(pattern string) (matches []string, err error) {
	return vfs.utils.Glob(vfs, pattern)
}

// Idm returns the identity manager of the file system.
// If the file system does not have an identity manager, avfs.DummyIdm is returned.
func (vfs *OrefaFS) Idm() avfs.IdentityMgr {
	return avfs.NotImplementedIdm
}

// IsAbs reports whether the path is absolute.
func (vfs *OrefaFS) IsAbs(path string) bool {
	return vfs.utils.IsAbs(path)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *OrefaFS) IsPathSeparator(c uint8) bool {
	return vfs.utils.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (vfs *OrefaFS) Join(elem ...string) string {
	return vfs.utils.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *OrefaFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	if !vfs.HasFeature(avfs.FeatChownUser) {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.OpNotPermitted}
	}

	absPath, _ := vfs.Abs(name)

	vfs.mu.RLock()
	nd, ok := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
	}

	nd.mu.Lock()
	nd.setOwner(uid, gid)
	nd.mu.Unlock()

	return nil
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *OrefaFS) Link(oldname, newname string) error {
	const op = "link"

	oAbsPath, _ := vfs.Abs(oldname)
	nAbsPath, _ := vfs.Abs(newname)

	nDirName, nFileName := vfs.splitPath(nAbsPath)

	vfs.mu.RLock()
	oChild, oChildOk := vfs.nodes[oAbsPath]
	_, nChildOk := vfs.nodes[nAbsPath]
	nParent, nParentOk := vfs.nodes[nDirName]
	vfs.mu.RUnlock()

	if !oChildOk {
		err := vfs.err.NoSuchFile

		if vfs.OSType() == avfs.OsWindows {
			oDirName, _ := vfs.splitPath(oAbsPath)

			vfs.mu.RLock()
			_, oParentOk := vfs.nodes[oDirName]
			vfs.mu.RUnlock()

			if !oParentOk {
				err = vfs.err.NoSuchDir
			}
		}

		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: err}
	}

	if !nParentOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.NoSuchFile}
	}

	oChild.mu.Lock()
	defer oChild.mu.Unlock()

	nParent.mu.Lock()
	defer nParent.mu.Unlock()

	if oChild.mode.IsDir() {
		err := error(avfs.ErrOpNotPermitted)
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinAccessDenied
		}

		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: err}
	}

	if nChildOk {
		err := vfs.err.FileExists
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinAlreadyExists
		}

		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: err}
	}

	vfs.mu.Lock()
	vfs.nodes[nAbsPath] = oChild
	vfs.mu.Unlock()

	nParent.addChild(nFileName, oChild)
	oChild.nlink++

	return nil
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Lstat(name string) (fs.FileInfo, error) {
	op := "lstat"
	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
	}

	return vfs.stat(name, op)
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
func (vfs *OrefaFS) Match(pattern, name string) (matched bool, err error) {
	return vfs.utils.Match(pattern, name)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Mkdir(name string, perm fs.FileMode) error {
	const op = "mkdir"

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchDir}
	}

	absPath, _ := vfs.Abs(name)
	dirName, fileName := vfs.splitPath(absPath)

	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	_, childOk := vfs.nodes[absPath]
	parent, parentOk := vfs.nodes[dirName]

	if childOk {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.FileExists}
	}

	if !parentOk {
		for !parentOk {
			dirName, _ = vfs.splitPath(dirName)
			parent, parentOk = vfs.nodes[dirName]
		}

		if parent.mode.IsDir() {
			return &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchDir}
		}

		return &fs.PathError{Op: op, Path: name, Err: vfs.err.NotADirectory}
	}

	if !parent.mode.IsDir() {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.NotADirectory}
	}

	vfs.createDir(parent, absPath, fileName, perm)

	return nil
}

// MkdirAll creates a directory named name,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If name is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *OrefaFS) MkdirAll(path string, perm fs.FileMode) error {
	const op = "mkdir"

	absPath, _ := vfs.Abs(path)

	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	child, childOk := vfs.nodes[absPath]
	if childOk {
		if child.mode.IsDir() {
			return nil
		}

		return &fs.PathError{Op: op, Path: path, Err: vfs.err.NotADirectory}
	}

	var (
		ds     []string
		parent *node
	)

	dirName := absPath

	for {
		nd, ok := vfs.nodes[dirName]
		if ok {
			parent = nd
			if !parent.mode.IsDir() {
				return &fs.PathError{Op: op, Path: dirName, Err: vfs.err.NotADirectory}
			}

			break
		}

		ds = append(ds, dirName)

		dirName, _ = vfs.splitPath(dirName)
	}

	for _, absPath := range ds {
		_, fileName := vfs.splitPath(absPath)

		parent = vfs.createDir(parent, absPath, fileName, perm)
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
func (vfs *OrefaFS) MkdirTemp(dir, pattern string) (string, error) {
	return vfs.utils.MkdirTemp(vfs, dir, pattern)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Open(name string) (avfs.File, error) {
	return vfs.utils.Open(vfs, name)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) OpenFile(name string, flag int, perm fs.FileMode) (avfs.File, error) {
	const op = "open"

	var (
		at      int64
		pm      avfs.PermMode
		nilFile *OrefaFile
	)

	if flag == os.O_RDONLY || flag&os.O_RDWR != 0 {
		pm = avfs.PermRead
	}

	if flag&(os.O_APPEND|os.O_CREATE|os.O_RDWR|os.O_TRUNC|os.O_WRONLY) != 0 {
		pm |= avfs.PermWrite
	}

	absPath, _ := vfs.Abs(name)
	dirName, fileName := vfs.splitPath(absPath)

	vfs.mu.RLock()
	parent, parentOk := vfs.nodes[dirName]
	child, childOk := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !childOk {
		if !parentOk {
			return nil, &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchDir}
		}

		if flag&os.O_CREATE == 0 {
			if !parent.mode.IsDir() {
				return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: vfs.err.NotADirectory}
			}

			return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
		}

		if !parent.mode.IsDir() {
			return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: vfs.err.NotADirectory}
		}

		if pm&avfs.PermWrite == 0 {
			return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: vfs.err.PermDenied}
		}

		vfs.mu.Lock()
		defer vfs.mu.Unlock()

		// test for race conditions when opening file in exclusive mode.
		_, childOk = vfs.nodes[absPath]
		if childOk && flag&(os.O_CREATE|os.O_EXCL) == os.O_CREATE|os.O_EXCL {
			return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: vfs.err.FileExists}
		}

		child = vfs.createFile(parent, absPath, fileName, perm)
	} else {
		if child.mode.IsDir() {
			if pm&avfs.PermWrite != 0 {
				return nilFile, &fs.PathError{Op: op, Path: name, Err: vfs.err.IsADirectory}
			}
		} else {
			if flag&(os.O_CREATE|os.O_EXCL) == os.O_CREATE|os.O_EXCL {
				return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: vfs.err.FileExists}
			}

			if flag&os.O_TRUNC != 0 {
				child.mu.Lock()
				child.truncate(0)
				child.mu.Unlock()
			}

			if flag&os.O_APPEND != 0 {
				at = child.Size()
			}
		}
	}

	f := &OrefaFile{
		vfs:      vfs,
		nd:       child,
		permMode: pm,
		name:     name,
		at:       at,
	}

	return f, nil
}

// PathSeparator return the OS-specific path separator.
func (vfs *OrefaFS) PathSeparator() uint8 {
	return vfs.utils.PathSeparator()
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *OrefaFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return vfs.utils.ReadDir(vfs, name)
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
func (vfs *OrefaFS) ReadFile(name string) ([]byte, error) {
	return vfs.utils.ReadFile(vfs, name)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Readlink(name string) (string, error) {
	const op = "readlink"

	err := error(avfs.ErrPermDenied)
	if vfs.OSType() == avfs.OsWindows {
		err = avfs.ErrWinNotReparsePoint
	}

	return "", &fs.PathError{Op: op, Path: name, Err: err}
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (vfs *OrefaFS) Rel(basepath, targpath string) (string, error) {
	return vfs.utils.Rel(basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Remove(name string) error {
	const op = "remove"

	absPath, _ := vfs.Abs(name)
	dirName, fileName := vfs.splitPath(absPath)

	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	child, childOk := vfs.nodes[absPath]
	parent, parentOk := vfs.nodes[dirName]

	if !childOk || !parentOk {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	child.mu.Lock()
	defer child.mu.Unlock()

	if child.mode.IsDir() && len(child.children) != 0 {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.DirNotEmpty}
	}

	child.remove()

	delete(parent.children, fileName)
	delete(vfs.nodes, absPath)

	return nil
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) RemoveAll(path string) error {
	if path == "" {
		// fail silently to retain compatibility with previous behavior of RemoveAll.
		return nil
	}

	absPath, _ := vfs.Abs(path)
	dirName, fileName := vfs.splitPath(absPath)

	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	child, childOk := vfs.nodes[absPath]
	parent, parentOk := vfs.nodes[dirName]

	if !childOk || !parentOk {
		return nil
	}

	if child.mode.IsDir() {
		vfs.removeAll(absPath, child)
	}

	child.remove()

	delete(parent.children, fileName)
	delete(vfs.nodes, absPath)

	return nil
}

func (vfs *OrefaFS) removeAll(absPath string, rootNode *node) {
	if rootNode.mode.IsDir() {
		for fileName, nd := range rootNode.children {
			path := absPath + string(avfs.PathSeparator) + fileName

			vfs.removeAll(path, nd)
		}
	}

	rootNode.remove()
	delete(vfs.nodes, absPath)
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *OrefaFS) Rename(oldname, newname string) error {
	const op = "rename"

	oAbsPath, _ := vfs.Abs(oldname)
	nAbsPath, _ := vfs.Abs(newname)

	if oAbsPath == nAbsPath {
		return nil
	}

	oDirName, oFileName := vfs.splitPath(oAbsPath)
	nDirName, nFileName := vfs.splitPath(nAbsPath)

	vfs.mu.RLock()
	oChild, oChildOk := vfs.nodes[oAbsPath]
	oParent, oParentOk := vfs.nodes[oDirName]
	nChild, nChildOk := vfs.nodes[nAbsPath]
	nParent, nParentOk := vfs.nodes[nDirName]
	vfs.mu.RUnlock()

	if !oChildOk || !oParentOk || !nParentOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.NoSuchFile}
	}

	if oChild.mode.IsDir() && nChildOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.FileExists}
	}

	if !oChild.mode.IsDir() && nChildOk && nChild.mode.IsDir() {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.FileExists}
	}

	nParent.mu.Lock()

	if nParent != oParent {
		oParent.mu.Lock()
	}

	nParent.children[nFileName] = oChild

	delete(oParent.children, oFileName)

	vfs.mu.Lock()

	vfs.nodes[nAbsPath] = oChild
	delete(vfs.nodes, oAbsPath)

	if oChild.mode.IsDir() {
		oRoot := oAbsPath + string(avfs.PathSeparator)

		for absPath, node := range vfs.nodes {
			if strings.HasPrefix(absPath, oRoot) {
				nPath := nAbsPath + absPath[len(oAbsPath):]
				vfs.nodes[nPath] = node

				delete(vfs.nodes, absPath)
			}
		}
	}

	vfs.mu.Unlock()

	if nParent != oParent {
		oParent.mu.Unlock()
	}

	nParent.mu.Unlock()

	return nil
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (vfs *OrefaFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	fs1, ok1 := fi1.(*OrefaInfo)
	if !ok1 {
		return false
	}

	fs2, ok2 := fi2.(*OrefaInfo)
	if !ok2 {
		return false
	}

	return fs1.id == fs2.id
}

// SetUMask sets the file mode creation mask.
func (vfs *OrefaFS) SetUMask(mask fs.FileMode) {
	atomic.StoreUint32((*uint32)(&vfs.umask), uint32(mask))
}

// SetUser sets and returns the current user.
// If the user is not found, the returned error is of type UnknownUserError.
func (vfs *OrefaFS) SetUser(name string) (avfs.UserReader, error) {
	return nil, vfs.err.PermDenied
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *OrefaFS) Split(path string) (dir, file string) {
	return vfs.utils.Split(path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Stat(path string) (fs.FileInfo, error) {
	op := "stat"
	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
	}

	return vfs.stat(path, op)
}

// stat is the internal function used by Stat and Lstat.
func (vfs *OrefaFS) stat(path, op string) (fs.FileInfo, error) {
	absPath, _ := vfs.Abs(path)
	dirName, fileName := vfs.splitPath(absPath)

	vfs.mu.RLock()
	child, childOk := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !childOk {
		for {
			vfs.mu.RLock()
			parent, parentOk := vfs.nodes[dirName]
			vfs.mu.RUnlock()

			if parentOk {
				if parent.mode.IsDir() {
					return nil, &fs.PathError{Op: op, Path: path, Err: vfs.err.NoSuchFile}
				}

				return nil, &fs.PathError{Op: op, Path: path, Err: vfs.err.NotADirectory}
			}

			dirName, _ = vfs.splitPath(dirName)
		}
	}

	fst := child.fillStatFrom(fileName)

	return fst, nil
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *OrefaFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	err := error(avfs.ErrPermDenied)
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
func (vfs *OrefaFS) TempDir() string {
	return vfs.utils.TempDir(vfs.user.Name())
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *OrefaFS) ToSlash(path string) string {
	return vfs.utils.ToSlash(path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *OrefaFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return info.Sys().(avfs.SysStater)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Truncate(name string, size int64) error {
	op := "truncate"

	absPath, _ := vfs.Abs(name)

	vfs.mu.RLock()
	child, childOk := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !childOk {
		if vfs.OSType() != avfs.OsWindows {
			return &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
		}

		return nil
	}

	if child.mode.IsDir() {
		if vfs.OSType() == avfs.OsWindows {
			op = "open"
		}

		return &fs.PathError{Op: op, Path: name, Err: vfs.err.IsADirectory}
	}

	if size < 0 {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.InvalidArgument}
	}

	child.mu.Lock()
	child.truncate(size)
	child.mu.Unlock()

	return nil
}

// UMask returns the file mode creation mask.
func (vfs *OrefaFS) UMask() fs.FileMode {
	u := atomic.LoadUint32((*uint32)(&vfs.umask))

	return fs.FileMode(u)
}

// User returns the current user.
// If the file system does not have a current user, the user avfs.DefaultUser is returned.
func (vfs *OrefaFS) User() avfs.UserReader {
	return vfs.user
}

// Utils returns the file utils of the current file system.
func (vfs *OrefaFS) Utils() avfs.Utils {
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
func (vfs *OrefaFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return vfs.utils.WalkDir(vfs, root, fn)
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
func (vfs *OrefaFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return vfs.utils.WriteFile(vfs, name, data, perm)
}
