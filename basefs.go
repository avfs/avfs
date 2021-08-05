//
//  Copyright 2021 The AVFS authors
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

package avfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// BaseFS file system functions.

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (vfs *BaseFS) Abs(path string) (string, error) {
	if vfs.IsAbs(path) {
		return vfs.Clean(path), nil
	}

	wd, err := vfs.Getwd()
	if err != nil {
		return "", err
	}

	return vfs.Join(wd, path), nil
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (vfs *BaseFS) Base(path string) string {
	if path == "" {
		return "."
	}

	// Strip trailing slashes.
	for len(path) > 0 && vfs.IsPathSeparator(path[len(path)-1]) {
		path = path[0 : len(path)-1]
	}

	// Throw away volume name
	path = path[len(vfs.VolumeName(path)):]

	// Find the last element
	i := len(path) - 1
	for i >= 0 && !vfs.IsPathSeparator(path[i]) {
		i--
	}

	if i >= 0 {
		path = path[i+1:]
	}

	// If empty now, it had only slashes.
	if path == "" {
		return string(vfs.pathSeparator)
	}

	return path
}

// Chdir changes the current working directory to the named directory.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Chdir(dir string) error {
	const op = "chdir"

	return &fs.PathError{Op: op, Path: dir, Err: ErrPermDenied}
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
func (vfs *BaseFS) Chmod(name string, mode fs.FileMode) error {
	const op = "chmod"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *BaseFS) Chown(name string, uid, gid int) error {
	const op = "chown"

	return &fs.PathError{Op: op, Path: name, Err: ErrOpNotPermitted}
}

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Chroot(path string) error {
	const op = "chroot"

	return &fs.PathError{Op: op, Path: path, Err: ErrOpNotPermitted}
}

// Chtimes changes the access and modification times of the named
// file, similar to the Unix utime() or utimes() functions.
//
// The underlying file system may truncate or round the values to a
// less precise time unit.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Chtimes(name string, atime, mtime time.Time) error {
	const op = "chtimes"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
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
func (vfs *BaseFS) Clean(path string) string {
	originalPath := path
	volLen := vfs.volumeNameLen(path)

	path = path[volLen:]
	if path == "" {
		if volLen > 1 && originalPath[1] != ':' {
			// should be UNC
			return vfs.FromSlash(originalPath)
		}

		return originalPath + "."
	}

	rooted := vfs.IsPathSeparator(path[0])

	// Invariants:
	//	reading from path; r is index of next byte to process.
	//	writing to buf; w is index of next byte to write.
	//	dotdot is index in buf where .. must stop, either because
	//		it is the leading slash or it is a leading ../../.. prefix.
	n := len(path)
	out := lazybuf{path: path, volAndPath: originalPath, volLen: volLen}
	r, dotdot := 0, 0

	if rooted {
		out.append(vfs.pathSeparator)

		r, dotdot = 1, 1
	}

	for r < n {
		switch {
		case vfs.IsPathSeparator(path[r]):
			// empty path element
			r++
		case path[r] == '.' && (r+1 == n || vfs.IsPathSeparator(path[r+1])):
			// . element
			r++
		case path[r] == '.' && path[r+1] == '.' && (r+2 == n || vfs.IsPathSeparator(path[r+2])):
			// .. element: remove to last separator
			r += 2

			switch {
			case out.w > dotdot:
				// can backtrack
				out.w--
				for out.w > dotdot && !vfs.IsPathSeparator(out.index(out.w)) {
					out.w--
				}
			case !rooted:
				// cannot backtrack, but not rooted, so append .. element.
				if out.w > 0 {
					out.append(vfs.pathSeparator)
				}

				out.append('.')
				out.append('.')
				dotdot = out.w
			}
		default:
			// real path element.
			// add slash if needed
			if rooted && out.w != 1 || !rooted && out.w != 0 {
				out.append(vfs.pathSeparator)
			}
			// copy element
			for ; r < n && !vfs.IsPathSeparator(path[r]); r++ {
				out.append(path[r])
			}
		}
	}

	// Turn empty string into "."
	if out.w == 0 {
		out.append('.')
	}

	return vfs.FromSlash(out.string())
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Create(name string) (File, error) {
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
func (vfs *BaseFS) CreateTemp(dir, pattern string) (File, error) {
	const op = "createtemp"

	if dir == "" {
		dir = vfs.TempDir()
	}

	prefix, suffix, err := vfs.prefixAndSuffix(pattern)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: pattern, Err: err}
	}

	prefix = vfs.Join(dir, prefix)

	try := 0

	for {
		name := prefix + vfs.nextRandom() + suffix

		f, err := vfs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if vfs.IsExist(err) {
			try++
			if try < 10000 {
				continue
			}

			return nil, &fs.PathError{Op: op, Path: dir + string(vfs.pathSeparator) + prefix + "*" + suffix, Err: fs.ErrExist}
		}

		return f, err
	}
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vfs *BaseFS) Dir(path string) string {
	vol := vfs.VolumeName(path)

	i := len(path) - 1
	for i >= len(vol) && !vfs.IsPathSeparator(path[i]) {
		i--
	}

	dir := vfs.Clean(path[len(vol) : i+1])
	if dir == "." && len(vol) > 2 {
		// must be UNC
		return vol
	}

	return vol + dir
}

// EvalSymlinks returns the path name after the evaluation of any symbolic
// links.
// If path is relative the result will be relative to the current directory,
// unless one of the components is an absolute symbolic link.
// EvalSymlinks calls Clean on the result.
func (vfs *BaseFS) EvalSymlinks(path string) (string, error) {
	const op = "lstat"

	return "", &fs.PathError{Op: op, Path: path, Err: ErrPermDenied}
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vfs *BaseFS) FromSlash(path string) string {
	if vfs.osType != OsWindows {
		return path
	}

	return strings.ReplaceAll(path, "/", string(vfs.pathSeparator))
}

// GetUMask returns the file mode creation mask.
func (vfs *BaseFS) GetUMask() fs.FileMode {
	return 0o022
}

// Getwd returns a rooted path name corresponding to the
// current directory. If the current directory can be
// reached via multiple paths (due to symbolic links),
// Getwd may return any one of them.
func (vfs *BaseFS) Getwd() (dir string, err error) {
	const op = "getwd"

	return "", &fs.PathError{Op: op, Path: dir, Err: ErrPermDenied}
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (vfs *BaseFS) Glob(pattern string) (matches []string, err error) {
	// Check pattern is well-formed.
	if _, err = vfs.Match(pattern, ""); err != nil {
		return nil, err
	}

	if !vfs.hasMeta(pattern) {
		if _, err = vfs.Lstat(pattern); err != nil {
			return nil, nil
		}

		return []string{pattern}, nil
	}

	dir, file := vfs.Split(pattern)
	volumeLen := 0

	if vfs.osType == OsWindows {
		volumeLen, dir = vfs.cleanGlobPathWindows(dir)
	} else {
		dir = vfs.cleanGlobPath(dir)
	}

	if !vfs.hasMeta(dir[volumeLen:]) {
		return vfs.glob(dir, file, nil)
	}

	// Prevent infinite recursion. See issue 15879.
	if dir == pattern {
		return nil, filepath.ErrBadPattern
	}

	var m []string

	m, err = vfs.Glob(dir)
	if err != nil {
		return
	}

	for _, d := range m {
		matches, err = vfs.glob(d, file, matches)
		if err != nil {
			return
		}
	}

	return
}

// IsAbs reports whether the path is absolute.
func (vfs *BaseFS) IsAbs(path string) bool {
	if vfs.osType != OsWindows {
		return strings.HasPrefix(path, "/")
	}

	if isReservedName(path) {
		return true
	}

	l := vfs.volumeNameLen(path)
	if l == 0 {
		return false
	}

	path = path[l:]
	if path == "" {
		return false
	}

	return isSlash(path[0])
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (vfs *BaseFS) IsExist(err error) bool {
	return errors.Is(err, ErrFileExists)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (vfs *BaseFS) IsNotExist(err error) bool {
	return errors.Is(err, ErrNoSuchFileOrDir)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vfs *BaseFS) IsPathSeparator(c uint8) bool {
	if vfs.osType != OsWindows {
		return PathSeparator == c
	}

	// NOTE: Windows accept / as path separator.
	return c == '\\' || c == '/'
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (vfs *BaseFS) Join(elem ...string) string {
	// If there's a bug here, fix the logic in ./path_plan9.go too.
	for i, e := range elem {
		if e != "" {
			return vfs.Clean(strings.Join(elem[i:], string(vfs.pathSeparator)))
		}
	}

	return ""
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *BaseFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	return &fs.PathError{Op: op, Path: name, Err: ErrOpNotPermitted}
}

// Link creates newname as a hard link to the oldname file.
// If there is an error, it will be of type *LinkError.
func (vfs *BaseFS) Link(oldname, newname string) error {
	const op = "link"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: ErrPermDenied}
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Lstat(name string) (fs.FileInfo, error) {
	const op = "lstat"

	return nil, &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
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
func (vfs *BaseFS) Match(pattern, name string) (matched bool, err error) {
Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string

		star, chunk, pattern = vfs.scanChunk(pattern)
		if star && chunk == "" {
			// Trailing * matches rest of string unless it has a /.
			return !strings.Contains(name, string(vfs.pathSeparator)), nil
		}

		// Look for match at current position.
		t, ok, err := vfs.matchChunk(chunk, name)

		// if we're the last chunk, make sure we've exhausted the name
		// otherwise we'll give a false result even if we could still match
		// using the star
		if ok && (t == "" || len(pattern) > 0) {
			name = t

			continue
		}

		if err != nil {
			return false, err
		}

		if star {
			// Look for match skipping i+1 bytes.
			// Cannot skip /.
			for i := 0; i < len(name) && name[i] != vfs.pathSeparator; i++ {
				t, ok, err := vfs.matchChunk(chunk, name[i+1:])
				if ok {
					// if we're the last chunk, make sure we exhausted the name
					if pattern == "" && len(t) > 0 {
						continue
					}
					name = t

					continue Pattern
				}
				if err != nil {
					return false, err
				}
			}
		}

		return false, nil
	}

	return name == "", nil
}

// Mkdir creates a new directory with the specified name and permission
// bits (before umask).
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Mkdir(name string, perm fs.FileMode) error {
	const op = "mkdir"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// MkdirAll creates a directory named path,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing
// and returns nil.
func (vfs *BaseFS) MkdirAll(path string, perm fs.FileMode) error {
	const op = "mkdir"

	return &fs.PathError{Op: op, Path: path, Err: ErrPermDenied}
}

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (vfs *BaseFS) MkdirTemp(dir, pattern string) (string, error) {
	const op = "mkdirtemp"

	if dir == "" {
		dir = vfs.TempDir()
	}

	prefix, suffix, err := vfs.prefixAndSuffix(pattern)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: pattern, Err: err}
	}

	prefix = vfs.Join(dir, prefix)
	try := 0

	for {
		name := prefix + vfs.nextRandom() + suffix
		err := vfs.Mkdir(name, 0o700)

		if err == nil {
			return name, nil
		}

		if vfs.IsExist(err) {
			try++
			if try < 10000 {
				continue
			}

			return "", &fs.PathError{Op: op, Path: dir + string(vfs.pathSeparator) + prefix + "*" + suffix, Err: fs.ErrExist}
		}

		if vfs.IsNotExist(err) {
			_, err = vfs.Stat(dir)
			if vfs.IsNotExist(err) {
				return "", err
			}
		}

		return "", err
	}
}

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Open(name string) (File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// OpenFile is the generalized open call; most users will use Open
// or Create instead. It opens the named file with specified flag
// (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). If successful,
// methods on the returned DummyFile can be used for I/O.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	const op = "open"

	return &BaseFile{}, &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// PathSeparator return the OS-specific path separator.
func (vfs *BaseFS) PathSeparator() uint8 {
	return vfs.pathSeparator
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vfs *BaseFS) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := vfs.Open(name)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	dirs, err := f.ReadDir(-1)

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	return dirs, err
}

// Readlink returns the destination of the named symbolic link.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Readlink(name string) (string, error) {
	const op = "readlink"

	return "", &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
func (vfs *BaseFS) ReadFile(name string) ([]byte, error) {
	f, err := vfs.Open(name)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var size int

	if info, err := f.Stat(); err == nil {
		size64 := info.Size()
		if int64(int(size64)) == size64 {
			size = int(size64)
		}
	}

	size++ // one byte for final read at EOF

	// If a file claims a small size, read at least 512 bytes.
	// In particular, files in Linux's /proc claim size 0 but
	// then do not work right if read in small pieces,
	// so an initial read of 1 byte would not work correctly.
	if size < 512 {
		size = 512
	}

	data := make([]byte, 0, size)

	for {
		if len(data) >= cap(data) {
			d := append(data[:cap(data)], 0)
			data = d[:len(data)]
		}

		n, err := f.Read(data[len(data):cap(data)])
		data = data[:len(data)+n]

		if err != nil {
			if err == io.EOF {
				err = nil
			}

			return data, err
		}
	}
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// Join(basepath, Rel(basepath, targpath)) is equivalent to targpath itself.
// On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
// An error is returned if targpath can't be made relative to basepath or if
// knowing the current working directory would be necessary to compute it.
// Rel calls Clean on the result.
func (vfs *BaseFS) Rel(basepath, targpath string) (string, error) {
	baseVol := vfs.VolumeName(basepath)
	targVol := vfs.VolumeName(targpath)
	base := vfs.Clean(basepath)
	targ := vfs.Clean(targpath)

	if sameWord(targ, base) {
		return ".", nil
	}

	base = base[len(baseVol):]
	targ = targ[len(targVol):]

	if base == "." {
		base = ""
	} else if base == "" && vfs.volumeNameLen(baseVol) > 2 /* isUNC */ {
		// Treat any targetpath matching `\\host\share` basepath as absolute path.
		base = string(vfs.pathSeparator)
	}

	// Can't use IsAbs - `\a` and `a` are both relative in Windows.
	baseSlashed := len(base) > 0 && base[0] == vfs.pathSeparator
	targSlashed := len(targ) > 0 && targ[0] == vfs.pathSeparator

	if baseSlashed != targSlashed || !sameWord(baseVol, targVol) {
		return "", errors.New("Rel: can't make " + targpath + " relative to " + basepath)
	}

	// Position base[b0:bi] and targ[t0:ti] at the first differing elements.
	bl := len(base)
	tl := len(targ)

	var b0, bi, t0, ti int

	for {
		for bi < bl && base[bi] != vfs.pathSeparator {
			bi++
		}

		for ti < tl && targ[ti] != vfs.pathSeparator {
			ti++
		}

		if !sameWord(targ[t0:ti], base[b0:bi]) {
			break
		}

		if bi < bl {
			bi++
		}

		if ti < tl {
			ti++
		}

		b0 = bi
		t0 = ti
	}

	if base[b0:bi] == ".." {
		return "", errors.New("Rel: can't make " + targpath + " relative to " + basepath)
	}

	if b0 != bl {
		// Base elements left. Must go up before going down.
		seps := strings.Count(base[b0:bl], string(vfs.pathSeparator))
		size := 2 + seps*3

		if tl != t0 {
			size += 1 + tl - t0
		}

		buf := make([]byte, size)
		n := copy(buf, "..")

		for i := 0; i < seps; i++ {
			buf[n] = vfs.pathSeparator
			copy(buf[n+1:], "..")
			n += 3
		}

		if t0 != tl {
			buf[n] = vfs.pathSeparator
			copy(buf[n+1:], targ[t0:])
		}

		return string(buf), nil
	}

	return targ[t0:], nil
}

// Remove removes the named file or (empty) directory.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Remove(name string) error {
	const op = "remove"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) RemoveAll(path string) error {
	const op = "removeall"

	return &fs.PathError{Op: op, Path: path, Err: ErrPermDenied}
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func (vfs *BaseFS) Rename(oldname, newname string) error {
	const op = "rename"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: ErrPermDenied}
}

// SameFile reports whether fi1 and fi2 describe the same file.
// For example, on Unix this means that the device and inode fields
// of the two underlying structures are identical; on other systems
// the decision may be based on the path names.
// SameFile only applies to results returned by this package's Stat.
// It returns false in other cases.
func (vfs *BaseFS) SameFile(fi1, fi2 fs.FileInfo) bool {
	return false
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vfs *BaseFS) Split(path string) (dir, file string) {
	vol := vfs.VolumeName(path)

	i := len(path) - 1
	for i >= len(vol) && !vfs.IsPathSeparator(path[i]) {
		i--
	}

	return path[:i+1], path[i+1:]
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Stat(name string) (fs.FileInfo, error) {
	const op = "stat"

	return nil, &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// Symlink creates newname as a symbolic link to oldname.
// If there is an error, it will be of type *LinkError.
func (vfs *BaseFS) Symlink(oldname, newname string) error {
	const op = "symlink"

	return &os.LinkError{Op: op, Old: oldname, New: newname, Err: ErrPermDenied}
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
func (vfs *BaseFS) TempDir() string {
	return TmpDir
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vfs *BaseFS) ToSlash(path string) string {
	if vfs.pathSeparator == '/' {
		return path
	}

	return strings.ReplaceAll(path, string(vfs.pathSeparator), "/")
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *BaseFS) ToSysStat(info fs.FileInfo) SysStater {
	return &BaseSysStat{}
}

// Truncate changes the size of the named file.
// If the file is a symbolic link, it changes the size of the link's target.
// If there is an error, it will be of type *PathError.
func (vfs *BaseFS) Truncate(name string, size int64) error {
	const op = "truncate"

	return &fs.PathError{Op: op, Path: name, Err: ErrPermDenied}
}

// UMask sets the file mode creation mask.
func (vfs *BaseFS) UMask(mask fs.FileMode) {
	_ = mask
}

// VolumeName returns leading volume name.
// Given "C:\foo\bar" it returns "C:" on Windows.
// Given "\\host\share\foo" it returns "\\host\share".
// On other platforms it returns "".
func (vfs *BaseFS) VolumeName(path string) string {
	return path[:vfs.volumeNameLen(path)]
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
func (vfs *BaseFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	info, err := vfs.Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = vfs.walkDir(root, &statDirEntry{info}, fn)
	}

	if err == filepath.SkipDir {
		return nil
	}

	return err
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
func (vfs *BaseFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	f, err := vfs.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}

	return err
}

// RunTimeOS returns the current Operating System type.
func RunTimeOS() OSType {
	switch runtime.GOOS {
	case "linux":
		return OsLinux
	case "darwin":
		return OsDarwin
	case "windows":
		return OsWindows
	default:
		return OsUnknown
	}
}
