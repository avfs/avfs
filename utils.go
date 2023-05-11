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
	_ "unsafe" // for go:linkname only.
)

var currentOSType = initOSsType() //nolint:gochecknoglobals // Store the current OS Type.

// BuildFeatures returns the features available depending on build tags.
func BuildFeatures() Features {
	return buildFeatSetOSType
}

// CurrentOSType returns the current OSType.
func CurrentOSType() OSType {
	return currentOSType
}

// initOSsType initialize the current OSType.
func initOSsType() OSType {
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

// Utils regroups common functions used by emulated file systems.
//
// Most of these functions are extracted from Go standard library
// and adapted to be used indifferently on Unix or Windows system.
type Utils[T VFSBase] struct {
	features      Features // features defines the list of features available for the file system.
	osType        OSType   // OSType defines the operating system type.
	pathSeparator uint8    // pathSeparator is the OS-specific path separator.
}

// Features returns the set of features provided by the file system or identity manager.
func (ut *Utils[_]) Features() Features {
	return ut.features
}

// HasFeature returns true if the file system or identity manager provides a given features.
func (ut *Utils[_]) HasFeature(feature Features) bool {
	return ut.features&feature == feature
}

// SetFeatures sets the features of the file system or identity manager.
func (ut *Utils[_]) SetFeatures(feature Features) {
	ut.features = feature
}

// Abs returns an absolute representation of path.
// If the path is not absolute it will be joined with the current
// working directory to turn it into an absolute path. The absolute
// path name for a given file is not guaranteed to be unique.
// Abs calls Clean on the result.
func (ut *Utils[T]) Abs(vfs T, path string) (string, error) {
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
func AdminGroupName(osType OSType) string {
	switch osType {
	case OsWindows:
		return "Administrators"
	default:
		return "root"
	}
}

// AdminUserName returns the name of the administrator of the file system.
func AdminUserName(osType OSType) string {
	switch osType {
	case OsWindows:
		return "ContainerAdministrator"
	default:
		return "root"
	}
}

// DirInfo contains information to create a directory.
type DirInfo struct {
	Path string
	Perm fs.FileMode
}

// Create creates or truncates the named file. If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned DummyFile can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func (*Utils[T]) Create(vfs T, name string) (File, error) {
	return vfs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, DefaultFilePerm)
}

