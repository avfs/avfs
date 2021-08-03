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
	"path/filepath"
	"strings"
	"sync/atomic"
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
func (vfs *OrefaFS) Abs(path string) (string, error) {
	return vfsutils.Abs(vfs, path)
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *OrefaFS) Base(path string) string {
	return vfsutils.Base(path)
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
		return &fs.PathError{Op: op, Path: dir, Err: avfs.ErrNoSuchFileOrDir}
	}

	if !nd.mode.IsDir() {
		return &fs.PathError{Op: op, Path: dir, Err: avfs.ErrNotADirectory}
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
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
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

	absPath, _ := vfs.Abs(name)

	vfs.mu.RLock()
	nd, ok := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
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

	return &fs.PathError{Op: op, Path: path, Err: avfs.ErrOpNotPermitted}
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
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
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
	return vfsutils.Clean(path)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Create(name string) (avfs.File, error) {
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
func (vfs *OrefaFS) CreateTemp(dir, pattern string) (avfs.File, error) {
	return vfsutils.CreateTemp(vfs, dir, pattern)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *OrefaFS) Dir(path string) string {
	return vfsutils.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *OrefaFS) EvalSymlinks(path string) (string, error) {
	const op = "lstat"

	return "", &fs.PathError{Op: op, Path: path, Err: avfs.ErrPermDenied}
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *OrefaFS) FromSlash(path string) string {
	return path
}

// GetUMask returns the file mode creation mask.
func (vfs *OrefaFS) GetUMask() fs.FileMode {
	u := atomic.LoadInt32(&vfs.umask)

	return fs.FileMode(u)
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
	return vfsutils.Glob(vfs, pattern)
}

// IsAbs reports whether the path is absolute.
func (vfs *OrefaFS) IsAbs(path string) bool {
	return vfsutils.IsAbs(path)
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (vfs *OrefaFS) IsExist(err error) bool {
	return vfsutils.IsExist(err)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (vfs *OrefaFS) IsNotExist(err error) bool {
	return vfsutils.IsNotExist(err)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *OrefaFS) IsPathSeparator(c uint8) bool {
	return vfsutils.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (vfs *OrefaFS) Join(elem ...string) string {
	return vfsutils.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *OrefaFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	absPath, _ := vfs.Abs(name)

	vfs.mu.RLock()
	nd, ok := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
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

	nDirName, nFileName := split(nAbsPath)

	vfs.mu.RLock()
	oChild, oChildOk := vfs.nodes[oAbsPath]
	_, nChildOk := vfs.nodes[nAbsPath]
	nParent, nParentOk := vfs.nodes[nDirName]
	vfs.mu.RUnlock()

	if !oChildOk || !nParentOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrNoSuchFileOrDir}
	}

	oChild.mu.Lock()
	defer oChild.mu.Unlock()

	nParent.mu.Lock()
	defer nParent.mu.Unlock()

	if oChild.mode.IsDir() {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrOpNotPermitted}
	}

	if nChildOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrFileExists}
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
	const op = "lstat"

	return vfs.stat(name, op)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Mkdir(name string, perm fs.FileMode) error {
	const op = "mkdir"

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: avfs.ErrNoSuchFileOrDir}
	}

	absPath, _ := vfs.Abs(name)
	dirName, fileName := split(absPath)

	vfs.mu.RLock()
	_, childOk := vfs.nodes[absPath]
	parent, parentOk := vfs.nodes[dirName]
	vfs.mu.RUnlock()

	if childOk {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrFileExists}
	}

	if !parentOk {
		for !parentOk {
			dirName, _ = split(dirName)

			vfs.mu.RLock()
			parent, parentOk = vfs.nodes[dirName]
			vfs.mu.RUnlock()
		}

		if parent.mode.IsDir() {
			return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
		}

		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNotADirectory}
	}

	if !parent.mode.IsDir() {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNotADirectory}
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

	vfs.mu.RLock()
	child, childOk := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if childOk {
		if child.mode.IsDir() {
			return nil
		}

		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrNotADirectory}
	}

	var (
		ds     []string
		parent *node
	)

	dirName := absPath

	for {
		vfs.mu.RLock()
		nd, ok := vfs.nodes[dirName]
		vfs.mu.RUnlock()

		if ok {
			parent = nd
			if !parent.mode.IsDir() {
				return &fs.PathError{Op: op, Path: dirName, Err: avfs.ErrNotADirectory}
			}

			break
		}

		ds = append(ds, dirName)

		dirName, _ = split(dirName)
	}

	for _, absPath := range ds {
		_, fileName := split(absPath)

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
func (vfs *OrefaFS) MkdirTemp(dir, prefix string) (name string, err error) {
	return vfsutils.MkdirTemp(vfs, dir, prefix)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Open(path string) (avfs.File, error) {
	return vfs.OpenFile(path, os.O_RDONLY, 0)
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
	dirName, fileName := split(absPath)

	vfs.mu.RLock()
	child, childOk := vfs.nodes[absPath]
	parent, parentOk := vfs.nodes[dirName]
	vfs.mu.RUnlock()

	if !childOk {
		if !parentOk {
			return nil, &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
		}

		if flag&os.O_CREATE == 0 {
			if !parent.mode.IsDir() {
				return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: avfs.ErrNotADirectory}
			}

			return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
		}

		if !parent.mode.IsDir() {
			return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: avfs.ErrNotADirectory}
		}

		if pm&avfs.PermWrite == 0 {
			return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
		}

		child = vfs.createFile(parent, absPath, fileName, perm)
	} else {
		if child.mode.IsDir() {
			if pm&avfs.PermWrite != 0 {
				return nilFile, &fs.PathError{Op: op, Path: name, Err: avfs.ErrIsADirectory}
			}
		} else {
			if flag&(os.O_CREATE|os.O_EXCL) == os.O_CREATE|os.O_EXCL {
				return &OrefaFile{}, &fs.PathError{Op: op, Path: name, Err: avfs.ErrFileExists}
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
		orFS:     vfs,
		nd:       child,
		permMode: pm,
		name:     name,
		at:       at,
	}

	return f, nil
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *OrefaFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return vfsutils.ReadDir(vfs, name)
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func (vfs *OrefaFS) ReadFile(filename string) ([]byte, error) {
	return vfsutils.ReadFile(vfs, filename)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Readlink(name string) (string, error) {
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
func (vfs *OrefaFS) Rel(basepath, targpath string) (string, error) {
	return vfsutils.Rel(basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Remove(name string) error {
	const op = "remove"

	absPath, _ := vfs.Abs(name)
	dirName, fileName := split(absPath)

	vfs.mu.RLock()
	child, childOk := vfs.nodes[absPath]
	parent, parentOk := vfs.nodes[dirName]
	vfs.mu.RUnlock()

	if !childOk || !parentOk {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	child.mu.Lock()
	defer child.mu.Unlock()

	if child.mode.IsDir() && len(child.children) != 0 {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrDirNotEmpty}
	}

	delete(parent.children, fileName)

	child.remove()

	vfs.mu.Lock()
	delete(vfs.nodes, absPath)
	vfs.mu.Unlock()

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
	dirName, fileName := split(absPath)

	vfs.mu.RLock()
	child, childOk := vfs.nodes[absPath]
	parent, parentOk := vfs.nodes[dirName]
	vfs.mu.RUnlock()

	if !childOk || !parentOk {
		return nil
	}

	if child.mode.IsDir() {
		vfs.removeAll(absPath, child)
	}

	vfs.mu.Lock()
	delete(vfs.nodes, absPath)
	vfs.mu.Unlock()

	child.remove()

	parent.mu.Lock()
	delete(parent.children, fileName)
	parent.mu.Unlock()

	return nil
}

func (vfs *OrefaFS) removeAll(absPath string, rootNode *node) {
	if rootNode.mode.IsDir() {
		for fileName, nd := range rootNode.children {
			path := absPath + string(avfs.PathSeparator) + fileName

			vfs.removeAll(path, nd)
		}
	}

	vfs.mu.Lock()
	delete(vfs.nodes, absPath)
	vfs.mu.Unlock()

	rootNode.remove()
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

	oDirName, oFileName := split(oAbsPath)
	nDirName, nFileName := split(nAbsPath)

	vfs.mu.RLock()
	oChild, oChildOk := vfs.nodes[oAbsPath]
	oParent, oParentOk := vfs.nodes[oDirName]
	nChild, nChildOk := vfs.nodes[nAbsPath]
	nParent, nParentOk := vfs.nodes[nDirName]
	vfs.mu.RUnlock()

	if !oChildOk || !oParentOk || !nParentOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrNoSuchFileOrDir}
	}

	if oChild.mode.IsDir() && nChildOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrFileExists}
	}

	if !oChild.mode.IsDir() && nChildOk && nChild.mode.IsDir() {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrFileExists}
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

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *OrefaFS) Split(path string) (dir, file string) {
	return vfsutils.Split(vfs, path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Stat(path string) (fs.FileInfo, error) {
	const op = "stat"

	return vfs.stat(path, op)
}

// stat is the internal function used by Stat and Lstat.
func (vfs *OrefaFS) stat(path, op string) (fs.FileInfo, error) {
	absPath, _ := vfs.Abs(path)
	dirName, fileName := split(absPath)

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
					return nil, &fs.PathError{Op: op, Path: path, Err: avfs.ErrNoSuchFileOrDir}
				}

				return nil, &fs.PathError{Op: op, Path: path, Err: avfs.ErrNotADirectory}
			}

			dirName, _ = split(dirName)
		}
	}

	fst := child.fillStatFrom(fileName)

	return fst, nil
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *OrefaFS) Symlink(oldname, newname string) error {
	const op = "symlink"

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
func (vfs *OrefaFS) TempDir() string {
	return avfs.TmpDir
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *OrefaFS) ToSlash(path string) string {
	return path
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *OrefaFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return info.Sys().(avfs.SysStater)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *OrefaFS) Truncate(name string, size int64) error {
	const op = "truncate"

	absPath, _ := vfs.Abs(name)

	vfs.mu.RLock()
	child, childOk := vfs.nodes[absPath]
	vfs.mu.RUnlock()

	if !childOk {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
	}

	if child.mode.IsDir() {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrIsADirectory}
	}

	if size < 0 {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrInvalidArgument}
	}

	child.mu.Lock()
	child.truncate(size)
	child.mu.Unlock()

	return nil
}

// UMask sets the file mode creation mask.
func (vfs *OrefaFS) UMask(mask fs.FileMode) {
	atomic.StoreInt32(&vfs.umask, int32(mask))
}

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.
func (vfs *OrefaFS) Walk(root string, walkFn filepath.WalkFunc) error {
	return vfsutils.Walk(vfs, root, walkFn)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func (vfs *OrefaFS) WriteFile(filename string, data []byte, perm fs.FileMode) error {
	return vfsutils.WriteFile(vfs, filename, data, perm)
}
