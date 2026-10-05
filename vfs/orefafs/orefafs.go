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
//
// it supports several features :
//   - can emulate Linux or Windows systems regardless of the host system
//   - supports Hard links
package orefafs

import (
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/avfs/avfs"
)

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls [Clean] on the result.
func (vfs *OrefaFS) Abs(path string) (string, error) {
	return vfs.userDir.Abs(path)
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *OrefaFS) Base(path string) string {
	return vfs.userDir.Base(path)
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) Chdir(dir string) error {
	const op = "chdir"

	var err error

	if dir == "" {
		switch vfs.OSType() {
		case avfs.OsWindows:
			err = avfs.ErrWinInvalidName
		default:
			err = avfs.ErrNoSuchFileOrDir
		}

		return &fs.PathError{Op: op, Path: "", Err: err}
	}

	absPath, _ := vfs.Abs(dir)

	vfs.storage.mu.RLock()
	nd, ok := vfs.storage.nodes[absPath]
	vfs.storage.mu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: dir, Err: vfs.err.NoSuchFile}
	}

	if !nd.mode.IsDir() {
		switch vfs.OSType() {
		case avfs.OsWindows:
			err = avfs.ErrWinDirNameInvalid
		default:
			err = vfs.err.NotADirectory
		}

		return &fs.PathError{Op: op, Path: dir, Err: err}
	}

	_ = vfs.SetCurDir(absPath)

	return nil
}

// Chmod changes the mode of the named file to mode.
// If the file is a symbolic link, it changes the mode of the link's target.
// If there is an error, it will be of type [*PathError].
//
// A different subset of the mode bits are used, depending on the
// operating system.
//
// On Unix, the mode's permission bits, [ModeSetuid], [ModeSetgid], and
// [ModeSticky] are used.
//
// On Windows, only the 0o200 bit (owner writable) of mode is used; it
// controls whether the file's read-only attribute is set or cleared.
// The other bits are currently unused. For compatibility with Go 1.12
// and earlier, use a non-zero mode. Use mode 0o400 for a read-only
// file and 0o600 for a readable+writable file.
//
// On Plan 9, the mode's permission bits, [ModeAppend], [ModeExclusive],
// and [ModeTemporary] are used.
func (vfs *OrefaFS) Chmod(name string, mode fs.FileMode) error {
	const op = "chmod"

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchDir}
	}

	absPath, _ := vfs.Abs(name)

	vfs.storage.mu.RLock()
	nd, ok := vfs.storage.nodes[absPath]
	vfs.storage.mu.RUnlock()

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
// If there is an error, it will be of type [*PathError].
//
// On Windows or Plan 9, Chown always returns the [syscall.EWINDOWS] or
// [syscall.EPLAN9] error, wrapped in [*PathError].
func (vfs *OrefaFS) Chown(name string, uid, gid int) error {
	const op = "chown"

	if vfs.OSType() == avfs.OsWindows {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrWinNotSupported}
	}

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: avfs.ErrNoSuchFileOrDir}
	}

	absPath, _ := vfs.Abs(name)

	vfs.storage.mu.RLock()
	nd, ok := vfs.storage.nodes[absPath]
	vfs.storage.mu.RUnlock()

	if !ok {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrNoSuchFileOrDir}
	}

	nd.mu.Lock()
	nd.setOwner(uid, gid)
	nd.mu.Unlock()

	return nil
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
// A zero [time.Time] value will leave the corresponding file time unchanged.
//
// The underlying filesystem may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) Chtimes(name string, _, mtime time.Time) error {
	const op = "chtimes"

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchDir}
	}

	absPath, _ := vfs.Abs(name)

	vfs.storage.mu.RLock()
	nd, ok := vfs.storage.nodes[absPath]
	vfs.storage.mu.RUnlock()

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
// On Windows, Clean does not modify the volume name other than to replace
// occurrences of "/" with `\`.
// For example, Clean("//host/share/../x") returns `\\host\share\x`.
//
// See also Rob Pike, "Lexical File Names in Plan 9 or
// Getting Dot-Dot Right,"
// https://9p.io/sys/doc/lexnames.html
func (vfs *OrefaFS) Clean(path string) string {
	return vfs.userDir.Clean(path)
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0o666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode [O_RDWR].
// The directory containing the file must already exist.
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) Create(name string) (avfs.File, error) {
	return avfs.Create(vfs, name)
}

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// The file is created with mode 0o600 (before umask).
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by [TempDir].
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (vfs *OrefaFS) CreateTemp(dir, pattern string) (avfs.File, error) {
	return avfs.CreateTemp(vfs, dir, pattern)
}

