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
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Utils are utilities functions used by some file system.
type Utils struct {
	// OSType defines the operating system type.
	osType OSType

	// pathSeparator is the OS-specific path separator.
	pathSeparator uint8
}

func NewUtils(osType OSType) Utils {
	sep := PathSeparator
	if osType == OsWindows {
		sep = PathSeparatorWin
	}

	vu := Utils{
		osType:        osType,
		pathSeparator: uint8(sep),
	}

	return vu
}

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (vu *Utils) Abs(vfs VFS, path string) (string, error) {
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
func (vu *Utils) Base(vfs VFS, path string) string {
	if path == "" {
		return "."
	}

	// Strip trailing slashes.
	for len(path) > 0 && vfs.IsPathSeparator(path[len(path)-1]) {
		path = path[0 : len(path)-1]
	}

	// Throw away volume name
	path = path[len(vu.VolumeName(path)):]

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
		return string(vu.pathSeparator)
	}

	return path
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
func (vu *Utils) Clean(path string) string {
	originalPath := path
	volLen := vu.volumeNameLen(path)

	path = path[volLen:]
	if path == "" {
		if volLen > 1 && originalPath[1] != ':' {
			// should be UNC
			return vu.FromSlash(originalPath)
		}

		return originalPath + "."
	}

	rooted := vu.IsPathSeparator(path[0])

	// Invariants:
	//	reading from path; r is index of next byte to process.
	//	writing to buf; w is index of next byte to write.
	//	dotdot is index in buf where .. must stop, either because
	//		it is the leading slash or it is a leading ../../.. prefix.
	n := len(path)
	out := lazybuf{path: path, volAndPath: originalPath, volLen: volLen}
	r, dotdot := 0, 0

	if rooted {
		out.append(vu.pathSeparator)

		r, dotdot = 1, 1
	}

	for r < n {
		switch {
		case vu.IsPathSeparator(path[r]):
			// empty path element
			r++
		case path[r] == '.' && (r+1 == n || vu.IsPathSeparator(path[r+1])):
			// . element
			r++
		case path[r] == '.' && path[r+1] == '.' && (r+2 == n || vu.IsPathSeparator(path[r+2])):
			// .. element: remove to last separator
			r += 2

			switch {
			case out.w > dotdot:
				// can backtrack
				out.w--
				for out.w > dotdot && !vu.IsPathSeparator(out.index(out.w)) {
					out.w--
				}
			case !rooted:
				// cannot backtrack, but not rooted, so append .. element.
				if out.w > 0 {
					out.append(vu.pathSeparator)
				}

				out.append('.')
				out.append('.')
				dotdot = out.w
			}
		default:
			// real path element.
			// add slash if needed
			if rooted && out.w != 1 || !rooted && out.w != 0 {
				out.append(vu.pathSeparator)
			}
			// copy element
			for ; r < n && !vu.IsPathSeparator(path[r]); r++ {
				out.append(path[r])
			}
		}
	}

	// Turn empty string into "."
	if out.w == 0 {
		out.append('.')
	}

	return vu.FromSlash(out.string())
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (vu *Utils) Create(vfs VFS, name string) (File, error) {
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
func (vu *Utils) CreateTemp(vfs VFS, dir, pattern string) (File, error) {
	const op = "createtemp"

	if dir == "" {
		dir = vfs.TempDir()
	}

	prefix, suffix, err := vu.prefixAndSuffix(pattern)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: pattern, Err: err}
	}

	prefix = vfs.Join(dir, prefix)

	try := 0

	for {
		name := prefix + vu.nextRandom() + suffix

		f, err := vfs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if vfs.IsExist(err) {
			try++
			if try < 10000 {
				continue
			}

			return nil, &fs.PathError{Op: op, Path: dir + string(vu.pathSeparator) + prefix + "*" + suffix, Err: fs.ErrExist}
		}

		return f, err
	}
}

// CopyFile copies a file between file systems and returns the hash sum of the source file.
func (vu *Utils) CopyFile(dstFs, srcFs VFS, dstPath, srcPath string, hasher hash.Hash) (sum []byte, err error) {
	src, err := srcFs.Open(srcPath)
	if err != nil {
		return nil, err
	}

	defer src.Close()

	dst, err := dstFs.Create(dstPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		cerr := dst.Close()
		if cerr == nil {
			err = cerr
		}
	}()

	var out io.Writer

	if hasher == nil {
		out = dst
	} else {
		hasher.Reset()
		out = io.MultiWriter(dst, hasher)
	}

	_, err = copyBufPool(out, src)
	if err != nil {
		return nil, err
	}

	err = dst.Sync()
	if err != nil {
		return nil, err
	}

	info, err := srcFs.Stat(srcPath)
	if err != nil {
		return nil, err
	}

	err = dstFs.Chmod(dstPath, info.Mode())
	if err != nil {
		return nil, err
	}

	if hasher == nil {
		return nil, nil
	}

	return hasher.Sum(nil), nil
}

// Dir returns all but the last element of path, typically the path's directory.
// After dropping the final element, Dir calls Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (vu *Utils) Dir(path string) string {
	vol := vu.VolumeName(path)

	i := len(path) - 1
	for i >= len(vol) && !vu.IsPathSeparator(path[i]) {
		i--
	}

	dir := vu.Clean(path[len(vol) : i+1])
	if dir == "." && len(vol) > 2 {
		// must be UNC
		return vol
	}

	return vol + dir
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (vu *Utils) FromSlash(path string) string {
	if vu.osType != OsWindows {
		return path
	}

	return strings.ReplaceAll(path, "/", string(vu.pathSeparator))
}

// HashFile hashes a file and returns the hash sum.
func (vu *Utils) HashFile(vfs VFS, name string, hasher hash.Hash) (sum []byte, err error) {
	f, err := vfs.Open(name)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	hasher.Reset()

	_, err = copyBufPool(hasher, f)
	if err != nil {
		return nil, err
	}

	sum = hasher.Sum(nil)

	return sum, nil
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (vu *Utils) Glob(vfs VFS, pattern string) (matches []string, err error) {
	// Check pattern is well-formed.
	if _, err = vu.Match(pattern, ""); err != nil {
		return nil, err
	}

	if !vu.hasMeta(pattern) {
		if _, err = vfs.Lstat(pattern); err != nil {
			return nil, nil
		}

		return []string{pattern}, nil
	}

	dir, file := vfs.Split(pattern)
	volumeLen := 0

	if vu.osType == OsWindows {
		volumeLen, dir = vu.cleanGlobPathWindows(dir)
	} else {
		dir = vu.cleanGlobPath(dir)
	}

	if !vu.hasMeta(dir[volumeLen:]) {
		return vu.glob(vfs, dir, file, nil)
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
		matches, err = vu.glob(vfs, d, file, matches)
		if err != nil {
			return
		}
	}

	return //nolint:nakedret // Adapted from standard library.
}

// IsAbs reports whether the path is absolute.
func (vu *Utils) IsAbs(path string) bool {
	if vu.osType != OsWindows {
		return strings.HasPrefix(path, "/")
	}

	if isReservedName(path) {
		return true
	}

	l := vu.volumeNameLen(path)
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
func (vu *Utils) IsExist(err error) bool {
	return errors.Is(err, ErrFileExists)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (vu *Utils) IsNotExist(err error) bool {
	return errors.Is(err, ErrNoSuchFileOrDir)
}

// IsPathSeparator reports whether c is a directory separator character.
func (vu *Utils) IsPathSeparator(c uint8) bool {
	if vu.osType != OsWindows {
		return PathSeparator == c
	}

	return c == '\\' || c == '/'
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (vu *Utils) Join(elem ...string) string {
	// If there's a bug here, fix the logic in ./path_plan9.go too.
	for i, e := range elem {
		if e != "" {
			return vu.Clean(strings.Join(elem[i:], string(vu.pathSeparator)))
		}
	}

	return ""
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
func (vu *Utils) Match(pattern, name string) (matched bool, err error) {
Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string

		star, chunk, pattern = vu.scanChunk(pattern)
		if star && chunk == "" {
			// Trailing * matches rest of string unless it has a /.
			return !strings.Contains(name, string(vu.pathSeparator)), nil
		}

		// Look for match at current position.
		t, ok, err := vu.matchChunk(chunk, name)

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
			for i := 0; i < len(name) && name[i] != vu.pathSeparator; i++ {
				t, ok, err := vu.matchChunk(chunk, name[i+1:])
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

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (vu *Utils) MkdirTemp(vfs VFS, dir, pattern string) (string, error) {
	const op = "mkdirtemp"

	if dir == "" {
		dir = vfs.TempDir()
	}

	prefix, suffix, err := vu.prefixAndSuffix(pattern)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: pattern, Err: err}
	}

	prefix = vu.Join(dir, prefix)
	try := 0

	for {
		name := prefix + vu.nextRandom() + suffix
		err = vfs.Mkdir(name, 0o700)

		if err == nil {
			return name, nil
		}

		if vfs.IsExist(err) {
			try++
			if try < 10000 {
				continue
			}

			return "", &fs.PathError{Op: op, Path: dir + string(vu.pathSeparator) + prefix + "*" + suffix, Err: fs.ErrExist}
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
func (vu *Utils) Open(vfs VFS, name string) (File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// PathSeparator return the OS-specific path separator.
func (vu *Utils) PathSeparator() uint8 {
	return vu.pathSeparator
}

// OSType returns the operating system type of the file system.
func (vu *Utils) OSType() OSType {
	return vu.osType
}

// SegmentPath segments string key paths by separator (using avfs.PathSeparator).
// For example with path = "/a/b/c" it will return in successive calls :
//
// "a", "/b/c"
// "b", "/c"
// "c", ""
//
// 	for start, end, isLast := 1, 0, len(path) <= 1; !isLast; start = end + 1 {
//		end, isLast = avfs.SegmentPath(path, start)
//		fmt.Println(path[start:end], path[end:])
//	}
//
func (vu *Utils) SegmentPath(path string, start int) (end int, isLast bool) {
	pos := strings.IndexRune(path[start:], rune(vu.pathSeparator))
	if pos != -1 {
		return start + pos, false
	}

	return len(path), true
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (vu *Utils) ReadDir(vfs VFS, name string) ([]fs.DirEntry, error) {
	f, err := vfs.Open(name)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	dirs, err := f.ReadDir(-1)

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })

	return dirs, err
}

// ReadFile reads the named file and returns the contents.
// A successful call returns err == nil, not err == EOF.
// Because ReadFile reads the whole file, it does not treat an EOF from Read
// as an error to be reported.
func (vu *Utils) ReadFile(vfs VFS, name string) ([]byte, error) {
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
func (vu *Utils) Rel(basepath, targpath string) (string, error) {
	baseVol := vu.VolumeName(basepath)
	targVol := vu.VolumeName(targpath)
	base := vu.Clean(basepath)
	targ := vu.Clean(targpath)

	if sameWord(targ, base) {
		return ".", nil
	}

	base = base[len(baseVol):]
	targ = targ[len(targVol):]

	if base == "." {
		base = ""
	} else if base == "" && vu.volumeNameLen(baseVol) > 2 /* isUNC */ {
		// Treat any targetpath matching `\\host\share` basepath as absolute path.
		base = string(vu.pathSeparator)
	}

	// Can't use IsAbs - `\a` and `a` are both relative in Windows.
	baseSlashed := len(base) > 0 && base[0] == vu.pathSeparator
	targSlashed := len(targ) > 0 && targ[0] == vu.pathSeparator

	if baseSlashed != targSlashed || !sameWord(baseVol, targVol) {
		return "", errors.New("Rel: can't make " + targpath + " relative to " + basepath)
	}

	// Position base[b0:bi] and targ[t0:ti] at the first differing elements.
	bl := len(base)
	tl := len(targ)

	var b0, bi, t0, ti int

	for {
		for bi < bl && base[bi] != vu.pathSeparator {
			bi++
		}

		for ti < tl && targ[ti] != vu.pathSeparator {
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
		seps := strings.Count(base[b0:bl], string(vu.pathSeparator))
		size := 2 + seps*3

		if tl != t0 {
			size += 1 + tl - t0
		}

		buf := make([]byte, size)
		n := copy(buf, "..")

		for i := 0; i < seps; i++ {
			buf[n] = vu.pathSeparator
			copy(buf[n+1:], "..")
			n += 3
		}

		if t0 != tl {
			buf[n] = vu.pathSeparator
			copy(buf[n+1:], targ[t0:])
		}

		return string(buf), nil
	}

	return targ[t0:], nil
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (vu *Utils) Split(path string) (dir, file string) {
	vol := vu.VolumeName(path)

	i := len(path) - 1
	for i >= len(vol) && !vu.IsPathSeparator(path[i]) {
		i--
	}

	return path[:i+1], path[i+1:]
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
func (vu *Utils) TempDir() string {
	return TmpDir
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (vu *Utils) ToSlash(path string) string {
	if vu.pathSeparator == '/' {
		return path
	}

	return strings.ReplaceAll(path, string(vu.pathSeparator), "/")
}

// VolumeName returns leading volume name.
// Given "C:\foo\bar" it returns "C:" on Windows.
// Given "\\host\share\foo" it returns "\\host\share".
// On other platforms it returns "".
func (vu *Utils) VolumeName(path string) string {
	return path[:vu.volumeNameLen(path)]
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
func (vu *Utils) WalkDir(vfs VFS, root string, fn fs.WalkDirFunc) error {
	info, err := vfs.Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = vu.walkDir(vfs, root, &statDirEntry{info}, fn)
	}

	if err == filepath.SkipDir {
		return nil
	}

	return err
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
func (vu *Utils) WriteFile(vfs VFS, name string, data []byte, perm fs.FileMode) error {
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

// CreateBaseDirs creates base directories on a file system.
func CreateBaseDirs(vfs VFS, basePath string) error {
	for _, dir := range BaseDirs {
		path := vfs.Join(basePath, dir.Path)

		err := vfs.Mkdir(path, dir.Perm)
		if err != nil {
			return err
		}

		err = vfs.Chmod(path, dir.Perm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateHomeDir creates the home directory of a user.
func CreateHomeDir(vfs VFS, u UserReader) (UserReader, error) {
	userDir := vfs.Join(HomeDir, u.Name())

	err := vfs.Mkdir(userDir, HomeDirPerm)
	if err != nil {
		return nil, err
	}

	err = vfs.Chown(userDir, u.Uid(), u.Gid())
	if err != nil {
		return nil, err
	}

	return u, nil
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

// BaseDir is a directory always present in a file system.
type BaseDir struct {
	Path string
	Perm fs.FileMode
}

// BaseDirs are the base directories present in a file system.
var BaseDirs = []BaseDir{ //nolint:gochecknoglobals // Used by CreateBaseDirs and TestCreateBaseDirs.
	{Path: HomeDir, Perm: 0o755},
	{Path: RootDir, Perm: 0o700},
	{Path: TmpDir, Perm: 0o777},
}

type statDirEntry struct {
	info fs.FileInfo
}

func (d *statDirEntry) Name() string               { return d.info.Name() }
func (d *statDirEntry) IsDir() bool                { return d.info.IsDir() }
func (d *statDirEntry) Type() fs.FileMode          { return d.info.Mode().Type() }
func (d *statDirEntry) Info() (fs.FileInfo, error) { return d.info, nil }
