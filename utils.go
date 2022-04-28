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
	"sort"
	"strings"
)

// OSUtils is the default utilities structure for the current operating system.
var OSUtils = NewUtils(currentOSType()) //nolint:gochecknoglobals // Utils for the current operating system.

// Utils regroups common functions used by emulated file systems.
//
// Most of these functions are extracted and adapted from Go standard library
// to be used indifferently on Unix or Windows system.
type Utils struct {
	// OSType defines the operating system type.
	osType OSType

	// pathSeparator is the OS-specific path separator.
	pathSeparator uint8
}

func NewUtils(osType OSType) Utils {
	sep := '/'
	if osType == OsWindows {
		sep = '\\'
	}

	return Utils{
		osType:        osType,
		pathSeparator: uint8(sep),
	}
}

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (ut *Utils) Abs(vfs VFS, path string) (string, error) {
	if ut.IsAbs(path) {
		return ut.Clean(path), nil
	}

	wd, err := vfs.Getwd()
	if err != nil {
		return "", err
	}

	return ut.Join(wd, path), nil
}

// AdminGroupName returns the name of the administrator group of the file system.
func (ut *Utils) AdminGroupName() string {
	switch ut.osType {
	case OsWindows:
		return "Administrators"
	default:
		return "root"
	}
}

// AdminUserName returns the name of the administrator of the file system.
func (ut *Utils) AdminUserName() string {
	switch ut.osType {
	case OsWindows:
		return "ContainerAdministrator"
	default:
		return "root"
	}
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
		return string(ut.pathSeparator)
	}

	return path
}

// DirInfo contains information to create a directory.
type DirInfo struct {
	Path string
	Perm fs.FileMode
}

