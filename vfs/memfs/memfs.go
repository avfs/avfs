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
// it supports several features :
//   - can emulate Linux or Windows systems regardless of the host system
//   - checks files permissions
//   - supports different Identity managers
//   - supports multiple concurrent users
//   - supports Hard links
//   - supports symbolic links
package memfs

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
func (vfs *MemFS) Abs(path string) (string, error) {
	return avfs.Abs(vfs, path, vfs.CurDir())
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *MemFS) Base(path string) string {
	return avfs.Base(vfs, path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Chdir(dir string) error {
	const op = "chdir"

	_, child, pi, err := vfs.searchNode(dir, slmLstat)
	if err != vfs.err.FileExists {
		return &fs.PathError{Op: op, Path: dir, Err: err}
	}

	c, ok := child.(*dirNode)
	if !ok {
		err = vfs.err.NotADirectory
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinDirNameInvalid
		}

		return &fs.PathError{Op: op, Path: dir, Err: err}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.checkPermission(avfs.OpenLookup, vfs.User()) {
		return &fs.PathError{Op: op, Path: dir, Err: vfs.err.PermDenied}
	}

	_ = vfs.SetCurDir(pi.Path())

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

	_, child, _, err := vfs.searchNode(name, slmEval)
	if err != vfs.err.FileExists || child == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	child.Lock()
	defer child.Unlock()

	if !child.setMode(mode, vfs.User()) {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.OpNotPermitted}
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

	if (vfs.HasFeature(avfs.FeatIdentityMgr) && !vfs.User().IsAdmin()) || vfs.OSType() == avfs.OsWindows {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.OpNotPermitted}
	}

	_, child, _, err := vfs.searchNode(name, slmEval)
	if err != vfs.err.FileExists || child == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	child.Lock()
	child.setOwner(uid, gid)
	child.Unlock()

	return nil
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Chtimes(name string, _, mtime time.Time) error {
	const op = "chtimes"

	_, child, _, err := vfs.searchNode(name, slmLstat)
	if err != vfs.err.FileExists || child == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	child.Lock()
	defer child.Unlock()

	if !child.setModTime(mtime, vfs.User()) {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.OpNotPermitted}
	}

	return nil
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
func (vfs *MemFS) Clean(path string) string {
	return avfs.Clean(vfs, path)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Create(name string) (avfs.File, error) {
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
func (vfs *MemFS) CreateTemp(dir, pattern string) (avfs.File, error) {
	return avfs.CreateTemp(vfs, dir, pattern)
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *MemFS) Dir(path string) string {
	return avfs.Dir(vfs, path)
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *MemFS) EvalSymlinks(path string) (string, error) {
	const op = "lstat"

	_, _, pi, err := vfs.searchNode(path, slmEval)
	if err != vfs.err.FileExists {
		return "", &fs.PathError{Op: op, Path: pi.LeftPart(), Err: err}
	}

	return pi.Path(), nil
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *MemFS) FromSlash(path string) string {
	return avfs.FromSlash(vfs, path)
}

// Getwd returns a rooted name link corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *MemFS) Getwd() (dir string, err error) {
	return vfs.CurDir(), nil
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
	return avfs.Glob(vfs, pattern)
}

// IsAbs reports whether the path is absolute.
func (vfs *MemFS) IsAbs(path string) bool {
	return avfs.IsAbs(vfs, path)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *MemFS) IsPathSeparator(c uint8) bool {
	return avfs.IsPathSeparator(vfs, c)
}

// Join joins any number of path elements into a single path,
// separating them with an OS specific Separator. Empty elements
// are ignored. The result is Cleaned. However, if the argument
// list is empty or all its elements are empty, Join returns
// an empty string.
// On Windows, the result will only be a UNC path if the first
// non-empty element is a UNC path.
func (vfs *MemFS) Join(elem ...string) string {
	return avfs.Join(vfs, elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *MemFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	if (vfs.HasFeature(avfs.FeatIdentityMgr) && !vfs.User().IsAdmin()) || vfs.OSType() == avfs.OsWindows {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.OpNotPermitted}
	}

	_, child, _, err := vfs.searchNode(name, slmLstat)
	if err != vfs.err.FileExists || child == nil {
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

	_, oChild, _, oerr := vfs.searchNode(oldname, slmLstat)
	if oerr != vfs.err.FileExists || oChild == nil {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: oerr}
	}

	nParent, _, pi, nerr := vfs.searchNode(newname, slmLstat)
	if !vfs.isNotExist(nerr) {
		if vfs.OSType() == avfs.OsWindows {
			nerr = avfs.ErrWinAlreadyExists
		}

		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: nerr}
	}

	nParent.mu.Lock()
	defer nParent.mu.Unlock()

	if !nParent.checkPermission(avfs.OpenWrite, vfs.User()) {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.PermDenied}
	}

	c, ok := oChild.(*fileNode)
	if !ok {
		err := error(avfs.ErrOpNotPermitted)
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinAccessDenied
		}

		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: err}
	}

	c.mu.Lock()
	nParent.addChild(pi.Part(), c)

	c.nlink++
	c.mu.Unlock()

	return nil
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Lstat(path string) (fs.FileInfo, error) {
	op := "lstat"
	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
	}

	_, child, pi, err := vfs.searchNode(path, slmLstat)
	if err != vfs.err.FileExists || child == nil {
		return nil, &fs.PathError{Op: op, Path: path, Err: err}
	}

	fst := child.fillStatFrom(pi.Part())

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
func (vfs *MemFS) Match(pattern, name string) (matched bool, err error) {
	return avfs.Match(vfs, pattern, name)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Mkdir(name string, perm fs.FileMode) error {
	const op = "mkdir"

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchDir}
	}

	parent, _, pi, err := vfs.searchNode(name, slmEval)
	if !vfs.isNotExist(err) || !pi.IsLast() {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if !parent.checkPermission(avfs.OpenWrite|avfs.OpenLookup, vfs.User()) {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.PermDenied}
	}

	part := pi.Part()
	if parent.children[part] != nil {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.FileExists}
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

	parent, child, pi, err := vfs.searchNode(path, slmEval)
	switch child.(type) {
	case *dirNode:
		if err != vfs.err.FileExists {
			return &fs.PathError{Op: op, Path: path, Err: err}
		}

		return nil
	case *fileNode:
		return &fs.PathError{Op: op, Path: pi.LeftPart(), Err: vfs.err.NotADirectory}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if !parent.checkPermission(avfs.OpenWrite|avfs.OpenLookup, vfs.User()) {
		return &fs.PathError{Op: op, Path: path, Err: vfs.err.PermDenied}
	}

	dn := parent

	for {
		part := pi.Part()
		if dn.children[part] != nil {
			break
		}

		dn = vfs.createDir(dn, part, perm)

		if !pi.Next() {
			break
		}
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
	return avfs.MkdirTemp(vfs, dir, pattern)
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Open(name string) (avfs.File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) OpenFile(name string, flag int, perm fs.FileMode) (avfs.File, error) {
	op := "open"

	if name == "" {
		return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
	}

	at := int64(0)
	om := avfs.ToOpenMode(flag)

	parent, child, pi, err := vfs.searchNode(name, slmEval)
	if err != vfs.err.FileExists && !vfs.isNotExist(err) || !pi.IsLast() {
		return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: err}
	}

	if vfs.isNotExist(err) {
		if om&avfs.OpenCreate == 0 {
			return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: err}
		}

		parent.mu.Lock()
		defer parent.mu.Unlock()

		if om&avfs.OpenWrite == 0 || !parent.checkPermission(avfs.OpenWrite|avfs.OpenLookup, vfs.User()) {
			return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.PermDenied}
		}

		part := pi.Part()

		child = parent.children[part]
		if child == nil {
			child = vfs.createFile(parent, part, perm)
			f := &MemFile{
				nd:       child,
				vfs:      vfs,
				name:     name,
				at:       at,
				openMode: om,
			}

			return f, nil
		}
	}

	switch c := child.(type) {
	case *fileNode:
		c.mu.Lock()
		defer c.mu.Unlock()

		if !c.checkPermission(om, vfs.User()) {
			return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.PermDenied}
		}

		if om&avfs.OpenCreateExcl != 0 {
			return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.FileExists}
		}

		if om&avfs.OpenDir != 0 {
			if vfs.OSType() == avfs.OsWindows {
				op = "readdir"
			}

			return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.NotADirectory}
		}

		if om&avfs.OpenTruncate != 0 {
			c.truncate(0)
		}

		if om&avfs.OpenAppend != 0 {
			at = c.size()
		}

	case *dirNode:
		c.mu.Lock()
		defer c.mu.Unlock()

		if om&avfs.OpenWrite != 0 {
			return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.IsADirectory}
		}

		if !c.checkPermission(om, vfs.User()) {
			return (*MemFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.PermDenied}
		}
	}

	f := &MemFile{
		nd:       child,
		vfs:      vfs,
		name:     name,
		at:       at,
		openMode: om,
	}

	return f, nil
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *MemFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return avfs.ReadDir(vfs, name)
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
func (vfs *MemFS) ReadFile(name string) ([]byte, error) {
	return avfs.ReadFile(vfs, name)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Readlink(name string) (string, error) {
	const op = "readlink"

	_, child, _, err := vfs.searchNode(name, slmLstat)
	if err != vfs.err.FileExists {
		return "", &fs.PathError{Op: op, Path: name, Err: err}
	}

	sl, ok := child.(*symlinkNode)
	if !ok {
		err = avfs.ErrInvalidArgument
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinNotReparsePoint
		}

		return "", &fs.PathError{Op: op, Path: name, Err: err}
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
	return avfs.Rel(vfs, basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Remove(name string) error {
	const op = "remove"

	parent, child, pi, err := vfs.searchNode(name, slmLstat)
	if err != vfs.err.FileExists || child == nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if !parent.checkPermission(avfs.OpenWrite, vfs.User()) {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.PermDenied}
	}

	child.Lock()
	defer child.Unlock()

	if c, ok := child.(*dirNode); ok {
		if len(c.children) != 0 {
			return &fs.PathError{Op: op, Path: name, Err: vfs.err.DirNotEmpty}
		}
	}

	part := pi.Part()
	if parent.children[part] == nil {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchDir}
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

	parent, child, pi, err := vfs.searchNode(path, slmLstat)
	if vfs.isNotExist(err) {
		return nil
	}

	if err != vfs.err.FileExists {
		return &fs.PathError{Op: op, Path: path, Err: err}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if c, ok := child.(*dirNode); ok && len(c.children) != 0 {
		err = vfs.removeAll(c)
		if err != nil {
			return &fs.PathError{Op: op, Path: path, Err: err}
		}
	}

	if ok := parent.checkPermission(avfs.OpenWrite, vfs.User()); !ok {
		return &fs.PathError{Op: op, Path: path, Err: vfs.err.PermDenied}
	}

	parent.removeChild(pi.Part())
	child.delete()

	return nil
}

func (vfs *MemFS) removeAll(parent *dirNode) error {
	parent.mu.Lock()
	defer parent.mu.Unlock()

	if ok := parent.checkPermission(avfs.OpenWrite, vfs.User()); !ok {
		return vfs.err.PermDenied
	}

	for _, child := range parent.children {
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

	oParent, oChild, oPI, oErr := vfs.searchNode(oldpath, slmLstat)
	if oErr != vfs.err.FileExists {
		return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: oErr}
	}

	nParent, nChild, nPI, nErr := vfs.searchNode(newpath, slmLstat)
	if nErr != vfs.err.FileExists && !vfs.isNotExist(nErr) {
		return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: nErr}
	}

	oParent.mu.Lock()
	defer oParent.mu.Unlock()

	if !oParent.checkPermission(avfs.OpenWrite, vfs.User()) {
		return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: vfs.err.PermDenied}
	}

	if nParent != oParent {
		nParent.mu.Lock()
		defer nParent.mu.Unlock()

		if !nParent.checkPermission(avfs.OpenWrite, vfs.User()) {
			return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: vfs.err.PermDenied}
		}
	}

	if oPI.Path() == nPI.Path() {
		return nil
	}

	switch oChild.(type) {
	case *dirNode:
		if !vfs.isNotExist(nErr) {
			if vfs.OSType() == avfs.OsWindows {
				nErr = avfs.ErrWinAccessDenied
			}

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
			err := error(avfs.ErrFileExists)
			if vfs.OSType() == avfs.OsWindows {
				err = avfs.ErrWinAccessDenied
			}

			return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: err}
		}
	}

	nParent.addChild(nPI.Part(), oChild)
	oParent.removeChild(oPI.Part())

	return nil
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (*MemFS) SameFile(fi1, fi2 fs.FileInfo) bool {
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

// SetUserByName sets the current user by name.
// If the user is not found, the returned error is of type UnknownUserError.
func (vfs *MemFS) SetUserByName(name string) error {
	return avfs.SetUserByName(vfs, name)
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *MemFS) Split(path string) (dir, file string) {
	return avfs.Split(vfs, path)
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Stat(path string) (fs.FileInfo, error) {
	op := "stat"
	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
	}

	_, child, pi, err := vfs.searchNode(path, slmStat)
	if err != vfs.err.FileExists || child == nil {
		return nil, &fs.PathError{Op: op, Path: path, Err: err}
	}

	fst := child.fillStatFrom(pi.Part())

	return fst, nil
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (vfs *MemFS) Sub(dir string) (avfs.VFS, error) {
	const op = "sub"

	_, child, _, err := vfs.searchNode(dir, slmEval)
	if err != vfs.err.FileExists || child == nil {
		return nil, &fs.PathError{Op: op, Path: dir, Err: err}
	}

	c, ok := child.(*dirNode)
	if !ok {
		return nil, &fs.PathError{Op: op, Path: dir, Err: vfs.err.NotADirectory}
	}

	subFS := *vfs
	subFS.rootNode = c

	return &subFS, nil
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *MemFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	parent, _, pi, nerr := vfs.searchNode(newname, slmLstat)
	if !vfs.isNotExist(nerr) {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: nerr}
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	if !parent.checkPermission(avfs.OpenWrite, vfs.User()) {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.PermDenied}
	}

	link := vfs.Clean(oldname)

	vfs.createSymlink(parent, pi.Part(), link)

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
	return avfs.TempDir(vfs)
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *MemFS) ToSlash(path string) string {
	return avfs.ToSlash(vfs, path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (*MemFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return info.Sys().(avfs.SysStater) //nolint:forcetypeassert // type assertion must be checked
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *MemFS) Truncate(name string, size int64) error {
	op := "truncate"

	_, child, _, err := vfs.searchNode(name, slmEval)
	if err != vfs.err.FileExists {
		if vfs.OSType() == avfs.OsWindows {
			op = "open"
		}

		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	c, ok := child.(*fileNode)
	if !ok {
		if vfs.OSType() == avfs.OsWindows {
			op = "open"
		}

		return &fs.PathError{Op: op, Path: name, Err: vfs.err.IsADirectory}
	}

	if size < 0 {
		err = error(avfs.ErrInvalidArgument)
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinInvalidParameter
		}

		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	c.mu.Lock()
	c.truncate(size)
	c.mu.Unlock()

	return nil
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
	return avfs.WalkDir(vfs, root, fn)
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
func (vfs *MemFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return avfs.WriteFile(vfs, name, data, perm)
}