// CurDir returns the current directory.
func (vfs *OrefaFS) CurDir() string {
	return vfs.userDir.CurDir()
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls [Clean] on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
//
// On Windows, given a volume-only name such as "C:", Dir returns "C:.",
// the current directory on drive C. To obtain the drive's root "C:\",
// use [VolumeName] combined with a separator.
func (vfs *OrefaFS) Dir(path string) string {
	return vfs.userDir.Dir(path)
}

// DirMode returns the default file mode for new directories.
func (vfs *OrefaFS) DirMode() fs.FileMode {
	return vfs.userDir.DirMode()
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls [Clean] on the result.
func (vfs *OrefaFS) EvalSymlinks(path string) (string, error) {
	op := "lstat"
	if vfs.OSType() == avfs.OsWindows {
		op = "CreateFile"
	}

	return "", &fs.PathError{Op: op, Path: path, Err: vfs.err.PermDenied}
}

// Features returns the set of features provided by the file system.
func (vfs *OrefaFS) Features() avfs.Features {
	return vfs.userDir.Features()
}

// FileMode returns the default file mode for new files.
func (vfs *OrefaFS) FileMode() fs.FileMode {
	return vfs.userDir.FileMode()
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
//
// See also the Localize function, which converts a slash-separated path
// as used by the io/fs package to an operating system path.
func (vfs *OrefaFS) FromSlash(path string) string {
	return vfs.userDir.FromSlash(path)
}

// Getwd returns an absolute path name corresponding to the current directory.
func (vfs *OrefaFS) Getwd() (string, error) {
	return vfs.userDir.Getwd()
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in [Match]. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the [Separator] is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is [ErrBadPattern], when pattern
// is malformed.
func (vfs *OrefaFS) Glob(pattern string) (matches []string, err error) {
	return avfs.Glob(vfs, pattern)
}

// HasFeature returns true if the file system provides a given feature.
func (vfs *OrefaFS) HasFeature(feature avfs.Features) bool {
	return vfs.userDir.HasFeature(feature)
}

// Idm returns the identity manager of the file system.
func (vfs *OrefaFS) Idm() avfs.IdmMgr {
	return vfs.userDir.Idm()
}

// InitIdm sets the identity manager of the file system.
// It must be called once, during construction.
func (vfs *OrefaFS) InitIdm(idm avfs.IdmMgr) error {
	return vfs.userDir.InitIdm(idm)
}

// InitOSType sets the OS type of the file system.
// It must be called once, during construction.
func (vfs *OrefaFS) InitOSType(ost avfs.OSType) error {
	return vfs.userDir.InitOSType(ost)
}

// IsAbs reports whether the path is absolute.
func (vfs *OrefaFS) IsAbs(path string) bool {
	return vfs.userDir.IsAbs(path)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *OrefaFS) IsPathSeparator(c uint8) bool {
	return vfs.userDir.IsPathSeparator(c)
}

// Join joins any number of path elements into a single path,
// separating them with an OS specific [Separator]. Empty elements
// are ignored. The result is Cleaned. However, if the argument
// list is empty or all its elements are empty, Join returns
// an empty string.
// On Windows, the result will only be a UNC path if the first
// non-empty element is a UNC path.
func (vfs *OrefaFS) Join(elem ...string) string {
	return vfs.userDir.Join(elem...)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type [*PathError].
//
// On Windows, it always returns the [syscall.EWINDOWS] error, wrapped
// in [*PathError].
func (vfs *OrefaFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	if vfs.OSType() == avfs.OsWindows {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.OpNotPermitted}
	}

	absPath, _ := vfs.Abs(name)

	vfs.storage.mu.RLock()
	nd, ok := vfs.storage.nodes[absPath]
	vfs.storage.mu.RUnlock()

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

	if oldname == "" || newname == "" {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.NoSuchDir}
	}

	oAbsPath, _ := vfs.Abs(oldname)
	nAbsPath, _ := vfs.Abs(newname)

	nDirName, nFileName := avfs.SplitAbs(vfs, nAbsPath)

	vfs.storage.mu.RLock()
	oChild, oChildOk := vfs.storage.nodes[oAbsPath]
	_, nChildOk := vfs.storage.nodes[nAbsPath]
	nParent, nParentOk := vfs.storage.nodes[nDirName]
	vfs.storage.mu.RUnlock()

	if !oChildOk {
		err := vfs.err.NoSuchFile

		if vfs.OSType() == avfs.OsWindows {
			oDirName, _ := avfs.SplitAbs(vfs, oAbsPath)

			vfs.storage.mu.RLock()
			_, oParentOk := vfs.storage.nodes[oDirName]
			vfs.storage.mu.RUnlock()

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

	vfs.storage.mu.Lock()
	vfs.storage.nodes[nAbsPath] = oChild
	vfs.storage.mu.Unlock()

	nParent.addChild(nFileName, oChild)

	oChild.nlink++

	return nil
}

// Lstat returns a [FileInfo] describing the named file.
// If the file is a symbolic link, the returned [FileInfo]
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type [*PathError].
//
// On Windows, if the file is a reparse point that is a surrogate for another
// named entity (such as a symbolic link or mounted folder), the returned
// [FileInfo] describes the reparse point, and makes no attempt to resolve it.
func (vfs *OrefaFS) Lstat(name string) (fs.FileInfo, error) {
	var op string

	if name == "" {
		switch vfs.OSType() {
		case avfs.OsWindows:
			op = "Lstat"
		default:
			op = "lstat"
		}

		return nil, &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchDir}
	}

	switch vfs.OSType() {
	case avfs.OsWindows:
		op = avfs.OpWinCreateFile
	default:
		op = "lstat"
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
//		'\\' c      matches character c (except on Windows)
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c (except on Windows)
//		lo '-' hi   matches character c for lo <= c <= hi
//
// Path segments in the pattern must be separated by [Separator].
//
// Match requires pattern to match all of name, not just a substring.
// The only possible returned error is [ErrBadPattern], when pattern
// is malformed.
//
// On Windows, escaping is disabled. Instead, '\\' is treated as
// path separator.
func (vfs *OrefaFS) Match(pattern, name string) (bool, error) {
	return vfs.userDir.Match(pattern, name)
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) Mkdir(name string, perm fs.FileMode) error {
	const op = "mkdir"

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchDir}
	}

	absPath, _ := vfs.Abs(name)
	dirName, fileName := avfs.SplitAbs(vfs, absPath)

	vfs.storage.mu.Lock()
	defer vfs.storage.mu.Unlock()

	_, childOk := vfs.storage.nodes[absPath]
	parent, parentOk := vfs.storage.nodes[dirName]

	if childOk {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.FileExists}
	}

	if !parentOk {
		for !parentOk {
			dirName, _ = avfs.SplitAbs(vfs, dirName)
			parent, parentOk = vfs.storage.nodes[dirName]
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

// MkdirAll creates a directory named path,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *OrefaFS) MkdirAll(path string, perm fs.FileMode) error {
	const op = "mkdir"

	if path == "" {
		return &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchDir}
	}

	absPath, _ := vfs.Abs(path)

	vfs.storage.mu.Lock()
	defer vfs.storage.mu.Unlock()

	child, childOk := vfs.storage.nodes[absPath]
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
		nd, ok := vfs.storage.nodes[dirName]
		if ok {
			parent = nd
			if !parent.mode.IsDir() {
				return &fs.PathError{Op: op, Path: dirName, Err: vfs.err.NotADirectory}
			}

			break
		}

		ds = append(ds, dirName)

		dirName, _ = avfs.SplitAbs(vfs, dirName)
	}

	for _, absPath = range ds {
		_, fileName := avfs.SplitAbs(vfs, absPath)

		parent = vfs.createDir(parent, absPath, fileName, perm)
	}

	return nil
}

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// The directory is created with mode 0o700 (before umask).
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by [TempDir].
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (vfs *OrefaFS) MkdirTemp(dir, pattern string) (string, error) {
	return avfs.MkdirTemp(vfs, dir, pattern)
}

// OSType returns the operating system type of the file system.
func (vfs *OrefaFS) OSType() avfs.OSType {
	return vfs.userDir.OSType()
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode [O_RDONLY].
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) Open(name string) (avfs.File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask);
// the containing directory must exist. If successful,
// methods on the returned File can be used for I/O.
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) OpenFile(name string, flag int, perm fs.FileMode) (avfs.File, error) {
	op := "open"

	if name == "" {
		return (*OrefaFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
	}

	at := int64(0)
	om := avfs.ToOpenMode(flag)

	absPath, _ := vfs.Abs(name)
	dirName, fileName := avfs.SplitAbs(vfs, absPath)

	vfs.storage.mu.RLock()
	parent, parentOk := vfs.storage.nodes[dirName]
	child, childOk := vfs.storage.nodes[absPath]
	vfs.storage.mu.RUnlock()

	if !childOk {
		if !parentOk {
			return (*OrefaFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchDir}
		}

		if !parent.mode.IsDir() {
			return (*OrefaFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.NotADirectory}
		}

		if om&avfs.OpenCreate == 0 {
			return (*OrefaFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
		}

		if om&avfs.OpenWrite == 0 {
			return (*OrefaFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.PermDenied}
		}

		vfs.storage.mu.Lock()
		defer vfs.storage.mu.Unlock()

		// test for race conditions when opening file in exclusive mode.
		_, childOk = vfs.storage.nodes[absPath]
		if childOk && om&avfs.OpenCreateExcl != 0 {
			return (*OrefaFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.FileExists}
		}

		child = vfs.createFile(parent, absPath, fileName, perm)
	} else {
		if child.mode.IsDir() {
			if om&avfs.OpenWrite != 0 {
				return (*OrefaFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.IsADirectory}
			}
		} else {
			if om&avfs.OpenCreateExcl != 0 {
				return (*OrefaFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.FileExists}
			}

			if om&avfs.OpenDir != 0 {
				return (*OrefaFile)(nil), &fs.PathError{Op: op, Path: name, Err: vfs.err.NotADirectory}
			}

			if om&avfs.OpenTruncate != 0 {
				child.mu.Lock()
				child.truncate(0)
				child.mu.Unlock()
			}

			if om&avfs.OpenAppend != 0 {
				at = child.Size()
			}
		}
	}

	f := &OrefaFile{
		vfs:      vfs,
		nd:       child,
		openMode: om,
		name:     name,
		at:       at,
	}

	return f, nil
}

// PathSeparator returns the OS-specific path separator.
func (vfs *OrefaFS) PathSeparator() uint8 {
	return vfs.userDir.PathSeparator()
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *OrefaFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return avfs.ReadDir(vfs, name)
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) ReadFile(name string) ([]byte, error) {
	return avfs.ReadFile(vfs, name)
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type [*PathError].
//
// If the link destination is relative, Readlink returns the relative path
// without resolving it to an absolute one.
func (vfs *OrefaFS) Readlink(name string) (string, error) {
	const op = "readlink"

	var err error

	switch vfs.OSType() {
	case avfs.OsWindows:
		err = avfs.ErrWinNotReparsePoint
	default:
		err = avfs.ErrPermDenied
	}

	return "", &fs.PathError{Op: op, Path: name, Err: err}
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// [Join](basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
//
// The returned path will always be relative to basepath, even if basepath and
// targpath share no elements. Rel calls [Clean] on the result.
//
// An error is returned if targpath can't be made relative to basepath
// or if knowing the current working directory would be necessary to compute it.
func (vfs *OrefaFS) Rel(basepath, targpath string) (string, error) {
	return vfs.userDir.Rel(basepath, targpath)
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) Remove(name string) error {
	const op = "remove"

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchDir}
	}

	absPath, _ := vfs.Abs(name)
	dirName, fileName := avfs.SplitAbs(vfs, absPath)

	vfs.storage.mu.Lock()
	defer vfs.storage.mu.Unlock()

	child, childOk := vfs.storage.nodes[absPath]
	parent, parentOk := vfs.storage.nodes[dirName]

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
	delete(vfs.storage.nodes, absPath)

	return nil
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) RemoveAll(path string) error {
	if path == "" {
		// fail silently to retain compatibility with previous behavior of RemoveAll.
		return nil
	}

	absPath, _ := vfs.Abs(path)
	dirName, fileName := avfs.SplitAbs(vfs, absPath)

	vfs.storage.mu.Lock()
	defer vfs.storage.mu.Unlock()

	child, childOk := vfs.storage.nodes[absPath]
	parent, parentOk := vfs.storage.nodes[dirName]

	if !childOk || !parentOk {
		return nil
	}

	if child.mode.IsDir() {
		vfs.removeAll(absPath, child)
	}

	child.remove()

	delete(parent.children, fileName)
	delete(vfs.storage.nodes, absPath)

	return nil
}

// Rename renames (moves) oldname to newname.
// If newname already exists and is not a directory, Rename replaces it.
// If newname already exists and is a directory, Rename returns an error.
// OS-specific restrictions may apply when oldname and newname are in different directories.
// Even within the same directory, on non-Unix platforms Rename is not an atomic operation.
// If there is an error, it will be of type *LinkError.
func (vfs *OrefaFS) Rename(oldname, newname string) error {
	const op = "rename"

	if oldname == "" || newname == "" {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.NoSuchDir}
	}

	oAbsPath, _ := vfs.Abs(oldname)
	nAbsPath, _ := vfs.Abs(newname)

	if oAbsPath == nAbsPath {
		return nil
	}

	oDirName, oFileName := avfs.SplitAbs(vfs, oAbsPath)
	nDirName, nFileName := avfs.SplitAbs(vfs, nAbsPath)

	vfs.storage.mu.RLock()
	oChild, oChildOk := vfs.storage.nodes[oAbsPath]
	oParent, oParentOk := vfs.storage.nodes[oDirName]
	nChild, nChildOk := vfs.storage.nodes[nAbsPath]
	nParent, nParentOk := vfs.storage.nodes[nDirName]
	vfs.storage.mu.RUnlock()

	if !oChildOk || !oParentOk || !nParentOk {
		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.NoSuchFile}
	}

	if (oChild.mode.IsDir() && nChildOk) || (!oChild.mode.IsDir() && nChildOk && nChild.mode.IsDir()) {
		err := vfs.err.FileExists
		if vfs.OSType() == avfs.OsWindows {
			err = avfs.ErrWinAccessDenied
		}

		return &os.LinkError{Op: op, Old: oldname, New: newname, Err: err}
	}

	nParent.mu.Lock()
	defer nParent.mu.Unlock()

	if nParent != oParent {
		oParent.mu.Lock()
		defer oParent.mu.Unlock()
	}

	nParent.children[nFileName] = oChild

	delete(oParent.children, oFileName)

	vfs.storage.mu.Lock()
	defer vfs.storage.mu.Unlock()

	vfs.storage.nodes[nAbsPath] = oChild
	delete(vfs.storage.nodes, oAbsPath)

	if oChild.mode.IsDir() {
		oRoot := oAbsPath + string(vfs.PathSeparator())

		for absPath, node := range vfs.storage.nodes {
			if strings.HasPrefix(absPath, oRoot) {
				nPath := nAbsPath + absPath[len(oAbsPath):]
				vfs.storage.nodes[nPath] = node

				delete(vfs.storage.nodes, absPath)
			}
		}
	}

	return nil
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's [Stat].
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

// SetCurDir sets the current directory.
//
// Unlike Chdir, it resolves nothing and checks no permission: the caller is
// responsible for having authorized the directory beforehand.
func (vfs *OrefaFS) SetCurDir(curDir string) error {
	return vfs.userDir.SetCurDir(curDir)
}

// SetFeatures sets the features of the file system.
func (vfs *OrefaFS) SetFeatures(feature avfs.Features) error {
	return vfs.userDir.SetFeatures(feature)
}

// SetUMask sets the file mode creation mask.
func (vfs *OrefaFS) SetUMask(mask fs.FileMode) error {
	return vfs.userDir.SetUMask(mask)
}

// Split splits path immediately following the final [Separator],
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *OrefaFS) Split(path string) (dir, file string) {
	return vfs.userDir.Split(path)
}

// Stat returns a [FileInfo] describing the named file.
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) Stat(path string) (fs.FileInfo, error) {
	var op string

	if path == "" {
		switch vfs.OSType() {
		case avfs.OsWindows:
			op = "Stat"
		default:
			op = "stat"
		}

		return nil, &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchDir}
	}

	switch vfs.OSType() {
	case avfs.OsWindows:
		op = avfs.OpWinCreateFile
	default:
		op = "stat"
	}

	return vfs.stat(path, op)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (vfs *OrefaFS) Sub(dir string) (avfs.VFS, error) {
	const op = "sub"

	return nil, &fs.PathError{Op: op, Path: dir, Err: vfs.err.PermDenied}
}

// Symlink creates newname as a symbolic link to oldname.
// On Windows, a symlink to a non-existent oldname creates a file symlink;
// if oldname is later created as a directory the symlink will not work.
// If there is an error, it will be of type *LinkError.
func (vfs *OrefaFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: vfs.err.PermDenied}
}