// BaseDirs returns an array of directories always present in the file system.
func (ut *Utils) BaseDirs(basePath string) []DirInfo {
	const volumeNameLen = 2

	switch ut.osType {
	case OsWindows:
		return []DirInfo{
			{Path: ut.Join(basePath, ut.HomeDir()[volumeNameLen:]), Perm: DefaultDirPerm},
			{Path: ut.Join(basePath, ut.TempDir(ut.AdminUserName())[volumeNameLen:]), Perm: DefaultDirPerm},
			{Path: ut.Join(basePath, ut.TempDir(DefaultUser.Name())[volumeNameLen:]), Perm: DefaultDirPerm},
			{Path: ut.Join(basePath, `\Windows`), Perm: DefaultDirPerm},
		}
	default:
		return []DirInfo{
			{Path: ut.Join(basePath, ut.HomeDir()), Perm: ut.HomeDirPerm()},
			{Path: ut.Join(basePath, "/root"), Perm: 0o700},
			{Path: ut.Join(basePath, "/tmp"), Perm: 0o777},
		}
	}
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
	volLen := ut.VolumeNameLen(path)

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
		out.append(ut.pathSeparator)

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
					out.append(ut.pathSeparator)
				}

				out.append('.')
				out.append('.')
				dotdot = out.w
			}
		default:
			// real path element.
			// add slash if needed
			if rooted && out.w != 1 || !rooted && out.w != 0 {
				out.append(ut.pathSeparator)
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

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (ut *Utils) Create(vfs VFS, name string) (File, error) {
	return vfs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

// CreateBaseDirs creates base directories on a file system.
func (ut *Utils) CreateBaseDirs(vfs VFS, basePath string) error {
	dirs := ut.BaseDirs(basePath)
	for _, dir := range dirs {
		err := vfs.MkdirAll(dir.Path, dir.Perm)
		if err != nil {
			return err
		}

		if ut.osType != OsWindows {
			err = vfs.Chmod(dir.Path, dir.Perm)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateHomeDir creates and returns the home directory of a user.
// If there is an error, it will be of type *PathError.
func (ut *Utils) CreateHomeDir(vfs VFS, u UserReader) (string, error) {
	userDir := ut.HomeDirUser(vfs, u)

	err := vfs.Mkdir(userDir, ut.HomeDirPerm())
	if err != nil {
		return "", err
	}

	switch ut.osType {
	case OsWindows:
		err = vfs.MkdirAll(ut.TempDir(u.Name()), DefaultDirPerm)
	default:
		err = vfs.Chown(userDir, u.Uid(), u.Gid())
	}

	if err != nil {
		return "", err
	}

	return userDir, nil
}

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (ut *Utils) CreateTemp(vfs VFS, dir, pattern string) (File, error) {
	const op = "createtemp"

	if dir == "" {
		dir = ut.TempDir(vfs.User().Name())
	}

	prefix, suffix, err := ut.prefixAndSuffix(pattern)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: pattern, Err: err}
	}

	prefix = ut.joinPath(dir, prefix)

	try := 0

	for {
		name := prefix + nextRandom() + suffix

		f, err := vfs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if ut.IsExist(err) {
			try++
			if try < 10000 {
				continue
			}

			return nil, &fs.PathError{Op: op, Path: dir + string(ut.pathSeparator) + prefix + "*" + suffix, Err: fs.ErrExist}
		}

		return f, err
	}
}

// DefaultGroupName returns the name of the default users group of the file system.
func (ut *Utils) DefaultGroupName() string {
	switch ut.osType {
	case OsWindows:
		return "Users"
	default:
		return "users"
	}
}

// DefaultUserName returns the name of the default username of the file system.
func (ut *Utils) DefaultUserName() string {
	switch ut.osType {
	case OsWindows:
		return "Default"
	default:
		return "user"
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
	if ut.osType != OsWindows {
		return path
	}

	return strings.ReplaceAll(path, "/", string(ut.pathSeparator))
}

// FromUnixPath returns valid path from a unix path.
func (ut *Utils) FromUnixPath(volumeName, path string) string {
	if ut.OSType() != OsWindows {
		return path
	}

	if path[0] == '/' {
		return ut.Join(volumeName, path)
	}

	return ut.FromSlash(path)
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (ut *Utils) Glob(vfs VFS, pattern string) (matches []string, err error) {
	// Check pattern is well-formed.
	if _, err = ut.Match(pattern, ""); err != nil {
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

	if ut.osType == OsWindows {
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

	m, err = vfs.Glob(dir)
	if err != nil {
		return
	}

	for _, d := range m {
		matches, err = ut.glob(vfs, d, file, matches)
		if err != nil {
			return
		}
	}

	return //nolint:nakedret // Adapted from standard library.
}

// HomeDir returns the home directory of the file system.
func (ut *Utils) HomeDir() string {
	switch ut.osType {
	case OsWindows:
		return DefaultVolume + `\Users`
	default:
		return "/home"
	}
}

// HomeDirUser returns the home directory of the user.
// If the file system does not have an identity manager, the root directory is returned.
func (ut *Utils) HomeDirUser(vfs VFS, u UserReader) string {
	if !vfs.HasFeature(FeatIdentityMgr) {
		return ut.FromUnixPath(DefaultVolume, "/")
	}

	name := u.Name()
	if ut.osType != OsWindows && name == ut.AdminUserName() {
		return "/root"
	}

	return ut.Join(ut.HomeDir(), name)
}

// HomeDirPerm return the default permission for home directories.
func (ut *Utils) HomeDirPerm() fs.FileMode {
	return 0o755
}

// IsAbs reports whether the path is absolute.
func (ut *Utils) IsAbs(path string) bool {
	if ut.osType != OsWindows {
		return strings.HasPrefix(path, "/")
	}

	if isReservedName(path) {
		return true
	}

	l := ut.VolumeNameLen(path)
	if l == 0 {
		return false
	}

	path = path[l:]
	if path == "" {
		return false
	}

	return isSlash(path[0])
}

// IsPathSeparator reports whether c is a directory separator character.
func (ut *Utils) IsPathSeparator(c uint8) bool {
	if ut.osType != OsWindows {
		return c == '/'
	}

	return c == '\\' || c == '/'
}

// Join joins any number of path elements into a single path, adding a
// separating slash if necessary. The result is Cleaned; in particular,
// all empty strings are ignored.
func (ut *Utils) Join(elem ...string) string {
	if ut.osType != OsWindows {
		// If there's a bug here, fix the logic in ./path_plan9.go too.
		for i, e := range elem {
			if e != "" {
				return ut.Clean(strings.Join(elem[i:], string(ut.pathSeparator)))
			}
		}

		return ""
	}

	return ut.joinWindows(elem)
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
			return !strings.Contains(name, string(ut.pathSeparator)), nil
		}

		// Look for match at current position.
		t, ok, err := ut.matchChunk(chunk, name)

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
			for i := 0; i < len(name) && name[i] != ut.pathSeparator; i++ {
				t, ok, err := ut.matchChunk(chunk, name[i+1:])
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
func (ut *Utils) MkdirTemp(vfs VFS, dir, pattern string) (string, error) {
	const op = "mkdirtemp"

	if dir == "" {
		dir = ut.TempDir(vfs.User().Name())
	}

	prefix, suffix, err := ut.prefixAndSuffix(pattern)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: pattern, Err: err}
	}

	prefix = ut.joinPath(dir, prefix)
	try := 0

	for {
		name := prefix + nextRandom() + suffix

		err := vfs.Mkdir(name, 0o700)
		if err == nil {
			return name, nil
		}

		if ut.IsExist(err) {
			try++
			if try < 10000 {
				continue
			}

			return "", &fs.PathError{Op: op, Path: dir + string(ut.pathSeparator) + prefix + "*" + suffix, Err: fs.ErrExist}
		}

		if ut.IsNotExist(err) {
			_, err := vfs.Stat(dir) // nolint:govet // declaration of "err" shadows declaration
			if ut.IsNotExist(err) {
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
func (ut *Utils) Open(vfs VFS, name string) (File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// PathSeparator return the OS-specific path separator.
func (ut *Utils) PathSeparator() uint8 {
	return ut.pathSeparator
}

// OSType returns the operating system type of the file system.
func (ut *Utils) OSType() OSType {
	return ut.osType
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (ut *Utils) ReadDir(vfs VFS, name string) ([]fs.DirEntry, error) {
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
func (ut *Utils) ReadFile(vfs VFS, name string) ([]byte, error) {
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
	} else if base == "" && ut.VolumeNameLen(baseVol) > 2 /* isUNC */ {
		// Treat any targetpath matching `\\host\share` basepath as absolute path.
		base = string(ut.pathSeparator)
	}

	// Can't use IsAbs - `\a` and `a` are both relative in Windows.
	baseSlashed := len(base) > 0 && base[0] == ut.pathSeparator
	targSlashed := len(targ) > 0 && targ[0] == ut.pathSeparator

	if baseSlashed != targSlashed || !ut.sameWord(baseVol, targVol) {
		return "", errors.New("Rel: can't make " + targpath + " relative to " + basepath)
	}

	// Position base[b0:bi] and targ[t0:ti] at the first differing elements.
	bl := len(base)
	tl := len(targ)

	var b0, bi, t0, ti int

	for {
		for bi < bl && base[bi] != ut.pathSeparator {
			bi++
		}

		for ti < tl && targ[ti] != ut.pathSeparator {
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
		seps := strings.Count(base[b0:bl], string(ut.pathSeparator))
		size := 2 + seps*3

		if tl != t0 {
			size += 1 + tl - t0
		}

		buf := make([]byte, size)
		n := copy(buf, "..")

		for i := 0; i < seps; i++ {
			buf[n] = ut.pathSeparator
			copy(buf[n+1:], "..")
			n += 3
		}

		if t0 != tl {
			buf[n] = ut.pathSeparator
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
func (ut *Utils) Split(path string) (dir, file string) {
	vol := ut.VolumeName(path)

	i := len(path) - 1
	for i >= len(vol) && !ut.IsPathSeparator(path[i]) {
		i--
	}

	return path[:i+1], path[i+1:]
}

// SplitAbs splits an absolute path immediately preceding the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, splitPath returns an empty dir
// and file set to path.
// The returned values have the property that path = dir + PathSeparator + file.
func (ut *Utils) SplitAbs(path string) (dir, file string) {
	l := ut.VolumeNameLen(path)

	i := len(path) - 1
	for i >= l && !ut.IsPathSeparator(path[i]) {
		i--
	}

	return path[:i], path[i+1:]
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
func (ut *Utils) TempDir(userName string) string {
	if ut.osType != OsWindows {
		return "/tmp"
	}

	dir := ut.Join(DefaultVolume, `\Users\`, userName, `\AppData\Local\Temp`)

	return ShortPathName(dir)
}

// ToSlash returns the result of replacing each separator character
// in path with a slash ('/') character. Multiple separators are
// replaced by multiple slashes.
func (ut *Utils) ToSlash(path string) string {
	if ut.pathSeparator == '/' {
		return path
	}

	return strings.ReplaceAll(path, string(ut.pathSeparator), "/")
}

// VolumeName returns leading volume name.
// Given "C:\foo\bar" it returns "C:" on Windows.
// Given "\\host\share\foo" it returns "\\host\share".
// On other platforms it returns "".
func (ut *Utils) VolumeName(path string) string {
	return path[:ut.VolumeNameLen(path)]
}

// VolumeNameLen returns length of the leading volume name on Windows.
// It returns 0 elsewhere.
func (ut *Utils) VolumeNameLen(path string) int {
	if ut.osType != OsWindows {
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
func (ut *Utils) WalkDir(vfs VFS, root string, fn fs.WalkDirFunc) error {
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

type statDirEntry struct {
	info fs.FileInfo
}

func (d *statDirEntry) Name() string               { return d.info.Name() }
func (d *statDirEntry) IsDir() bool                { return d.info.IsDir() }
func (d *statDirEntry) Type() fs.FileMode          { return d.info.Mode().Type() }
func (d *statDirEntry) Info() (fs.FileInfo, error) { return d.info, nil }

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
func (ut *Utils) WriteFile(vfs VFS, name string, data []byte, perm fs.FileMode) error {
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
