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

package vfsutils

import (
	"errors"
	"math/rand"
	"os"
	"runtime"
	"strings"

	"github.com/avfs/avfs"
)

var (
	// UMask is the global variable containing the file mode creation mask.
	UMask UMaskType //nolint:gochecknoglobals // Used by UMaskType Get and Set.

	// BaseDirs are the base directories present in a file system.
	BaseDirs = []struct { //nolint:gochecknoglobals // Used by CreateBaseDirs and TestCreateBaseDirs.
		Path string
		Perm os.FileMode
	}{
		{Path: avfs.HomeDir, Perm: 0o755},
		{Path: avfs.RootDir, Perm: 0o700},
		{Path: avfs.TmpDir, Perm: 0o777},
	}
)

// CreateBaseDirs creates base directories on a file system.
func CreateBaseDirs(vfs avfs.VFS, basePath string) error {
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
func CreateHomeDir(vfs avfs.VFS, u avfs.UserReader) (avfs.UserReader, error) {
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

// IsExist returns a boolean indicating whether the error is known to report
// that a file or directory already exists. It is satisfied by ErrExist as
// well as some syscall errors.
func IsExist(err error) bool {
	return errors.Is(err, avfs.ErrFileExists)
}

// IsNotExist returns a boolean indicating whether the error is known to
// report that a file or directory does not exist. It is satisfied by
// ErrNotExist as well as some syscall errors.
func IsNotExist(err error) bool {
	return errors.Is(err, avfs.ErrNoSuchFileOrDir)
}

// ErrRndTreeOutOfRange defines the generic error for out of range parameters RndTreeParams.
type ErrRndTreeOutOfRange string

var (
	ErrDepthOutOfRange    = ErrRndTreeOutOfRange("depth")
	ErrNameOutOfRange     = ErrRndTreeOutOfRange("name")
	ErrDirsOutOfRange     = ErrRndTreeOutOfRange("dirs")
	ErrFilesOutOfRange    = ErrRndTreeOutOfRange("files")
	ErrFileLenOutOfRange  = ErrRndTreeOutOfRange("file length")
	ErrSymlinksOutOfRange = ErrRndTreeOutOfRange("symbolic links")
)

func (e ErrRndTreeOutOfRange) Error() string {
	return string(e) + " parameter out of range"
}

// RndTreeParams defines the parameters to generate a random file system tree
// of directories, files and symbolic links.
type RndTreeParams struct {
	MinDepth    int // MinDepth is the minimum depth of the file system tree (must be >= 1).
	MaxDepth    int // MaxDepth is the maximum depth of the file system tree (must be >= MinDepth).
	MinName     int // MinName is the minimum length of a name (must be >= 1).
	MaxName     int // MaxName is the minimum length of a name (must be >= MinName).
	MinDirs     int // MinDirs is the minimum number of directories of a parent directory (must be >= 0).
	MaxDirs     int // MaxDirs is the maximum number of directories of a parent directory (must be >= MinDirs).
	MinFiles    int // MinFiles is the minimum number of files of a parent directory (must be >= 0).
	MaxFiles    int // MaxFiles is the maximum number of Files of a parent directory (must be >= MinFiles).
	MinFileLen  int // MinFileLen is minimum size of a file (must be >= 0).
	MaxFileLen  int // MaxFileLen is maximum size of a file (must be >= MinFileLen).
	MinSymlinks int // MinSymlinks is the minimum number of symbolic links of a parent directory (must be >= 0).
	MaxSymlinks int // MaxSymlinks is the maximum number of symbolic links of a parent directory (must be >= MinSymlinks).
	MaxTreeSize int // MaxTreeSize is the cumulative maximum size of the files in bytes.
}

// RndTree is a random file system tree generator of directories, files and symbolic links.
type RndTree struct {
	vfs           avfs.VFS // virtual file system.
	Dirs          []string // all directories.
	Files         []string // all files.
	SymLinks      []string // all symbolic links.
	treeSize      int      // the tree size in bytes.
	RndTreeParams          // parameters of the tree.
}

// NewRndTree returns a new random tree generator.
func NewRndTree(vfs avfs.VFS, p *RndTreeParams) (*RndTree, error) {
	if p.MinDepth < 1 || p.MinDepth > p.MaxDepth {
		return nil, ErrRndTreeOutOfRange("depth")
	}

	if p.MinName < 1 || p.MinName > p.MaxName {
		return nil, ErrRndTreeOutOfRange("name")
	}

	if p.MinDirs < 0 || p.MinDirs > p.MaxDirs {
		return nil, ErrRndTreeOutOfRange("dirs")
	}

	if p.MinFiles < 0 || p.MinFiles > p.MaxFiles {
		return nil, ErrRndTreeOutOfRange("files")
	}

	if p.MinFileLen < 0 || p.MinFileLen > p.MaxFileLen {
		return nil, ErrRndTreeOutOfRange("file length")
	}

	if p.MinSymlinks < 0 || p.MinSymlinks > p.MaxSymlinks {
		return nil, ErrRndTreeOutOfRange("symbolic links")
	}

	if p.MaxTreeSize < 0 {
		return nil, ErrRndTreeOutOfRange("maximum size")
	}

	rt := &RndTree{
		RndTreeParams: *p,
		vfs:           vfs,
	}

	return rt, nil
}

// CreateTree creates a random tree structure on a given path.
func (rt *RndTree) CreateTree(path string) error {
	return rt.randTree(path, 1)
}

// randTree generates recursively a random subtree of directories, files and symbolic links.
func (rt *RndTree) randTree(parent string, depth int) error {
	dirs, err := rt.randDirs(parent)
	if err != nil {
		return err
	}

	rt.Dirs = append(rt.Dirs, dirs...)

	files, err := rt.randFiles(parent)
	if err != nil {
		return err
	}

	rt.Files = append(rt.Files, files...)

	if rt.vfs.HasFeature(avfs.FeatSymlink) {
		symLinks, err := rt.randSymlinks(parent) //nolint:govet // Shadows previous declaration of err.
		if err != nil {
			return err
		}

		rt.SymLinks = append(rt.SymLinks, symLinks...)
	}

	maxDepth := rt.MinDepth
	if rt.MinDepth < rt.MaxDepth {
		maxDepth += rand.Intn(rt.MaxDepth - rt.MinDepth) //nolint:gosec // No security-sensitive function.
	}

	depth++
	if depth > maxDepth {
		return nil
	}

	for _, dir := range dirs {
		err = rt.randTree(dir, depth)
		if err != nil {
			return err
		}
	}

	return nil
}

// randDirs generates random directories from a parent directory.
func (rt *RndTree) randDirs(parent string) ([]string, error) {
	const dirSize = 1024

	nbDirs := rt.MinDirs
	if rt.MinDirs < rt.MaxDirs {
		nbDirs += rand.Intn(rt.MaxDirs - rt.MinDirs) //nolint:gosec // No security-sensitive function.
	}

	dirs := make([]string, 0, nbDirs)

	for i := 0; i < nbDirs; i++ {
		if rt.MaxTreeSize > 0 && (rt.treeSize+nbDirs*dirSize > rt.MaxTreeSize) {
			return dirs, nil
		}

		rt.treeSize += nbDirs * dirSize

		path := rt.vfs.Join(parent, randName(rt.MinName, rt.MaxName))

		err := rt.vfs.Mkdir(path, avfs.DefaultDirPerm)
		if err != nil {
			return nil, err
		}

		dirs = append(dirs, path)
	}

	return dirs, nil
}

// randFiles generates random files from a parent directory.
func (rt *RndTree) randFiles(parent string) ([]string, error) {
	nbFiles := rt.MinFiles
	if rt.MinFiles < rt.MaxFiles {
		nbFiles += rand.Intn(rt.MaxFiles - rt.MinFiles) //nolint:gosec // No security-sensitive function.
	}

	files := make([]string, 0, nbFiles)

	for i := 0; i < nbFiles; i++ {
		size := rt.MinFileLen
		if rt.MinFileLen < rt.MaxFileLen {
			size += rand.Intn(rt.MaxFileLen - rt.MinFileLen) //nolint:gosec // No security-sensitive function.
		}

		if rt.MaxTreeSize > 0 && (rt.treeSize > rt.MaxTreeSize) {
			return files, nil
		}

		rt.treeSize += size
		buf := make([]byte, size)
		rand.Read(buf) //nolint:gosec // No security-sensitive function.

		path := rt.vfs.Join(parent, randName(rt.MinName, rt.MaxName))

		err := rt.vfs.WriteFile(path, buf, avfs.DefaultFilePerm)
		if err != nil {
			return nil, err
		}

		files = append(files, path)
	}

	return files, nil
}

// randSymlinks generates random symbolic links from a parent directory.
func (rt *RndTree) randSymlinks(parent string) ([]string, error) {
	const slSize = 1024

	if len(rt.Files) == 0 {
		return []string{}, nil
	}

	nbSl := rt.MinSymlinks
	if rt.MinSymlinks < rt.MaxSymlinks {
		nbSl += rand.Intn(rt.MaxSymlinks - rt.MinSymlinks) //nolint:gosec // No security-sensitive function.
	}

	sl := make([]string, 0, nbSl)

	for i := 0; i < nbSl; i++ {
		if rt.MaxTreeSize > 0 && (rt.treeSize+slSize > rt.MaxTreeSize) {
			return sl, nil
		}

		newName := rt.vfs.Join(parent, randName(rt.MinName, rt.MaxName))
		fileIdx := rand.Intn(len(rt.Files)) //nolint:gosec // No security-sensitive function.
		oldName := rt.Files[fileIdx]

		err := rt.vfs.Symlink(oldName, newName)
		if err != nil {
			return nil, err
		}

		sl = append(sl, newName)
	}

	return sl, nil
}

// randName generates a random name using different sets of runes (ASCII, Cyrillic, Devanagari).
func randName(minName, maxName int) string {
	nbR := minName
	if minName < maxName {
		nbR += rand.Intn(maxName - minName) //nolint:gosec // No security-sensitive function.
	}

	var name strings.Builder

	for i, s, e := 0, 0, 0; i < nbR; i++ {
		switch rand.Intn(4) { //nolint:gosec // No security-sensitive function.
		case 0: // ASCII Uppercase
			s = 65
			e = 90
		case 1: // ASCII Lowercase
			s = 97
			e = 122
		case 2: // Cyrillic
			s = 0x400
			e = 0x4ff
		case 3: // Devanagari
			s = 0x900
			e = 0x97f
		}

		r := rune(s + rand.Intn(e-s)) //nolint:gosec // No security-sensitive function.

		name.WriteRune(r)
	}

	return name.String()
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
func SegmentPath(path string, start int) (end int, isLast bool) {
	pos := strings.IndexRune(path[start:], avfs.PathSeparator)
	if pos != -1 {
		return start + pos, false
	}

	return len(path), true
}

// DummySysStat implements SysStater interface returned by os.FileInfo.Sys() for.
type DummySysStat struct{}

// Gid returns the group id.
func (sst *DummySysStat) Gid() int {
	return 0
}

// Uid returns the user id.
func (sst *DummySysStat) Uid() int {
	return 0
}

// Nlink returns the number of hard links.
func (sst *DummySysStat) Nlink() uint64 {
	return 1
}
