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

// Package vfsutils implements some file system utility functions.
package vfsutils

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/avfs/avfs"
)

// Utils regroups functions used in multiple file systems
type Utils struct {
	// OS-specific path separator
	PathSeparator uint8

	// OSType defines the operating system type.
	OSType avfs.OSType

	// Random number state.
	// We generate random temporary file names so that there's a good
	// chance the file doesn't exist yet - keeps the number of tries in
	// CreateTemp to a minimum.
	randno uint32
	randmu sync.Mutex
}

// NewUtils create and returns à Utils structure.
func NewUtils(osType avfs.OSType) *Utils {
	sep := avfs.PathSeparator
	if osType == avfs.OsWindows {
		sep = avfs.PathSeparatorWin
	}

	ut := &Utils{
		PathSeparator: uint8(sep),
		OSType:        osType,
	}

	return ut
}

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (ut *Utils) Abs(vfs avfs.VFS, path string) (string, error) {
	if ut.IsAbs(path) {
		return ut.Clean(path), nil
	}

	wd, err := vfs.Getwd()
	if err != nil {
		return "", err
	}

	return ut.Join(wd, path), nil
}

// Base returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (ut *Utils) Base(path string) string {
	if path == "" {
		return "."
	}

	// Strip trailing slashes.
	for len(path) > 0 && ut.IsPathSeparator(path[len(path)-1]) {
		path = path[0 : len(path)-1]
	}

	// Throw away volume name
	path = path[len(ut.VolumeName(path)):]

	// Find the last element
	i := len(path) - 1
	for i >= 0 && !ut.IsPathSeparator(path[i]) {
		i--
	}

	if i >= 0 {
		path = path[i+1:]
	}

	// If empty now, it had only slashes.
	if path == "" {
		return string(ut.PathSeparator)
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
func (ut *Utils) Clean(path string) string {
	originalPath := path
	volLen := ut.volumeNameLen(path)

	path = path[volLen:]
	if path == "" {
		if volLen > 1 && originalPath[1] != ':' {
			// should be UNC
			return ut.FromSlash(originalPath)
		}

		return originalPath + "."
	}

	rooted := ut.IsPathSeparator(path[0])

	// Invariants:
	//	reading from path; r is index of next byte to process.
	//	writing to buf; w is index of next byte to write.
	//	dotdot is index in buf where .. must stop, either because
	//		it is the leading slash or it is a leading ../../.. prefix.
	n := len(path)
	out := lazybuf{path: path, volAndPath: originalPath, volLen: volLen}
	r, dotdot := 0, 0

	if rooted {
		out.append(ut.PathSeparator)
		r, dotdot = 1, 1
	}

	for r < n {
		switch {
		case ut.IsPathSeparator(path[r]):
			// empty path element
			r++
		case path[r] == '.' && (r+1 == n || ut.IsPathSeparator(path[r+1])):
			// . element
			r++
		case path[r] == '.' && path[r+1] == '.' && (r+2 == n || ut.IsPathSeparator(path[r+2])):
			// .. element: remove to last separator
			r += 2
			switch {
			case out.w > dotdot:
				// can backtrack
				out.w--
				for out.w > dotdot && !ut.IsPathSeparator(out.index(out.w)) {
					out.w--
				}
			case !rooted:
				// cannot backtrack, but not rooted, so append .. element.
				if out.w > 0 {
					out.append(ut.PathSeparator)
				}
				out.append('.')
				out.append('.')
				dotdot = out.w
			}
		default:
			// real path element.
			// add slash if needed
			if rooted && out.w != 1 || !rooted && out.w != 0 {
				out.append(ut.PathSeparator)
			}
			// copy element
			for ; r < n && !ut.IsPathSeparator(path[r]); r++ {
				out.append(path[r])
			}
		}
	}

	// Turn empty string into "."
	if out.w == 0 {
		out.append('.')
	}

	return ut.FromSlash(out.string())
}

// BaseDir is a directory always present in a file system.
type BaseDir struct {
	Path string
	Perm fs.FileMode
}

// BaseDirs are the base directories present in a file system.
var BaseDirs = []BaseDir{ //nolint:gochecknoglobals // Used by CreateBaseDirs and TestCreateBaseDirs.
	{Path: avfs.HomeDir, Perm: 0o755},
	{Path: avfs.RootDir, Perm: 0o700},
	{Path: avfs.TmpDir, Perm: 0o777},
}

// CreateBaseDirs creates base directories on a file system.
func (ut *Utils) CreateBaseDirs(vfs avfs.VFS, basePath string) error {
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
func (ut *Utils) CreateHomeDir(vfs avfs.VFS, u avfs.UserReader) (avfs.UserReader, error) {
	userDir := vfs.Join(avfs.HomeDir, u.Name())

	err := vfs.Mkdir(userDir, avfs.HomeDirPerm)
	if err != nil {
		return nil, err
	}

	err = vfs.Chown(userDir, u.Uid(), u.Gid())
	if err != nil {
		return nil, err
	}

	return u, nil
}

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (ut *Utils) CreateTemp(vfs avfs.VFS, dir, pattern string) (avfs.File, error) {
	const op = "createtemp"

	if dir == "" {
		dir = vfs.TempDir()
	}

	prefix, suffix, err := ut.prefixAndSuffix(pattern)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: pattern, Err: err}
	}

	prefix = ut.joinPath(dir, prefix)

	try := 0

	for {
		name := prefix + ut.nextRandom() + suffix

		f, err := vfs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if ut.IsExist(err) {
			try++
			if try < 10000 {
				continue
			}

			return nil, &fs.PathError{Op: op, Path: dir + string(ut.PathSeparator) + prefix + "*" + suffix, Err: fs.ErrExist}
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
func (ut *Utils) Dir(path string) string {
	vol := ut.VolumeName(path)

	i := len(path) - 1
	for i >= len(vol) && !ut.IsPathSeparator(path[i]) {
		i--
	}

	dir := ut.Clean(path[len(vol) : i+1])
	if dir == "." && len(vol) > 2 {
		// must be UNC
		return vol
	}

	return vol + dir
}

// FromSlash returns the result of replacing each slash ('/') character
// in path with a separator character. Multiple slashes are replaced
// by multiple separators.
func (ut *Utils) FromSlash(path string) string {
	if ut.PathSeparator == '/' {
		return path
	}

	return strings.ReplaceAll(path, "/", string(ut.PathSeparator))
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (ut *Utils) Glob(vfs avfs.VFS, pattern string) (matches []string, err error) {
	// Check pattern is well-formed.
	if _, err := ut.Match(pattern, ""); err != nil {
		return nil, err
	}

	if !ut.hasMeta(pattern) {
		if _, err = vfs.Lstat(pattern); err != nil {
			return nil, nil
		}

		return []string{pattern}, nil
	}

	dir, file := ut.Split(pattern)
	volumeLen := 0

	if ut.OSType == avfs.OsWindows {
		volumeLen, dir = ut.cleanGlobPathWindows(dir)
	} else {
		dir = ut.cleanGlobPath(dir)
	}

	if !ut.hasMeta(dir[volumeLen:]) {
		return ut.glob(vfs, dir, file, nil)
	}

	// Prevent infinite recursion. See issue 15879.
	if dir == pattern {
		return nil, filepath.ErrBadPattern
	}

	var m []string
	m, err = ut.Glob(vfs, dir)
	if err != nil {
		return
	}

	for _, d := range m {
		matches, err = ut.glob(vfs, d, file, matches)
		if err != nil {
			return
		}
	}

	return
}

// cleanGlobPath prepares path for glob matching.
func (ut *Utils) cleanGlobPath(path string) string {
	switch path {
	case "":
		return "."
	case string(ut.PathSeparator):
		// do nothing to the path
		return path
	default:
		return path[0 : len(path)-1] // chop off trailing separator
	}
}

// cleanGlobPathWindows is windows version of cleanGlobPath.
func (ut *Utils) cleanGlobPathWindows(path string) (prefixLen int, cleaned string) {
	vollen := ut.volumeNameLen(path)
	switch {
	case path == "":
		return 0, "."
	case vollen+1 == len(path) && ut.IsPathSeparator(path[len(path)-1]): // /, \, C:\ and C:/
		// do nothing to the path
		return vollen + 1, path
	case vollen == len(path) && len(path) == 2: // C:
		return vollen, path + "." // convert C: into C:.
	default:
		if vollen >= len(path) {
			vollen = len(path) - 1
		}
		return vollen, path[0 : len(path)-1] // chop off trailing separator
	}
}

// glob searches for files matching pattern in the directory dir
// and appends them to matches. If the directory cannot be
// opened, it returns the existing matches. New matches are
// added in lexicographical order.
func (ut *Utils) glob(vfs avfs.VFS, dir, pattern string, matches []string) (m []string, e error) {
	m = matches
	fi, err := vfs.Stat(dir)
	if err != nil {
		return // ignore I/O error
	}
	if !fi.IsDir() {
		return // ignore I/O error
	}
	d, err := os.Open(dir)
	if err != nil {
		return // ignore I/O error
	}
	defer d.Close()

	names, _ := d.Readdirnames(-1)
	sort.Strings(names)

	for _, n := range names {
		matched, err := ut.Match(pattern, n)
		if err != nil {
			return m, err
		}
		if matched {
			m = append(m, ut.Join(dir, n))
		}
	}
	return
}

// hasMeta reports whether path contains any of the magic characters
// recognized by Match.
func (ut *Utils) hasMeta(path string) bool {
	magicChars := `*?[`

	if ut.OSType != avfs.OsWindows {
		magicChars = `*?[\`
	}

	return strings.ContainsAny(path, magicChars)
}

// IsAbs reports whether the path is absolute.
func (ut *Utils) IsAbs(path string) bool {
	return strings.HasPrefix(path, "/")
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func (ut *Utils) IsExist(err error) bool {
	return errors.Is(err, avfs.ErrFileExists)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func (ut *Utils) IsNotExist(err error) bool {
	return errors.Is(err, avfs.ErrNoSuchFileOrDir)
}

// IsPathSeparator reports whether c is a directory separator character.
func (ut *Utils) IsPathSeparator(c uint8) bool {
	return ut.PathSeparator == c
}

func (ut *Utils) Join(elem ...string) string {
	// If there's a bug here, fix the logic in ./path_plan9.go too.
	for i, e := range elem {
		if e != "" {
			return ut.Clean(strings.Join(elem[i:], string(ut.PathSeparator)))
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
func (ut *Utils) Match(pattern, name string) (matched bool, err error) {
Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string

		star, chunk, pattern = ut.scanChunk(pattern)
		if star && chunk == "" {
			// Trailing * matches rest of string unless it has a /.
			return !strings.Contains(name, string(ut.PathSeparator)), nil
		}

		// Look for match at current position.
		t, ok, err := ut.matchChunk(chunk, name)

		// if we're the last chunk, make sure we've exhausted the name
		// otherwise we'll give a false result even if we could still match
		// using the star
		if ok && (len(t) == 0 || len(pattern) > 0) {
			name = t

			continue
		}

		if err != nil {
			return false, err
		}

		if star {
			// Look for match skipping i+1 bytes.
			// Cannot skip /.
			for i := 0; i < len(name) && name[i] != ut.PathSeparator; i++ {
				t, ok, err := ut.matchChunk(chunk, name[i+1:])
				if ok {
					// if we're the last chunk, make sure we exhausted the name
					if len(pattern) == 0 && len(t) > 0 {
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

	return len(name) == 0, nil
}

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (ut *Utils) MkdirTemp(vfs avfs.VFS, dir, pattern string) (string, error) {
	const op = "mkdirtemp"

	if dir == "" {
		dir = vfs.TempDir()
	}

	prefix, suffix, err := ut.prefixAndSuffix(pattern)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: pattern, Err: err}
	}

	prefix = ut.joinPath(dir, prefix)
	try := 0

	for {
		name := prefix + ut.nextRandom() + suffix
		err := vfs.Mkdir(name, 0o700)

		if err == nil {
			return name, nil
		}

		if ut.IsExist(err) {
			if try++; try < 10000 {
				continue
			}

			return "", &fs.PathError{Op: op, Path: dir + string(ut.PathSeparator) + prefix + "*" + suffix, Err: fs.ErrExist}
		}

		if ut.IsNotExist(err) {
			_, err := vfs.Stat(dir)
			if ut.IsNotExist(err) {
				return "", err
			}
		}

		return "", err
	}
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (ut *Utils) ReadDir(vfs avfs.VFS, name string) ([]fs.DirEntry, error) {
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
func (ut *Utils) ReadFile(vfs avfs.VFS, name string) ([]byte, error) {
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
func (ut *Utils) Rel(basepath, targpath string) (string, error) {
	baseVol := ut.VolumeName(basepath)
	targVol := ut.VolumeName(targpath)
	base := ut.Clean(basepath)
	targ := ut.Clean(targpath)
	if ut.sameWord(targ, base) {
		return ".", nil
	}
	base = base[len(baseVol):]
	targ = targ[len(targVol):]
	if base == "." {
		base = ""
	} else if base == "" && ut.volumeNameLen(baseVol) > 2 /* isUNC */ {
		// Treat any targetpath matching `\\host\share` basepath as absolute path.
		base = string(ut.PathSeparator)
	}

	// Can't use IsAbs - `\a` and `a` are both relative in Windows.
	baseSlashed := len(base) > 0 && base[0] == ut.PathSeparator
	targSlashed := len(targ) > 0 && targ[0] == ut.PathSeparator
	if baseSlashed != targSlashed || !ut.sameWord(baseVol, targVol) {
		return "", errors.New("Rel: can't make " + targpath + " relative to " + basepath)
	}
	// Position base[b0:bi] and targ[t0:ti] at the first differing elements.
	bl := len(base)
	tl := len(targ)
	var b0, bi, t0, ti int
	for {
		for bi < bl && base[bi] != ut.PathSeparator {
			bi++
		}
		for ti < tl && targ[ti] != ut.PathSeparator {
			ti++
		}
		if !ut.sameWord(targ[t0:ti], base[b0:bi]) {
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
		seps := strings.Count(base[b0:bl], string(ut.PathSeparator))
		size := 2 + seps*3

		if tl != t0 {
			size += 1 + tl - t0
		}

		buf := make([]byte, size)
		n := copy(buf, "..")

		for i := 0; i < seps; i++ {
			buf[n] = ut.PathSeparator
			copy(buf[n+1:], "..")
			n += 3
		}

		if t0 != tl {
			buf[n] = ut.PathSeparator
			copy(buf[n+1:], targ[t0:])
		}

		return string(buf), nil
	}

	return targ[t0:], nil
}

// SegmentPath segments string key paths by separator (using avfs.PathSeparator).
// For example with path = "/a/b/c" it will return in successive calls :
//
// "a", "/b/c"
// "b", "/c"
// "c", ""
//
// 	for start, end, isLast := 1, 0, len(path) <= 1; !isLast; start = end + 1 {
//		end, isLast = vfsutils.SegmentPath(path, start)
//		fmt.Println(path[start:end], path[end:])
//	}
//
func (ut *Utils) SegmentPath(path string, start int) (end int, isLast bool) {
	pos := strings.IndexRune(path[start:], rune(ut.PathSeparator))
	if pos != -1 {
		return start + pos, false
	}

	return len(path), true
}

// Split splits path immediately following the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, Split returns an empty dir
// and file set to path.
// The returned values have the property that path = dir+file.
func (ut *Utils) Split(path string) (dir, file string) {
	vol := ut.VolumeName(path)

	i := len(path) - 1
	for i >= len(vol) && !ut.IsPathSeparator(path[i]) {
		i--
	}

	return path[:i+1], path[i+1:]
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (ut *Utils) ToSlash(path string) string {
	if ut.PathSeparator == '/' {
		return path
	}

	return strings.ReplaceAll(path, string(ut.PathSeparator), "/")
}

// VolumeName returns leading volume name.
// Given "C:\foo\bar" it returns "C:" on Windows.
// Given "\\host\share\foo" it returns "\\host\share".
// On other platforms it returns "".
func (ut *Utils) VolumeName(path string) string {
	return path[:ut.volumeNameLen(path)]
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
func (ut *Utils) WalkDir(vfs avfs.VFS, root string, fn fs.WalkDirFunc) error {
	info, err := vfs.Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = ut.walkDir(vfs, root, &statDirEntry{info}, fn)
	}

	if err == filepath.SkipDir {
		return nil
	}

	return err
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
func (ut *Utils) WriteFile(vfs avfs.VFS, name string, data []byte, perm fs.FileMode) error {
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

type statDirEntry struct {
	info fs.FileInfo
}

func (d *statDirEntry) Name() string               { return d.info.Name() }
func (d *statDirEntry) IsDir() bool                { return d.info.IsDir() }
func (d *statDirEntry) Type() fs.FileMode          { return d.info.Mode().Type() }
func (d *statDirEntry) Info() (fs.FileInfo, error) { return d.info, nil }

// walkDir recursively descends path, calling walkDirFn.
func (ut *Utils) walkDir(vfs avfs.VFS, path string, d fs.DirEntry, walkDirFn fs.WalkDirFunc) error {
	if err := walkDirFn(path, d, nil); err != nil || !d.IsDir() {
		if err == filepath.SkipDir && d.IsDir() {
			// Successfully skipped directory.
			err = nil
		}

		return err
	}

	dirs, err := vfs.ReadDir(path)
	if err != nil {
		// Second call, to report ReadDir error.
		err = walkDirFn(path, d, err)
		if err != nil {
			return err
		}
	}

	for _, d1 := range dirs {
		path1 := vfs.Join(path, d1.Name())
		if err := ut.walkDir(vfs, path1, d1, walkDirFn); err != nil {
			if err == filepath.SkipDir {
				break
			}

			return err
		}
	}

	return nil
}

// getEsc gets a possibly-escaped character from chunk, for a character class.
func (ut *Utils) getEsc(chunk string) (r rune, nchunk string, err error) {
	if len(chunk) == 0 || chunk[0] == '-' || chunk[0] == ']' {
		err = filepath.ErrBadPattern

		return
	}

	if chunk[0] == '\\' && runtime.GOOS != "windows" {
		chunk = chunk[1:]
		if len(chunk) == 0 {
			err = filepath.ErrBadPattern

			return
		}
	}

	r, n := utf8.DecodeRuneInString(chunk)
	if r == utf8.RuneError && n == 1 {
		err = filepath.ErrBadPattern
	}

	nchunk = chunk[n:]
	if len(nchunk) == 0 {
		err = filepath.ErrBadPattern
	}

	return
}

func (ut *Utils) joinPath(dir, name string) string {
	if len(dir) > 0 && ut.IsPathSeparator(dir[len(dir)-1]) {
		return dir + name
	}

	return dir + string(ut.PathSeparator) + name
}

// matchChunk checks whether chunk matches the beginning of s.
// If so, it returns the remainder of s (after the match).
// Chunk is all single-character operators: literals, char classes, and ?.
func (ut *Utils) matchChunk(chunk, s string) (rest string, ok bool, err error) {
	// failed records whether the match has failed.
	// After the match fails, the loop continues on processing chunk,
	// checking that the pattern is well-formed but no longer reading s.
	failed := false

	for len(chunk) > 0 {
		if !failed && len(s) == 0 {
			failed = true
		}

		switch chunk[0] {
		case '[':
			// character class
			var r rune

			if !failed {
				var n int
				r, n = utf8.DecodeRuneInString(s)
				s = s[n:]
			}

			chunk = chunk[1:]
			// possibly negated
			negated := false

			if len(chunk) > 0 && chunk[0] == '^' {
				negated = true
				chunk = chunk[1:]
			}

			// parse all ranges
			match := false
			nrange := 0

			for {
				if len(chunk) > 0 && chunk[0] == ']' && nrange > 0 {
					chunk = chunk[1:]
					break
				}

				var lo, hi rune
				if lo, chunk, err = ut.getEsc(chunk); err != nil {
					return "", false, err
				}

				hi = lo

				if chunk[0] == '-' {
					if hi, chunk, err = ut.getEsc(chunk[1:]); err != nil {
						return "", false, err
					}
				}

				if lo <= r && r <= hi {
					match = true
				}

				nrange++
			}
			if match == negated {
				failed = true
			}
		case '?':
			if !failed {
				if s[0] == ut.PathSeparator {
					failed = true
				}
				_, n := utf8.DecodeRuneInString(s)
				s = s[n:]
			}

			chunk = chunk[1:]
		case '\\':
			if ut.OSType != avfs.OsWindows {
				chunk = chunk[1:]
				if len(chunk) == 0 {
					return "", false, filepath.ErrBadPattern
				}
			}

			fallthrough
		default:
			if !failed {
				if chunk[0] != s[0] {
					failed = true
				}

				s = s[1:]
			}

			chunk = chunk[1:]
		}
	}

	if failed {
		return "", false, nil
	}

	return s, true, nil
}

// prefixAndSuffix splits pattern by the last wildcard "*", if applicable,
// returning prefix as the part before "*" and suffix as the part after "*".
func (ut *Utils) prefixAndSuffix(pattern string) (prefix, suffix string, err error) {
	for i := 0; i < len(pattern); i++ {
		if ut.IsPathSeparator(pattern[i]) {
			return "", "", avfs.ErrPatternHasSeparator
		}
	}

	if pos := strings.LastIndexByte(pattern, '*'); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}

	return prefix, suffix, nil
}

func (ut *Utils) nextRandom() string {
	ut.randmu.Lock()

	r := ut.randno
	if r == 0 {
		r = ut.reseed()
	}

	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	ut.randno = r
	ut.randmu.Unlock()

	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

func (ut *Utils) reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func (ut *Utils) sameWord(a, b string) bool {
	return a == b
}

// scanChunk gets the next segment of pattern, which is a non-star string
// possibly preceded by a star.
func (ut *Utils) scanChunk(pattern string) (star bool, chunk, rest string) {
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true
	}

	inrange := false
	var i int

Scan:
	for i = 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			if ut.OSType != avfs.OsWindows {
				// error check handled in matchChunk: bad pattern.
				if i+1 < len(pattern) {
					i++
				}
			}
		case '[':
			inrange = true
		case ']':
			inrange = false
		case '*':
			if !inrange {
				break Scan
			}
		}
	}

	return star, pattern[0:i], pattern[i:]
}

// volumeNameLen returns length of the leading volume name on Windows.
// It returns 0 elsewhere.
func (ut *Utils) volumeNameLen(path string) int {
	if ut.OSType != avfs.OsWindows {
		return 0
	}

	if len(path) < 2 {
		return 0
	}

	// with drive letter
	c := path[0]
	if path[1] == ':' && ('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
		return 2
	}

	// is it UNC? https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	if l := len(path); l >= 5 && isSlash(path[0]) && isSlash(path[1]) &&
		!isSlash(path[2]) && path[2] != '.' {
		// first, leading `\\` and next shouldn't be `\`. its server name.
		for n := 3; n < l-1; n++ {
			// second, next '\' shouldn't be repeated.
			if isSlash(path[n]) {
				n++
				// third, following something characters. its share name.
				if !isSlash(path[n]) {
					if path[n] == '.' {
						break
					}

					for ; n < l; n++ {
						if isSlash(path[n]) {
							break
						}
					}

					return n
				}

				break
			}
		}
	}

	return 0
}

func isSlash(c uint8) bool {
	return c == '\\' || c == '/'
}

// A lazybuf is a lazily constructed path buffer.
// It supports append, reading previously appended bytes,
// and retrieving the final string. It does not allocate a buffer
// to hold the output until that output diverges from s.
type lazybuf struct {
	path       string
	buf        []byte
	w          int
	volAndPath string
	volLen     int
}

func (b *lazybuf) index(i int) byte {
	if b.buf != nil {
		return b.buf[i]
	}

	return b.path[i]
}

func (b *lazybuf) append(c byte) {
	if b.buf == nil {
		if b.w < len(b.path) && b.path[b.w] == c {
			b.w++
			return
		}

		b.buf = make([]byte, len(b.path))
		copy(b.buf, b.path[:b.w])
	}

	b.buf[b.w] = c
	b.w++
}

func (b *lazybuf) string() string {
	if b.buf == nil {
		return b.volAndPath[:b.volLen+b.w]
	}

	return b.volAndPath[:b.volLen] + string(b.buf[:b.w])
}

// RunTimeOS returns the current Operating System type.
func RunTimeOS() avfs.OSType {
	switch runtime.GOOS {
	case "linux":
		return avfs.OsLinux
	case "darwin":
		return avfs.OsDarwin
	case "windows":
		return avfs.OsWindows
	default:
		return avfs.OsUnknown
	}
}