// CreateHomeDir creates and returns the home directory of a user.
// If there is an error, it will be of type *PathError.
func (ut *Utils[T]) CreateHomeDir(vfs T, u UserReader) (string, error) {
	userDir := ut.HomeDirUser(u)

	err := vfs.Mkdir(userDir, HomeDirPerm())
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

// CreateSystemDirs creates the system directories of a file system.
func (ut *Utils[T]) CreateSystemDirs(vfs T, basePath string) error {
	dirs := ut.SystemDirs(basePath)
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

// CreateTemp creates a new temporary file in the directory dir,
// opens the file for reading and writing, and returns the resulting file.
// The filename is generated by taking pattern and adding a random string to the end.
// If pattern includes a "*", the random string replaces the last "*".
// If dir is the empty string, CreateTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling CreateTemp simultaneously will not choose the same file.
// The caller can use the file's Name method to find the pathname of the file.
// It is the caller's responsibility to remove the file when it is no longer needed.
func (ut *Utils[T]) CreateTemp(vfs T, dir, pattern string) (File, error) {
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

// cleanGlobPath prepares path for glob matching.
func (ut *Utils[_]) cleanGlobPath(path string) string {
	switch path {
	case "":
		return "."
	case string(ut.pathSeparator):
		// do nothing to the path
		return path
	default:
		return path[0 : len(path)-1] // chop off trailing separator
	}
}

// cleanGlobPathWindows is Windows version of cleanGlobPath.
func (ut *Utils[_]) cleanGlobPathWindows(path string) (prefixLen int, cleaned string) {
	vollen := ut.VolumeNameLen(path)

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

// FromUnixPath returns valid path for Unix or Windows from a unix path.
// For Windows systems, absolute paths are prefixed with the default volume
// and relative paths are preserved.
func FromUnixPath(path string) string {
	if CurrentOSType() != OsWindows {
		return path
	}

	if path[0] != '/' {
		return filepath.FromSlash(path)
	}

	return filepath.Join(DefaultVolume, filepath.FromSlash(path))
}

// Glob returns the names of all files matching pattern or nil
// if there is no matching file. The syntax of patterns is the same
// as in Match. The pattern may describe hierarchical names such as
// /usr/*/bin/ed (assuming the Separator is '/').
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
func (ut *Utils[T]) Glob(vfs T, pattern string) (matches []string, err error) {
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

// glob searches for files matching pattern in the directory dir
// and appends them to matches. If the directory cannot be
// opened, it returns the existing matches. New matches are
// added in lexicographical order.
func (ut *Utils[T]) glob(vfs T, dir, pattern string, matches []string) (m []string, e error) {
	m = matches

	fi, err := vfs.Stat(dir)
	if err != nil {
		return // ignore I/O error
	}

	if !fi.IsDir() {
		return // ignore I/O error
	}

	d, err := vfs.OpenFile(dir, os.O_RDONLY, 0)
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
			m = append(m, vfs.Join(dir, n))
		}
	}

	return //nolint:nakedret // Adapted from standard library.
}

// hasMeta reports whether path contains any of the magic characters
// recognized by Match.
func (ut *Utils[_]) hasMeta(path string) bool {
	magicChars := `*?[`

	if ut.osType != OsWindows {
		magicChars = `*?[\`
	}

	return strings.ContainsAny(path, magicChars)
}

// HomeDir returns the home directory of the file system.
func (ut *Utils[_]) HomeDir() string {
	switch ut.osType {
	case OsWindows:
		return DefaultVolume + `\Users`
	default:
		return "/home"
	}
}

// HomeDirUser returns the home directory of the user.
// If the file system does not have an identity manager, the root directory is returned.
func (ut *Utils[T]) HomeDirUser(u UserReader) string {
	name := u.Name()
	if ut.osType == OsWindows {
		return ut.joinPath(ut.HomeDir(), name)
	}

	if name == AdminUserName(ut.osType) {
		return "/root"
	}

	return ut.joinPath(ut.HomeDir(), name)
}

// HomeDirPerm return the default permission for home directories.
func HomeDirPerm() fs.FileMode {
	return 0o700
}

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
//
// This function predates errors.Is. It only supports errors returned by
// the os package. New code should use errors.Is(err, fs.ErrExist).
func (*Utils[_]) IsExist(err error) bool {
	return errors.Is(err, fs.ErrExist)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
//
// This function predates errors.Is. It only supports errors returned by
// the os package. New code should use errors.Is(err, fs.ErrNotExist).
func (*Utils[_]) IsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

func (ut *Utils[_]) joinPath(dir, name string) string {
	if len(dir) > 0 && ut.IsPathSeparator(dir[len(dir)-1]) {
		return dir + name
	}

	return dir + string(ut.pathSeparator) + name
}

// MkdirTemp creates a new temporary directory in the directory dir
// and returns the pathname of the new directory.
// The new directory's name is generated by adding a random string to the end of pattern.
// If pattern includes a "*", the random string replaces the last "*" instead.
// If dir is the empty string, MkdirTemp uses the default directory for temporary files, as returned by TempDir.
// Multiple programs or goroutines calling MkdirTemp simultaneously will not choose the same directory.
// It is the caller's responsibility to remove the directory when it is no longer needed.
func (ut *Utils[T]) MkdirTemp(vfs T, dir, pattern string) (string, error) {
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
			_, err := vfs.Stat(dir) //nolint:govet // declaration of "err" shadows declaration
			if ut.IsNotExist(err) {
				return "", err
			}
		}

		return "", err
	}
}

// nextRandom is used in Utils.CreateTemp and Utils.MkdirTemp.
//
//go:linkname nextRandom os.nextRandom
func nextRandom() string

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (*Utils[T]) Open(vfs T, name string) (File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// OpenMode returns the open mode from the input flags.
func (*Utils[_]) OpenMode(flag int) OpenMode {
	var om OpenMode

	if flag == os.O_RDONLY {
		return OpenRead
	}

	if flag&os.O_RDWR != 0 {
		om = OpenRead | OpenWrite
	}

	if flag&(os.O_EXCL|os.O_CREATE) == (os.O_EXCL | os.O_CREATE) {
		om |= OpenCreate | OpenCreateExcl | OpenWrite
	}

	if flag&os.O_CREATE != 0 {
		om |= OpenCreate | OpenWrite
	}

	if flag&os.O_APPEND != 0 {
		om |= OpenAppend | OpenWrite
	}

	if flag&os.O_TRUNC != 0 {
		om |= OpenTruncate | OpenWrite
	}

	if flag&os.O_WRONLY != 0 {
		om |= OpenWrite
	}

	return om
}

// OSType returns the operating system type of the file system.
func (ut *Utils[_]) OSType() OSType {
	return ut.osType
}

// prefixAndSuffix splits pattern by the last wildcard "*", if applicable,
// returning prefix as the part before "*" and suffix as the part after "*".
func (ut *Utils[_]) prefixAndSuffix(pattern string) (prefix, suffix string, err error) {
	for i := 0; i < len(pattern); i++ {
		if ut.IsPathSeparator(pattern[i]) {
			return "", "", ErrPatternHasSeparator
		}
	}

	if pos := strings.LastIndexByte(pattern, '*'); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}

	return prefix, suffix, nil
}

// ReadDir reads the named directory,
// returning all its directory entries sorted by filename.
// If an error occurs reading the directory,
// ReadDir returns the entries it was able to read before the error,
// along with the error.
func (*Utils[T]) ReadDir(vfs T, name string) ([]fs.DirEntry, error) {
	f, err := vfs.OpenFile(name, os.O_RDONLY, 0)
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
func (*Utils[T]) ReadFile(vfs T, name string) ([]byte, error) {
	f, err := vfs.OpenFile(name, os.O_RDONLY, 0)
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
			d := append(data[:cap(data)], 0) //nolint:gocritic // append result not assigned to the same slice
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

// SetOSType sets the osType.
func (ut *Utils[_]) SetOSType(osType OSType) {
	if osType == OsUnknown {
		osType = CurrentOSType()
	}

	if buildFeatSetOSType == 0 && osType != CurrentOSType() {
		panic("Can't set OS type, use build tag 'avfs_setostype' to set OS type")
	}

	sep := uint8('/')
	if osType == OsWindows {
		sep = '\\'
	}

	ut.pathSeparator = sep
	ut.osType = osType
}

// SplitAbs splits an absolute path immediately preceding the final Separator,
// separating it into a directory and file name component.
// If there is no Separator in path, splitPath returns an empty dir
// and file set to path.
// The returned values have the property that path = dir + PathSeparator + file.
func (ut *Utils[_]) SplitAbs(path string) (dir, file string) {
	l := ut.VolumeNameLen(path)

	i := len(path) - 1
	for i >= l && !ut.IsPathSeparator(path[i]) {
		i--
	}

	return path[:i], path[i+1:]
}

// SystemDirs returns an array of system directories always present in the file system.
func (ut *Utils[_]) SystemDirs(basePath string) []DirInfo {
	const volumeNameLen = 2

	switch ut.osType {
	case OsWindows:
		return []DirInfo{
			{Path: ut.Join(basePath, ut.HomeDir()[volumeNameLen:]), Perm: DefaultDirPerm},
			{Path: ut.Join(basePath, ut.TempDir(AdminUserName(ut.osType))[volumeNameLen:]), Perm: DefaultDirPerm},
			{Path: ut.Join(basePath, ut.TempDir(DefaultName)[volumeNameLen:]), Perm: DefaultDirPerm},
			{Path: ut.Join(basePath, `\Windows`), Perm: DefaultDirPerm},
		}
	default:
		return []DirInfo{
			{Path: ut.Join(basePath, ut.HomeDir()), Perm: HomeDirPerm()},
			{Path: ut.Join(basePath, "/root"), Perm: 0o700},
			{Path: ut.Join(basePath, "/tmp"), Perm: 0o777},
		}
	}
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
func (ut *Utils[_]) TempDir(userName string) string {
	if ut.osType != OsWindows {
		return "/tmp"
	}

	dir := ut.Join(DefaultVolume, `\Users\`, userName, `\AppData\Local\Temp`)

	return ShortPathName(dir)
}

//go:linkname volumeNameLen path/filepath.volumeNameLen
func volumeNameLen(path string) int

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
func (ut *Utils[T]) WalkDir(vfs T, root string, fn fs.WalkDirFunc) error {
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

// walkDir recursively descends path, calling walkDirFn.
func (ut *Utils[T]) walkDir(vfs T, path string, d fs.DirEntry, walkDirFn fs.WalkDirFunc) error {
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
func (*Utils[T]) WriteFile(vfs T, name string, data []byte, perm fs.FileMode) error {
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
