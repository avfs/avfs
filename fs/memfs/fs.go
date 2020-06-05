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

package memfs

import (
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
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
func (fs *MemFs) Abs(path string) (string, error) {
	return fsutil.Abs(fs, path)
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (fs *MemFs) Base(path string) string {
	return fsutil.Base(path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Chdir(dir string) error {
	const op = "chdir"

	_, child, absPath, _, _, err := fs.searchNode(dir, slmLstat)
	if err != avfs.ErrFileExists {
		return &os.PathError{Op: op, Path: dir, Err: err}
	}

	c, ok := child.(*dirNode)
	if !ok {
		return &os.PathError{Op: op, Path: dir, Err: avfs.ErrNotADirectory}
	}

	if !c.checkPermission(avfs.WantLookup, fs.user) {
		return &os.PathError{Op: op, Path: dir, Err: avfs.ErrPermDenied}
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
func (fs *MemFs) Chmod(name string, mode os.FileMode) error {
	const op = "chmod"

	_, child, _, _, _, err := fs.searchNode(name, slmEval)
	if err != avfs.ErrFileExists || child == nil {
		return &os.PathError{Op: op, Path: name, Err: err}
	}

	err = child.setMode(mode, fs.user)
	if err != nil {
		return &os.PathError{Op: op, Path: name, Err: err}
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
func (fs *MemFs) Chown(name string, uid, gid int) error {
	const op = "chown"

	_, child, _, _, _, err := fs.searchNode(name, slmEval)
	if err != avfs.ErrFileExists || child == nil {
		return &os.PathError{Op: op, Path: name, Err: err}
	}

	if !child.checkPermission(avfs.WantWrite, fs.user) {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrOpNotPermitted}
	}

	child.setOwner(uid, gid)

	return nil
}

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Chroot(path string) error {
	const op = "chroot"

	if !fs.user.IsRoot() {
		return &os.PathError{Op: op, Path: path, Err: avfs.ErrOpNotPermitted}
	}

	_, child, _, _, _, err := fs.searchNode(path, slmEval)
	if err != avfs.ErrFileExists || child == nil {
		return &os.PathError{Op: op, Path: path, Err: err}
	}

	c, ok := child.(*dirNode)
	if !ok {
		return &os.PathError{Op: op, Path: path, Err: avfs.ErrNotADirectory}
	}

	fs.rootNode = c

	return nil
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Chtimes(name string, atime, mtime time.Time) error {
	const op = "chtimes"

	_, child, _, _, _, err := fs.searchNode(name, slmLstat)
	if err != avfs.ErrFileExists || child == nil {
		return &os.PathError{Op: op, Path: name, Err: err}
	}

	if !child.checkPermission(avfs.WantWrite, fs.user) {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrOpNotPermitted}
	}

	child.setModTime(mtime)

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
func (fs *MemFs) Clean(path string) string {
	return fsutil.Clean(path)
}

// Clone returns a shallow copy of the current file system.
func (fs *MemFs) Clone() avfs.Fs {
	newFs := *fs
	return &newFs
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Create(name string) (avfs.File, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (fs *MemFs) Dir(path string) string {
	return fsutil.Dir(path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (fs *MemFs) EvalSymlinks(path string) (string, error) {
	const op = "lstat"

	_, _, absPath, _, end, err := fs.searchNode(path, slmEval)
	if err != avfs.ErrFileExists {
		return "", &os.PathError{Op: op, Path: absPath[:end], Err: avfs.ErrNoSuchFileOrDir}
	}

	return absPath, nil
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
func (fs *MemFs) GetTempDir() string {
	return avfs.TmpDir
}

// GetUMask returns the file mode creation mask.
func (fs *MemFs) GetUMask() os.FileMode {
	u := atomic.LoadInt32(&fs.umask)

	return os.FileMode(u)
}

// Getwd returns a rooted name link corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (fs *MemFs) Getwd() (dir string, err error) {
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
func (fs *MemFs) Glob(pattern string) (matches []string, err error) {
	return fsutil.Glob(fs, pattern)
}

// IsAbs reports whether the path is absolute.
func (fs *MemFs) IsAbs(path string) bool {
	return fsutil.IsAbs(path)
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (fs *MemFs) IsExist(err error) bool {
	return fsutil.IsExist(err)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (fs *MemFs) IsNotExist(err error) bool {
	return fsutil.IsNotExist(err)
}

// IsPathSeparator reports whether c is a directory separator character.
func (fs *MemFs) IsPathSeparator(c uint8) bool {
	return fsutil.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (fs *MemFs) Join(elem ...string) string {
	return fsutil.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (fs *MemFs) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	_, child, _, _, _, err := fs.searchNode(name, slmLstat)
	if err != avfs.ErrFileExists || child == nil {
		return &os.PathError{Op: op, Path: name, Err: err}
	}

	if !child.checkPermission(avfs.WantWrite, fs.user) {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrOpNotPermitted}
	}

	child.setOwner(uid, gid)

	return nil
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (fs *MemFs) Link(oldname, newname string) error {
	const op = "link"

	_, oChild, _, _, _, oerr := fs.searchNode(oldname, slmLstat)
	if oerr != avfs.ErrFileExists || oChild == nil {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: oerr}
	}

	nParent, _, absPath, start, end, nerr := fs.searchNode(newname, slmLstat)
	if nParent == nil || nerr != avfs.ErrNoSuchFileOrDir {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: nerr}
	}

	if !nParent.checkPermission(avfs.WantWrite, fs.user) {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrOpNotPermitted}
	}

	c, ok := oChild.(*fileNode)
	if !ok {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrOpNotPermitted}
	}

	part := absPath[start:end]

	nParent.mu.Lock()
	c.mu.Lock()
	nParent.addChild(part, c)
	c.refCount++
	c.mu.Unlock()
	nParent.mu.Unlock()

	return nil
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Lstat(path string) (os.FileInfo, error) {
	const op = "lstat"

	_, child, absPath, start, end, err := fs.searchNode(path, slmLstat)
	if err != avfs.ErrFileExists || child == nil {
		return nil, &os.PathError{Op: op, Path: path, Err: err}
	}

	part := absPath[start:end]
	fst := child.fillStatFrom(part)

	return fst, nil
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Mkdir(name string, perm os.FileMode) error {
	const op = "mkdir"

	if name == "" {
		return &os.PathError{Op: op, Path: "", Err: avfs.ErrNoSuchFileOrDir}
	}

	parent, _, absPath, start, end, err := fs.searchNode(name, slmEval)
	if err != avfs.ErrNoSuchFileOrDir || end != len(absPath) || parent == nil {
		return &os.PathError{Op: op, Path: name, Err: err}
	}

	if !parent.checkPermission(avfs.WantWrite|avfs.WantLookup, fs.user) {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
	}

	part := absPath[start:end]

	parent.mu.Lock()
	_ = fs.createDir(parent, part, perm)
	parent.mu.Unlock()

	return nil
}

// MkdirAll creates a directory named name,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If name is already a directory, MkdirAll does nothing
// and returns nil.
func (fs *MemFs) MkdirAll(path string, perm os.FileMode) error {
	const op = "mkdir"

	parent, child, absPath, start, end, err := fs.searchNode(path, slmEval)
	if err == avfs.ErrFileExists {
		_, ok := child.(*dirNode)
		if !ok {
			return &os.PathError{Op: op, Path: path, Err: avfs.ErrNotADirectory}
		}

		return nil
	}

	if err != avfs.ErrNoSuchFileOrDir {
		return &os.PathError{Op: op, Path: absPath[:end], Err: err}
	}

	if parent == nil || !parent.checkPermission(avfs.WantWrite|avfs.WantLookup, fs.user) {
		return &os.PathError{Op: op, Path: absPath[:end], Err: avfs.ErrPermDenied}
	}

	dn := parent

	for isLast := len(absPath) <= 1; !isLast; start = end + 1 {
		end, isLast = fsutil.SegmentPath(absPath, start)

		part := absPath[start:end]

		dn.mu.Lock()
		newDn := fs.createDir(dn, part, perm)
		dn.mu.Unlock()

		dn = newDn
	}

	return nil
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Open(path string) (avfs.File, error) {
	return fs.OpenFile(path, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) OpenFile(name string, flag int, perm os.FileMode) (avfs.File, error) {
	const op = "open"

	var (
		at      int64
		wm      avfs.WantMode
		nilFile *MemFile
	)

	if flag == os.O_RDONLY || flag&os.O_RDWR != 0 {
		wm = avfs.WantRead
	}

	if flag&(os.O_APPEND|os.O_CREATE|os.O_RDWR|os.O_TRUNC|os.O_WRONLY) != 0 {
		wm |= avfs.WantWrite
	}

	parent, child, absPath, start, end, err := fs.searchNode(name, slmEval)
	switch err {
	case avfs.ErrNoSuchFileOrDir:
		if end < len(absPath) {
			return nil, &os.PathError{Op: op, Path: name, Err: err}
		}

		if flag&os.O_CREATE == 0 {
			return &MemFile{}, &os.PathError{Op: op, Path: name, Err: err}
		}

		if wm&avfs.WantWrite == 0 || parent == nil || !parent.checkPermission(wm, fs.user) {
			return &MemFile{}, &os.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
		}

		part := absPath[start:end]

		parent.mu.Lock()
		child = fs.createFile(parent, part, perm)
		parent.mu.Unlock()

	case avfs.ErrFileExists:
		switch c := child.(type) {
		case *fileNode:
			if !c.checkPermission(wm, fs.user) {
				return &MemFile{}, &os.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
			}

			if flag&(os.O_CREATE|os.O_EXCL) == os.O_CREATE|os.O_EXCL {
				return &MemFile{}, &os.PathError{Op: op, Path: name, Err: avfs.ErrFileExists}
			}

			if flag&os.O_TRUNC != 0 {
				c.mu.Lock()
				c.truncate(0)
				c.mu.Unlock()
			}

			if flag&os.O_APPEND != 0 {
				at = c.Size()
			}

		case *dirNode:
			if wm&avfs.WantWrite != 0 {
				return nilFile, &os.PathError{Op: op, Path: name, Err: avfs.ErrIsADirectory}
			}

			if !c.checkPermission(wm, fs.user) {
				return &MemFile{}, &os.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
			}

		default:
			return &MemFile{}, os.ErrInvalid
		}

	default:
		return &MemFile{}, &os.PathError{Op: op, Path: name, Err: err}
	}

	f := &MemFile{
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
func (fs *MemFs) ReadDir(dirname string) ([]os.FileInfo, error) {
	return fsutil.ReadDir(fs, dirname)
}

// ReadFile reads the file named by filename and returns the contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func (fs *MemFs) ReadFile(filename string) ([]byte, error) {
	return fsutil.ReadFile(fs, filename)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Readlink(name string) (string, error) {
	const op = "readlink"

	_, child, _, _, _, err := fs.searchNode(name, slmLstat)
	if err != avfs.ErrFileExists {
		return "", &os.PathError{Op: op, Path: name, Err: err}
	}

	sl, ok := child.(*symlinkNode)
	if !ok {
		return "", &os.PathError{Op: op, Path: name, Err: os.ErrInvalid}
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
func (fs *MemFs) Rel(basepath, targpath string) (string, error) {
	return fsutil.Rel(basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Remove(name string) error {
	const op = "remove"

	parent, child, absPath, start, end, err := fs.searchNode(name, slmLstat)
	if err != avfs.ErrFileExists || parent == nil {
		return &os.PathError{Op: op, Path: name, Err: err}
	}

	if !parent.checkPermission(avfs.WantWrite, fs.user) {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrPermDenied}
	}

	if c, ok := child.(*dirNode); ok {
		c.mu.RLock()
		l := len(c.children)
		c.mu.RUnlock()

		if l != 0 {
			return &os.PathError{Op: op, Path: name, Err: avfs.ErrDirNotEmpty}
		}
	}

	part := absPath[start:end]

	parent.mu.Lock()
	parent.removeChild(part)
	parent.mu.Unlock()

	if c, ok := child.(*fileNode); ok {
		c.mu.Lock()
		c.deleteData()
		c.mu.Unlock()
	}

	return nil
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (fs *MemFs) RemoveAll(path string) error {
	const op = "unlinkat"

	if path == "" {
		// fail silently to retain compatibility with previous behavior of RemoveAll.
		return nil
	}

	parent, child, absPath, start, end, err := fs.searchNode(path, slmLstat)
	if fsutil.IsNotExist(err) {
		return nil
	}

	if err != avfs.ErrFileExists || parent == nil {
		return &os.PathError{Op: op, Path: path, Err: err}
	}

	if !parent.checkPermission(avfs.WantWrite, fs.user) {
		return &os.PathError{Op: op, Path: path, Err: avfs.ErrPermDenied}
	}

	var rs removeStack

	switch c := child.(type) {
	case *dirNode:
		rs.push(c)

	case *fileNode:
		c.mu.Lock()
		c.deleteData()
		c.mu.Unlock()
	}

	for rs.len() > 0 {
		p := rs.pop()
		p.mu.Lock()
		for _, child := range p.children {
			switch c := child.(type) {
			case *dirNode:
				rs.push(c)

			case *fileNode:
				c.deleteData()
			}
		}

		p.children = nil
		p.mu.Unlock()
	}

	part := absPath[start:end]

	parent.mu.Lock()
	parent.removeChild(part)
	parent.mu.Unlock()

	return nil
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (fs *MemFs) Rename(oldname, newname string) error {
	const op = "rename"

	oparent, ochild, oabsPath, ostart, oend, oerr := fs.searchNode(oldname, slmLstat)
	if oerr != avfs.ErrFileExists {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: oerr}
	}

	if oparent == nil || !oparent.checkPermission(avfs.WantWrite, fs.user) {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrPermDenied}
	}

	nparent, nchild, nabsPath, nstart, nend, nerr := fs.searchNode(newname, slmLstat)
	if nparent == nil || !nparent.checkPermission(avfs.WantWrite, fs.user) {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrPermDenied}
	}

	if oabsPath == nabsPath {
		return nil
	}

	npart := nabsPath[nstart:nend]
	opart := oabsPath[ostart:oend]

	switch ochild.(type) {
	case *dirNode:
		if nerr != avfs.ErrNoSuchFileOrDir {
			return &os.LinkError{Op: op, Old: oldname, New: newname, Err: nerr}
		}

	case *fileNode:
		if nchild == nil {
			break
		}

		switch nc := nchild.(type) {
		case *fileNode:
			nc.mu.Lock()
			nc.deleteData()
			nc.mu.Unlock()
		default:
			return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrIsADirectory}
		}
	default:
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrPermDenied}
	}

	nparent.mu.Lock()

	if nparent != oparent {
		oparent.mu.Lock()
	}

	nparent.addChild(npart, ochild)
	oparent.removeChild(opart)

	if nparent != oparent {
		oparent.mu.Unlock()
	}

	nparent.mu.Unlock()

	return nil
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (fs *MemFs) SameFile(fi1, fi2 os.FileInfo) bool {
	return reflect.DeepEqual(fi1, fi2)
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (fs *MemFs) Split(path string) (dir, file string) {
	return fsutil.Split(path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Stat(path string) (os.FileInfo, error) {
	const op = "stat"

	_, child, absPath, start, end, err := fs.searchNode(path, slmStat)
	if err != avfs.ErrFileExists || child == nil {
		return nil, &os.PathError{Op: op, Path: path, Err: err}
	}

	part := absPath[start:end]
	fst := child.fillStatFrom(part)

	return fst, nil
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (fs *MemFs) Symlink(oldname, newname string) error {
	const op = "symlink"

	parent, _, absPath, start, end, nerr := fs.searchNode(newname, slmLstat)
	if nerr != avfs.ErrNoSuchFileOrDir || parent == nil {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: nerr}
	}

	if !parent.checkPermission(avfs.WantWrite, fs.user) {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: avfs.ErrPermDenied}
	}

	link := fsutil.Clean(oldname)
	part := absPath[start:end]

	parent.mu.Lock()
	fs.createSymlink(parent, part, link)
	parent.mu.Unlock()

	return nil
}

// TempDir creates a new temporary directory in the directory dir
// with a link beginning with prefix and returns the name of the
// new directory. If dir is the empty string, GetTempDir uses the
// default directory for temporary files (see os.GetTempDir).
// Multiple programs calling GetTempDir simultaneously
// will not choose the same directory. It is the caller's responsibility
// to removeNodes the directory when no longer needed.
func (fs *MemFs) TempDir(dir, prefix string) (name string, err error) {
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
func (fs *MemFs) TempFile(dir, pattern string) (f avfs.File, err error) {
	return fsutil.TempFile(fs, dir, pattern)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (fs *MemFs) Truncate(name string, size int64) error {
	const op = "truncate"

	parent, child, _, _, _, err := fs.searchNode(name, slmEval)
	if err != avfs.ErrFileExists || parent == nil {
		return &os.PathError{Op: op, Path: name, Err: err}
	}

	c, ok := child.(*fileNode)
	if !ok {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrIsADirectory}
	}

	c.mu.Lock()
	c.truncate(size)
	c.mu.Unlock()

	return nil
}

// UMask sets the file mode creation mask.
func (fs *MemFs) UMask(mask os.FileMode) {
	atomic.StoreInt32(&fs.umask, int32(mask))
}

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.
func (fs *MemFs) Walk(root string, walkFn filepath.WalkFunc) error {
	return fsutil.Walk(fs, root, walkFn)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func (fs *MemFs) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return fsutil.WriteFile(fs, filename, data, perm)
}

// File functions

// Chdir changes the current working directory to the file,
// which must be a directory.
// If there is an error, it will be of type *PathError.
func (f *MemFile) Chdir() error {
	const op = "chdir"

	if f == nil {
		return os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.name == "" {
		return os.ErrInvalid
	}

	if f.nd == nil {
		return &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	_, ok := f.nd.(*dirNode)
	if !ok {
		return &os.PathError{Op: op, Path: f.name, Err: avfs.ErrNotADirectory}
	}

	f.fs.curDir = f.name

	return nil
}

// Chmod changes the mode of the file to mode.
// If there is an error, it will be of type *PathError.
func (f *MemFile) Chmod(mode os.FileMode) error {
	const op = "chmod"

	if f == nil {
		return os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.name == "" {
		return os.ErrInvalid
	}

	if f.nd == nil {
		return &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	err := f.nd.setMode(mode, f.fs.user)
	if err != nil {
		return &os.PathError{Op: op, Path: f.name, Err: err}
	}

	return nil
}

// Chown changes the numeric uid and gid of the named file.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (f *MemFile) Chown(uid, gid int) error {
	const op = "chown"

	if f == nil {
		return os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.name == "" {
		return os.ErrInvalid
	}

	if f.nd == nil {
		return &os.PathError{Op: op, Path: f.name, Err: os.ErrClosed}
	}

	if !f.nd.checkPermission(avfs.WantWrite, f.fs.user) {
		return &os.PathError{Op: op, Path: f.name, Err: avfs.ErrOpNotPermitted}
	}

	f.nd.setOwner(uid, gid)

	return nil
}

// Close closes the File, rendering it unusable for I/O.
// On files that support SetDeadline, any pending I/O operations will
// be canceled and return immediately with an error.
func (f *MemFile) Close() error {
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
func (f *MemFile) Fd() uintptr {
	return uintptr(math.MaxUint64)
}

// Name returns the link of the file as presented to Open.
func (f *MemFile) Name() string {
	if f == nil {
		panic("")
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.name
}

// Read reads up to len(b) bytes from the MemFile.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *MemFile) Read(b []byte) (n int, err error) {
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

	nd, ok := f.nd.(*fileNode)
	if !ok {
		f.mu.RUnlock()

		return 0, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrIsADirectory}
	}

	if f.wantMode&avfs.WantRead == 0 {
		f.mu.RUnlock()

		return 0, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrBadFileDesc}
	}

	nd.mu.RLock()
	n = copy(b, nd.data[f.at:])
	nd.mu.RUnlock()
	f.mu.RUnlock()

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
func (f *MemFile) ReadAt(b []byte, off int64) (n int, err error) {
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

	nd, ok := f.nd.(*fileNode)
	if !ok {
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
func (f *MemFile) Readdir(n int) (fi []os.FileInfo, err error) {
	const op = "readdirent"

	if f == nil {
		return nil, os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.name == "" {
		return nil, os.ErrInvalid
	}

	if f.nd == nil {
		return nil, avfs.ErrFileClosing
	}

	nd, ok := f.nd.(*dirNode)
	if !ok {
		return nil, &os.SyscallError{Syscall: op, Err: avfs.ErrNotADirectory}
	}

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
func (f *MemFile) Readdirnames(n int) (names []string, err error) {
	const op = "readdirent"

	if f == nil {
		return nil, os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.name == "" {
		return nil, os.ErrInvalid
	}

	if f.nd == nil {
		return nil, avfs.ErrFileClosing
	}

	nd, ok := f.nd.(*dirNode)
	if !ok {
		return nil, &os.SyscallError{Syscall: op, Err: avfs.ErrNotADirectory}
	}

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
func (f *MemFile) Seek(offset int64, whence int) (ret int64, err error) {
	const op = "seek"

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

	nd, ok := f.nd.(*fileNode)
	if !ok {
		f.mu.RUnlock()

		return 0, nil
	}

	switch whence {
	case io.SeekStart:
		f.at = offset
	case io.SeekCurrent:
		f.at += offset
	case io.SeekEnd:
		nd.mu.RLock()
		f.at = int64(len(nd.data)) + offset
		nd.mu.RUnlock()
	}

	f.mu.RUnlock()

	f.mu.Lock()
	at := f.at
	f.mu.Unlock()

	return at, nil
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *MemFile) Stat() (os.FileInfo, error) {
	const op = "stat"

	if f == nil {
		return nil, os.ErrInvalid
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.name == "" {
		return nil, os.ErrInvalid
	}

	if f.nd == nil {
		return fStat{}, &os.PathError{Op: op, Path: f.name, Err: avfs.ErrFileClosing}
	}

	name := fsutil.Base(f.name)
	fst := f.nd.fillStatFrom(name)

	return fst, nil
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy
// of recently written data to disk.
func (f *MemFile) Sync() error {
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
func (f *MemFile) Truncate(size int64) error {
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

	nd, ok := f.nd.(*fileNode)
	if !ok || f.wantMode&avfs.WantWrite == 0 {
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
func (f *MemFile) Write(b []byte) (n int, err error) {
	const op = "write"

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

	nd, ok := f.nd.(*fileNode)
	if !ok || f.wantMode&avfs.WantWrite == 0 {
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
func (f *MemFile) WriteAt(b []byte, off int64) (n int, err error) {
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

	nd, ok := f.nd.(*fileNode)
	if !ok || f.wantMode&avfs.WantWrite == 0 {
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
func (f *MemFile) WriteString(s string) (n int, err error) {
	return f.Write([]byte(s))
}
