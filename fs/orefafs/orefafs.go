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
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
)

// file system functions.

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (fs *OrefaFs) Abs(path string) (string, error) {
	return fsutil.Abs(fs, path)
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (fs *OrefaFs) Base(path string) string {
	return fsutil.Base(path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Chdir(dir string) error {
	const op = "chdir"

	absPath, _ := fs.Abs(dir)

	nd, ok := fs.nodes[absPath]
	if !ok {
		return &os.PathError{Op: op, Path: dir, Err: avfs.ErrNoSuchFileOrDir}
	}

	if !nd.mode.IsDir() {
		return &os.PathError{Op: op, Path: dir, Err: avfs.ErrNotADirectory}
	}

	fs.curDir = absPath

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
// On Windows, the mode must be non-zero but otherwise only the 0200
// bit (owner writable) of mode is used; it controls whether the
// file's read-only attribute is set or cleared. attribute. The other
// bits are currently unused. Use mode 0400 for a read-only file and
// 0600 for a readable+writable file.
//
// On Plan 9, the mode's permission bits, ModeAppend, ModeExclusive,
// and ModeTemporary are used.
func (fs *OrefaFs) Chmod(name string, mode os.FileMode) error {
	const op = "chmod"

	absPath, _ := fs.Abs(name)

	nd, ok := fs.nodes[absPath]
	if !ok {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
	}

	nd.setMode(mode)

	return nil
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (fs *OrefaFs) Chown(name string, uid, gid int) error {
	const op = "chown"

	return &os.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
}

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Chroot(path string) error {
	const op = "chroot"

	return &os.PathError{Op: op, Path: path, Err: avfs.ErrPermDenied}
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Chtimes(name string, atime, mtime time.Time) error {
	const op = "chtimes"

	absPath, _ := fs.Abs(name)

	nd, ok := fs.nodes[absPath]
	if !ok {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
	}

	nd.setModTime(mtime)

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
func (fs *OrefaFs) Clean(path string) string {
	return fsutil.Clean(path)
}

// Clone returns the file system itself since if does not support this feature (FeatClonable).
func (fs *OrefaFs) Clone() avfs.Fs {
	return fs
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Create(name string) (avfs.File, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (fs *OrefaFs) Dir(path string) string {
	return fsutil.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (fs *OrefaFs) EvalSymlinks(path string) (string, error) {
	const op = "lstat"
	return "", &os.PathError{Op: op, Path: path, Err: avfs.ErrPermDenied}
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (fs *OrefaFs) FromSlash(path string) string {
	return path
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
func (fs *OrefaFs) GetTempDir() string {
	return avfs.TmpDir
}

// GetUMask returns the file mode creation mask.
func (fs *OrefaFs) GetUMask() os.FileMode {
	u := atomic.LoadInt32(&fs.umask)

	return os.FileMode(u)
}

// Getwd returns a rooted name link corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (fs *OrefaFs) Getwd() (dir string, err error) {
	dir = fs.curDir

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
func (fs *OrefaFs) Glob(pattern string) (matches []string, err error) {
	return fsutil.Glob(fs, pattern)
}

// IsAbs reports whether the path is absolute.
func (fs *OrefaFs) IsAbs(path string) bool {
	return fsutil.IsAbs(path)
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (fs *OrefaFs) IsExist(err error) bool {
	return fsutil.IsExist(err)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (fs *OrefaFs) IsNotExist(err error) bool {
	return fsutil.IsNotExist(err)
}

// IsPathSeparator reports whether c is a directory separator character.
func (fs *OrefaFs) IsPathSeparator(c uint8) bool {
	return fsutil.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (fs *OrefaFs) Join(elem ...string) string {
	return fsutil.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (fs *OrefaFs) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	return &os.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (fs *OrefaFs) Link(oldname, newname string) error {
	const op = "link"

	oAbsPath, _ := fs.Abs(oldname)
	nAbsPath, _ := fs.Abs(newname)

	nDirName, nFileName := split(nAbsPath)

	fs.mu.RLock()
	oChild, oChildOk := fs.nodes[oAbsPath]
	_, nChildOk := fs.nodes[nAbsPath]
	nParent, nParentOk := fs.nodes[nDirName]
	fs.mu.RUnlock()

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

	fs.mu.Lock()
	fs.nodes[nAbsPath] = oChild
	fs.mu.Unlock()

	nParent.addChild(nFileName, oChild)
	oChild.nlink++

	return nil
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Lstat(name string) (os.FileInfo, error) {
	const op = "lstat"

	return fs.stat(name, op)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Mkdir(name string, perm os.FileMode) error {
	const op = "mkdir"

	if name == "" {
		return &os.PathError{Op: op, Path: "", Err: avfs.ErrNoSuchFileOrDir}
	}

	absPath, _ := fs.Abs(name)
	dirName, fileName := split(absPath)

	fs.mu.RLock()
	_, childOk := fs.nodes[absPath]
	parent, parentOk := fs.nodes[dirName]
	fs.mu.RUnlock()

	if childOk {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrFileExists}
	}

	if !parentOk {
		for !parentOk {
			dirName, _ = split(dirName)

			fs.mu.RLock()
			parent, parentOk = fs.nodes[dirName]
			fs.mu.RUnlock()
		}

		if parent.mode.IsDir() {
			return &os.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
		}

		return &os.PathError{Op: op, Path: name, Err: avfs.ErrNotADirectory}
	}

	fs.createDir(parent, absPath, fileName, perm)

	return nil
}

// MkdirAll creates a directory named name,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If name is already a directory, MkdirAll does nothing
// and returns nil.
func (fs *OrefaFs) MkdirAll(path string, perm os.FileMode) error {
	const op = "mkdir"

	absPath, _ := fs.Abs(path)

	fs.mu.RLock()
	child, childOk := fs.nodes[absPath]
	fs.mu.RUnlock()

	if childOk {
		if child.mode.IsDir() {
			return nil
		}

		return &os.PathError{Op: op, Path: path, Err: avfs.ErrNotADirectory}
	}

	var (
		ds     []string
		parent *node
	)

	dirName := absPath

	for {
		fs.mu.RLock()
		nd, ok := fs.nodes[dirName]
		fs.mu.RUnlock()

		if ok {
			parent = nd
			if !parent.mode.IsDir() {
				return &os.PathError{Op: op, Path: dirName, Err: avfs.ErrNotADirectory}
			}

			break
		}

		ds = append(ds, dirName)

		dirName, _ = split(dirName)
	}

	for _, absPath := range ds {
		_, fileName := split(absPath)

		parent = fs.createDir(parent, absPath, fileName, perm)
	}

	return nil
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Open(path string) (avfs.File, error) {
	return fs.OpenFile(path, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) OpenFile(name string, flag int, perm os.FileMode) (avfs.File, error) {
	const op = "open"

	var (
		at      int64
		wm      avfs.WantMode
		nilFile *OrefaFile
	)

	if flag == os.O_RDONLY || flag&os.O_RDWR != 0 {
		wm = avfs.WantRead
	}

	if flag&(os.O_APPEND|os.O_CREATE|os.O_RDWR|os.O_TRUNC|os.O_WRONLY) != 0 {
		wm |= avfs.WantWrite
	}

	absPath, _ := fs.Abs(name)
	dirName, fileName := split(absPath)

	fs.mu.RLock()
	child, childOk := fs.nodes[absPath]
	parent, parentOk := fs.nodes[dirName]
	fs.mu.RUnlock()

	if !childOk {
		if !parentOk {
			return nil, &os.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
		}

		if flag&os.O_CREATE == 0 {
			return &OrefaFile{}, &os.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
		}

		if wm&avfs.WantWrite == 0 || parent == nil {
			return &OrefaFile{}, &os.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
		}

		child = fs.createFile(parent, absPath, fileName, perm)
	} else {
		if child.mode.IsDir() {
			if wm&avfs.WantWrite != 0 {
				return nilFile, &os.PathError{Op: op, Path: name, Err: avfs.ErrIsADirectory}
			}
		} else {
			if flag&(os.O_CREATE|os.O_EXCL) == os.O_CREATE|os.O_EXCL {
				return &OrefaFile{}, &os.PathError{Op: op, Path: name, Err: avfs.ErrFileExists}
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
		fs:       fs,
		nd:       child,
		wantMode: wm,
		name:     name,
		at:       at,
	}

	return f, nil
}

// ReadDir reads the directory named by dirname and returns
// a list of directory entries sorted by filename.
func (fs *OrefaFs) ReadDir(dirname string) ([]os.FileInfo, error) {
	return fsutil.ReadDir(fs, dirname)
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func (fs *OrefaFs) ReadFile(filename string) ([]byte, error) {
	return fsutil.ReadFile(fs, filename)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Readlink(name string) (string, error) {
	const op = "readlink"

	return "", &os.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (fs *OrefaFs) Rel(basepath, targpath string) (string, error) {
	return fsutil.Rel(basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Remove(name string) error {
	const op = "remove"

	absPath, _ := fs.Abs(name)
	dirName, fileName := split(absPath)

	fs.mu.RLock()
	child, childOk := fs.nodes[absPath]
	parent, parentOk := fs.nodes[dirName]
	fs.mu.RUnlock()

	if !childOk || !parentOk {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	child.mu.Lock()
	defer child.mu.Unlock()

	if child.mode.IsDir() && len(child.children) != 0 {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrDirNotEmpty}
	}

	delete(parent.children, fileName)

	child.remove()

	fs.mu.Lock()
	delete(fs.nodes, absPath)
	fs.mu.Unlock()

	return nil
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) RemoveAll(path string) error {
	if path == "" {
		// fail silently to retain compatibility with previous behavior of RemoveAll.
		return nil
	}

	absPath, _ := fs.Abs(path)
	dirName, fileName := split(absPath)

	fs.mu.RLock()
	child, childOk := fs.nodes[absPath]
	parent, parentOk := fs.nodes[dirName]
	fs.mu.RUnlock()

	if !childOk || !parentOk {
		return nil
	}

	if child.mode.IsDir() {
		fs.removeAll(absPath, child)
	}

	fs.mu.Lock()
	delete(fs.nodes, absPath)
	fs.mu.Unlock()

	child.remove()

	parent.mu.Lock()
	delete(parent.children, fileName)
	parent.mu.Unlock()

	return nil
}

func (fs *OrefaFs) removeAll(absPath string, rootNode *node) {
	if rootNode.mode.IsDir() {
		for fileName, nd := range rootNode.children {
			path := absPath + string(avfs.PathSeparator) + fileName

			fs.removeAll(path, nd)
		}
	}

	fs.mu.Lock()
	delete(fs.nodes, absPath)
	fs.mu.Unlock()

	rootNode.remove()
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (fs *OrefaFs) Rename(oldname, newname string) error {
	const op = "rename"

	oAbsPath, _ := fs.Abs(oldname)
	nAbsPath, _ := fs.Abs(newname)

	if oAbsPath == nAbsPath {
		return nil
	}

	oDirName, oFileName := split(oAbsPath)
	nDirName, nFileName := split(nAbsPath)

	fs.mu.RLock()
	oChild, oChildOk := fs.nodes[oAbsPath]
	oParent, oParentOk := fs.nodes[oDirName]
	nChild, nChildOk := fs.nodes[nAbsPath]
	nParent, nParentOk := fs.nodes[nDirName]
	fs.mu.RUnlock()

	if !oChildOk || !oParentOk || !nParentOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrNoSuchFileOrDir}
	}

	if oChild.mode.IsDir() && nChildOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrFileExists}
	}

	if !oChild.mode.IsDir() && nChildOk && nChild.mode.IsDir() {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrIsADirectory}
	}

	nParent.mu.Lock()

	if nParent != oParent {
		oParent.mu.Lock()
	}

	nParent.children[nFileName] = oChild

	delete(oParent.children, oFileName)

	fs.mu.Lock()

	fs.nodes[nAbsPath] = oChild
	delete(fs.nodes, oAbsPath)

	if oChild.mode.IsDir() {
		oRoot := oAbsPath + string(avfs.PathSeparator)

		for absPath, node := range fs.nodes {
			if strings.HasPrefix(absPath, oRoot) {
				nPath := nAbsPath + absPath[len(oAbsPath):]
				fs.nodes[nPath] = node

				delete(fs.nodes, absPath)
			}
		}
	}

	fs.mu.Unlock()

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
func (fs *OrefaFs) SameFile(fi1, fi2 os.FileInfo) bool {
	return reflect.DeepEqual(fi1, fi2)
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (fs *OrefaFs) Split(path string) (dir, file string) {
	return fsutil.Split(fs, path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Stat(path string) (os.FileInfo, error) {
	const op = "stat"

	return fs.stat(path, op)
}

// stat is the internal function used by Stat and Lstat.
func (fs *OrefaFs) stat(path, op string) (os.FileInfo, error) {
	absPath, _ := fs.Abs(path)
	dirName, fileName := split(absPath)

	fs.mu.RLock()
	child, childOk := fs.nodes[absPath]
	fs.mu.RUnlock()

	if !childOk {
		for {
			fs.mu.RLock()
			parent, parentOk := fs.nodes[dirName]
			fs.mu.RUnlock()

			if parentOk {
				if parent.mode.IsDir() {
					return nil, &os.PathError{Op: op, Path: path, Err: avfs.ErrNoSuchFileOrDir}
				}

				return nil, &os.PathError{Op: op, Path: path, Err: avfs.ErrNotADirectory}
			}

			dirName, _ = split(dirName)
		}
	}

	fst := child.fillStatFrom(fileName)

	return fst, nil
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (fs *OrefaFs) Symlink(oldname, newname string) error {
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
func (fs *OrefaFs) TempDir(dir, prefix string) (name string, err error) {
	return fsutil.TempDir(fs, dir, prefix)
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
func (fs *OrefaFs) TempFile(dir, pattern string) (f avfs.File, err error) {
	return fsutil.TempFile(fs, dir, pattern)
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (fs *OrefaFs) ToSlash(path string) string {
	return path
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (fs *OrefaFs) Truncate(name string, size int64) error {
	const op = "truncate"

	absPath, _ := fs.Abs(name)

	fs.mu.RLock()
	child, childOk := fs.nodes[absPath]
	fs.mu.RUnlock()

	if !childOk {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
	}

	if child.mode.IsDir() {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrIsADirectory}
	}

	child.mu.Lock()
	child.truncate(size)
	child.mu.Unlock()

	return nil
}

// UMask sets the file mode creation mask.
func (fs *OrefaFs) UMask(mask os.FileMode) {
	atomic.StoreInt32(&fs.umask, int32(mask))
}

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.
func (fs *OrefaFs) Walk(root string, walkFn filepath.WalkFunc) error {
	return fsutil.Walk(fs, root, walkFn)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func (fs *OrefaFs) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return fsutil.WriteFile(fs, filename, data, perm)
}

// File functions

// Chdir changes the current working directory to the file,
// which must be a directory.
// If there is an error, it will be of type *PathError.
func (f *OrefaFile) Chdir() error {
	const op = "chdir"

	if f == nil {
		return os.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return os.ErrInvalid
	}

	if f.nd == nil {
		return &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	if !f.nd.mode.IsDir() {
		return &os.PathError{Op: op, Path: f.name, Err: avfs.ErrNotADirectory}
	}

	f.fs.curDir = f.name

	return nil
}

// Chmod changes the mode of the file to mode.
// If there is an error, it will be of type *PathError.
func (f *OrefaFile) Chmod(mode os.FileMode) error {
	const op = "chmod"

	if f == nil {
		return os.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return os.ErrInvalid
	}

	if f.nd == nil {
		return &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	f.nd.setMode(mode)

	return nil
}

// Chown changes the numeric uid and gid of the named file.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (f *OrefaFile) Chown(uid, gid int) error {
	const op = "chown"

	if f == nil {
		return os.ErrInvalid
	}

	return &os.PathError{Op: op, Path: f.name, Err: avfs.ErrPermDenied}
}

// Close closes the File, rendering it unusable for I/O.
// On files that support SetDeadline, any pending I/O operations will
// be canceled and return immediately with an error.
func (f *OrefaFile) Close() error {
	const op = "close"

	if f == nil {
		return os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.nd == nil {
		if f.name == "" {
			return os.ErrInvalid
		}

		return &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	f.dirInfos = nil
	f.dirNames = nil
	f.nd = nil

	return nil
}

// Fd returns the integer Unix file descriptor referencing the open file.
// The file descriptor is valid only until f.Close is called or f is garbage collected.
// On Unix systems this will cause the SetDeadline methods to stop working.
func (f *OrefaFile) Fd() uintptr {
	return uintptr(math.MaxUint64)
}

// Name returns the link of the file as presented to Open.
func (f *OrefaFile) Name() string {
	return f.name
}

// Read reads up to len(b) bytes from the OrefaFile.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *OrefaFile) Read(b []byte) (n int, err error) {
	const op = "read"

	if f == nil {
		return 0, os.ErrInvalid
	}

	f.mu.RLock()

	if f.name == "" {
		f.mu.RUnlock()

		return 0, os.ErrInvalid
	}

	if f.nd == nil {
		f.mu.RUnlock()

		return 0, &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() {
		f.mu.RUnlock()

		return 0, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrIsADirectory}
	}

	if f.wantMode&avfs.WantRead == 0 {
		f.mu.RUnlock()

		return 0, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrBadFileDesc}
	}

	f.mu.RUnlock()

	nd.mu.RLock()
	n = copy(b, nd.data[f.at:])
	nd.mu.RUnlock()

	f.mu.Lock()
	f.at += int64(n)
	f.mu.Unlock()

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

// ReadAt reads len(b) bytes from the File starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (f *OrefaFile) ReadAt(b []byte, off int64) (n int, err error) {
	const op = "read"

	if f == nil {
		return 0, os.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return 0, os.ErrInvalid
	}

	if f.nd == nil {
		return 0, &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() {
		return 0, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrIsADirectory}
	}

	if off < 0 {
		return 0, &os.PathError{Op: "readat", Path: f.name, Err: avfs.ErrNegativeOffset}
	}

	if f.wantMode&avfs.WantRead == 0 {
		return 0, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrBadFileDesc}
	}

	nd.mu.RLock()
	defer nd.mu.RUnlock()

	if int(off) > len(nd.data) {
		return 0, io.EOF
	}

	n = copy(b, nd.data[off:])
	if n < len(b) {
		return n, io.EOF
	}

	return n, nil
}

// Readdir reads the contents of the directory associated with file and
// returns a slice of up to n FileInfo values, as would be returned
// by Lstat, in directory order. Subsequent calls on the same file will yield
// further FileInfos.
//
// If n > 0, Readdir returns at most n FileInfo structures. In this case, if
// Readdir returns an empty slice, it will return a non-nil error
// explaining why. At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdir returns all the FileInfo from the directory in
// a single slice. In this case, if Readdir succeeds (reads all
// the way to the end of the directory), it returns the slice and a
// nil error. If it encounters an error before the end of the
// directory, Readdir returns the FileInfo read until that point
// and a non-nil error.
func (f *OrefaFile) Readdir(n int) (fi []os.FileInfo, err error) {
	const op = "readdirent"

	if f == nil {
		return nil, os.ErrInvalid
	}

	f.mu.RLock()

	if f.name == "" {
		f.mu.RUnlock()

		return nil, os.ErrInvalid
	}

	if f.nd == nil {
		f.mu.RUnlock()

		return nil, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrFileClosing}
	}

	nd := f.nd
	if !nd.mode.IsDir() {
		f.mu.RUnlock()

		return nil, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrNotADirectory}
	}

	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	if n <= 0 || f.dirInfos == nil {
		nd.mu.RLock()
		infos := nd.infos()
		nd.mu.RUnlock()

		f.dirIndex = 0

		if n <= 0 {
			f.dirInfos = nil
			return infos, nil
		}

		f.dirInfos = infos
	}

	start := f.dirIndex
	if start >= len(f.dirInfos) {
		f.dirIndex = 0
		f.dirInfos = nil

		return nil, io.EOF
	}

	end := start + n
	if end > len(f.dirInfos) {
		end = len(f.dirInfos)
	}

	f.dirIndex = end

	return f.dirInfos[start:end], nil
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
func (f *OrefaFile) Readdirnames(n int) (names []string, err error) {
	const op = "readdirent"

	if f == nil {
		return nil, os.ErrInvalid
	}

	f.mu.RLock()

	if f.name == "" {
		f.mu.RUnlock()

		return nil, os.ErrInvalid
	}

	if f.nd == nil {
		f.mu.RUnlock()

		return nil, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrFileClosing}
	}

	nd := f.nd
	if !nd.mode.IsDir() {
		f.mu.RUnlock()

		return nil, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrNotADirectory}
	}

	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	if n <= 0 || f.dirNames == nil {
		nd.mu.RLock()
		names := nd.names()
		nd.mu.RUnlock()

		f.dirIndex = 0

		if n <= 0 {
			f.dirNames = nil
			return names, nil
		}

		f.dirNames = names
	}

	start := f.dirIndex
	if start >= len(f.dirNames) {
		f.dirIndex = 0
		f.dirNames = nil

		return nil, io.EOF
	}

	end := start + n
	if end > len(f.dirNames) {
		end = len(f.dirNames)
	}

	f.dirIndex = end

	return f.dirNames[start:end], nil
}

// Seek sets the offset for the next Read or Write on file to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means
// relative to the current offset, and 2 means relative to the end.
// It returns the new offset and an error, if any.
// The behavior of Seek on a file opened with O_APPEND is not specified.
func (f *OrefaFile) Seek(offset int64, whence int) (ret int64, err error) {
	const op = "seek"

	if f == nil {
		return 0, os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.name == "" {
		return 0, os.ErrInvalid
	}

	if f.nd == nil {
		return 0, &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() {
		return 0, nil
	}

	nd.mu.RLock()
	size := int64(len(nd.data))
	nd.mu.RUnlock()

	switch whence {
	case io.SeekStart:
		if offset < 0 {
			return 0, &os.PathError{Op: op, Path: f.name, Err: os.ErrInvalid}
		}

		f.at = offset
	case io.SeekCurrent:
		if f.at+offset < 0 {
			return 0, &os.PathError{Op: op, Path: f.name, Err: os.ErrInvalid}
		}

		f.at += offset
	case io.SeekEnd:
		if size+offset < 0 {
			return 0, &os.PathError{Op: op, Path: f.name, Err: os.ErrInvalid}
		}

		f.at = size + offset
	default:
		return 0, &os.PathError{Op: op, Path: f.name, Err: os.ErrInvalid}
	}

	return f.at, nil
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *OrefaFile) Stat() (os.FileInfo, error) {
	const op = "stat"

	if f == nil {
		return nil, os.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return fStat{}, os.ErrInvalid
	}

	if f.nd == nil {
		return fStat{}, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrFileClosing}
	}

	_, name := split(f.name)
	fst := f.nd.fillStatFrom(name)

	return fst, nil
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy
// of recently written data to disk.
func (f *OrefaFile) Sync() error {
	const op = "sync"

	if f == nil {
		return os.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return os.ErrInvalid
	}

	if f.nd == nil {
		return &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	return nil
}

// Truncate changes the size of the file.
// It does not change the I/O offset.
// If there is an error, it will be of type *PathError.
func (f *OrefaFile) Truncate(size int64) error {
	const op = "truncate"

	if f == nil {
		return os.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return os.ErrInvalid
	}

	if f.nd == nil {
		return &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() || f.wantMode&avfs.WantWrite == 0 {
		return &os.PathError{Op: op, Path: f.name, Err: os.ErrInvalid}
	}

	nd.mu.Lock()

	nd.truncate(size)
	nd.mtime = time.Now().UnixNano()

	nd.mu.Unlock()

	return nil
}

// Write writes len(b) bytes to the File.
// It returns the number of bytes written and an error, if any.
// Write returns a non-nil error when n != len(b).
func (f *OrefaFile) Write(b []byte) (n int, err error) {
	const op = "write"

	if f == nil {
		return 0, os.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return 0, os.ErrInvalid
	}

	if f.nd == nil {
		return 0, &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() || f.wantMode&avfs.WantWrite == 0 {
		return 0, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrBadFileDesc}
	}

	nd.mu.Lock()

	n = copy(nd.data[f.at:], b)
	if n < len(b) {
		nd.data = append(nd.data, b[n:]...)
		n = len(b)
	}

	nd.mtime = time.Now().UnixNano()

	nd.mu.Unlock()

	f.at += int64(n)

	return n, nil
}

// WriteAt writes len(b) bytes to the File starting at byte offset off.
// It returns the number of bytes written and an error, if any.
// WriteAt returns a non-nil error when n != len(b).
func (f *OrefaFile) WriteAt(b []byte, off int64) (n int, err error) {
	const op = "write"

	if f == nil {
		return 0, os.ErrInvalid
	}

	if off < 0 {
		return 0, &os.PathError{Op: "writeat", Path: f.name, Err: avfs.ErrNegativeOffset}
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return 0, os.ErrInvalid
	}

	if f.nd == nil {
		return 0, &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	nd := f.nd
	if nd.mode.IsDir() || f.wantMode&avfs.WantWrite == 0 {
		return 0, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrBadFileDesc}
	}

	nd.mu.Lock()

	diff := off + int64(len(b)) - nd.size()
	if diff > 0 {
		nd.data = append(nd.data, make([]byte, diff)...)
	}

	n = copy(nd.data[off:], b)

	nd.mtime = time.Now().UnixNano()

	nd.mu.Unlock()

	return n, nil
}

// WriteString is like Write, but writes the contents of string s rather than
// a slice of bytes.
func (f *OrefaFile) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}