// TempDir returns the default directory to use for temporary files.
func (vfs *OrefaFS) TempDir() string {
	return vfs.userDir.TempDir()
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *OrefaFS) ToSlash(path string) string {
	return vfs.userDir.ToSlash(path)
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that
// implements interface avfs.SysStater.
func (vfs *OrefaFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return vfs.userDir.ToSysStat(info)
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type [*PathError].
func (vfs *OrefaFS) Truncate(name string, size int64) error {
	var op string

	switch vfs.OSType() {
	case avfs.OsWindows:
		op = "open"
	default:
		op = "truncate"
	}

	if name == "" {
		return &fs.PathError{Op: op, Path: "", Err: vfs.err.NoSuchFile}
	}

	absPath, _ := vfs.Abs(name)

	vfs.storage.mu.RLock()
	child, childOk := vfs.storage.nodes[absPath]
	vfs.storage.mu.RUnlock()

	if !childOk {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.NoSuchFile}
	}

	if child.mode.IsDir() {
		return &fs.PathError{Op: op, Path: name, Err: vfs.err.IsADirectory}
	}

	if size < 0 {
		if vfs.OSType() == avfs.OsWindows {
			op = "truncate"
		}

		return &fs.PathError{Op: op, Path: name, Err: vfs.err.InvalidArgument}
	}

	child.mu.Lock()
	child.truncate(size)
	child.mu.Unlock()

	return nil
}

// UMask returns the file mode creation mask.
func (vfs *OrefaFS) UMask() fs.FileMode {
	return vfs.userDir.UMask()
}

// User returns the current user.
func (vfs *OrefaFS) User() avfs.UserReader {
	return vfs.userDir.User()
}

// UserHomeDir returns the current user's home directory.
func (vfs *OrefaFS) UserHomeDir() (string, error) {
	return vfs.userDir.UserHomeDir()
}

// VolumeName returns leading volume name.
// Given "C:\foo\bar" it returns "C:" on Windows.
// Given "\\host\share\foo" it returns "\\host\share".
// On other platforms it returns "".
func (vfs *OrefaFS) VolumeName(path string) string {
	return vfs.userDir.VolumeName(path)
}

// VolumeNameLen returns the length of the leading volume name on Windows.
// It returns 0 elsewhere.
func (vfs *OrefaFS) VolumeNameLen(path string) int {
	return vfs.userDir.VolumeNameLen(path)
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
//
// WalkDir calls fn with paths that use the separator character appropriate
// for the operating system. This is unlike [io/fs.WalkDir], which always
// uses slash separated paths.
func (vfs *OrefaFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return avfs.WalkDir(vfs, root, fn)
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
// Since WriteFile requires multiple system calls to complete, a failure mid-operation
// can leave the file in a partially written state.
func (vfs *OrefaFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return avfs.WriteFile(vfs, name, data, perm)
}

func (vfs *OrefaFS) removeAll(absPath string, rootNode *node) {
	if rootNode.mode.IsDir() {
		for fileName, nd := range rootNode.children {
			path := absPath + string(vfs.PathSeparator()) + fileName

			vfs.removeAll(path, nd)
		}
	}

	rootNode.remove()
	delete(vfs.storage.nodes, absPath)
}

// stat is the internal function used by Stat and Lstat.
func (vfs *OrefaFS) stat(path, op string) (fs.FileInfo, error) {
	absPath, _ := vfs.Abs(path)
	dirName, fileName := avfs.SplitAbs(vfs, absPath)

	vfs.storage.mu.RLock()
	child, childOk := vfs.storage.nodes[absPath]
	vfs.storage.mu.RUnlock()

	if !childOk {
		vfs.storage.mu.RLock()
		parent, parentOk := vfs.storage.nodes[dirName]
		vfs.storage.mu.RUnlock()

		if !parentOk {
			return nil, &fs.PathError{Op: op, Path: path, Err: vfs.err.NoSuchDir}
		}

		if parent.mode.IsDir() {
			return nil, &fs.PathError{Op: op, Path: path, Err: vfs.err.NoSuchFile}
		}

		return nil, &fs.PathError{Op: op, Path: path, Err: vfs.err.NotADirectory}
	}

	fst := child.fillStatFrom(fileName)

	return fst, nil
}
