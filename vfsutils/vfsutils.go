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

// Error returns an ErrRndTreeOutOfRange error.
func (e ErrRndTreeOutOfRange) Error() string {
	return string(e) + " parameter out of range"
}

var (
	ErrNameOutOfRange     = ErrRndTreeOutOfRange("name")
	ErrDirsOutOfRange     = ErrRndTreeOutOfRange("dirs")
	ErrFilesOutOfRange    = ErrRndTreeOutOfRange("files")
	ErrFileSizeOutOfRange = ErrRndTreeOutOfRange("file size")
	ErrSymlinksOutOfRange = ErrRndTreeOutOfRange("symbolic links")
)

// RndTreeParams defines the parameters to generate a random file system tree
// of directories, files and symbolic links.
type RndTreeParams struct {
	MinName     int  // MinName is the minimum length of a name (must be >= 1).
	MaxName     int  // MaxName is the minimum length of a name (must be >= MinName).
	MinDirs     int  // MinDirs is the minimum number of directories (must be >= 0).
	MaxDirs     int  // MaxDirs is the maximum number of directories (must be >= MinDirs).
	MinFiles    int  // MinFiles is the minimum number of files (must be >= 0).
	MaxFiles    int  // MaxFiles is the maximum number of Files (must be >= MinFiles).
	MinFileSize int  // MinFileSize is minimum size of a file (must be >= 0).
	MaxFileSize int  // MaxFileSize is maximum size of a file (must be >= MinFileSize).
	MinSymlinks int  // MinSymlinks is the minimum number of symbolic links (must be >= 0).
	MaxSymlinks int  // MaxSymlinks is the maximum number of symbolic links (must be >= MinSymlinks).
	OneLevel    bool //
}

// RndTree is a random file system tree generator of directories, files and symbolic links.
type RndTree struct {
	vfs           avfs.VFS // virtual file system.
	baseDir       string   //
	Dirs          []string // all directories.
	Files         []string // all files.
	SymLinks      []string // all symbolic links.
	RndTreeParams          // parameters of the tree.
}

// NewRndTree returns a new random tree generator.
func NewRndTree(vfs avfs.VFS, baseDir string, p *RndTreeParams) (*RndTree, error) {
	_, err := vfs.Stat(baseDir)
	if err != nil {
		return nil, err
	}

	if p.MinName < 1 || p.MinName > p.MaxName {
		return nil, ErrNameOutOfRange
	}

	if p.MinDirs < 0 || p.MinDirs > p.MaxDirs {
		return nil, ErrDirsOutOfRange
	}

	if p.MinFiles < 0 || p.MinFiles > p.MaxFiles {
		return nil, ErrFilesOutOfRange
	}

	if p.MinFileSize < 0 || p.MinFileSize > p.MaxFileSize {
		return nil, ErrFileSizeOutOfRange
	}

	if p.MinSymlinks < 0 || p.MinSymlinks > p.MaxSymlinks {
		return nil, ErrSymlinksOutOfRange
	}

	rt := &RndTree{
		vfs:           vfs,
		baseDir:       baseDir,
		RndTreeParams: *p,
	}

	return rt, nil
}

// CreateTree creates a random tree structure.
func (rt *RndTree) CreateTree() error {
	return rt.CreateSymlinks()
}

// CreateDirs creates random directories.
func (rt *RndTree) CreateDirs() error {
	if rt.Dirs != nil {
		return nil
	}

	nbDirs := randRange(rt.MinDirs, rt.MaxDirs)
	rt.Dirs = make([]string, nbDirs)
	vfs := rt.vfs

	for i := 0; i < nbDirs; i++ {
		parent := rt.RandDir(i)
		newDir := vfs.Join(parent, rt.RandName())

		err := vfs.Mkdir(newDir, avfs.DefaultDirPerm)
		if err != nil {
			return err
		}

		rt.Dirs[i] = newDir
	}

	return nil
}

// CreateFiles generates random files from already created directories.
func (rt *RndTree) CreateFiles() error {
	if rt.Files != nil {
		return nil
	}

	if rt.Dirs == nil {
		err := rt.CreateDirs()
		if err != nil {
			return err
		}
	}

	nbFiles := randRange(rt.MinFiles, rt.MaxFiles)
	rt.Files = make([]string, nbFiles)

	buf := make([]byte, rt.MaxFileSize)
	rand.Read(buf) //nolint:gosec // No security-sensitive function.

	vfs := rt.vfs

	for i := 0; i < nbFiles; i++ {
		parent := rt.RandDir(len(rt.Dirs))
		newFile := vfs.Join(parent, rt.RandName())
		size := randRange(rt.MinFileSize, rt.MaxFileSize)

		err := vfs.WriteFile(newFile, buf[:size], avfs.DefaultFilePerm)
		if err != nil {
			return err
		}

		rt.Files[i] = newFile
	}

	return nil
}

// CreateSymlinks creates random symbolic links from existing random files and directories.
func (rt *RndTree) CreateSymlinks() error {
	if rt.SymLinks != nil {
		return nil
	}

	if rt.Files == nil {
		err := rt.CreateFiles()
		if err != nil {
			return err
		}
	}

	vfs := rt.vfs
	if !vfs.HasFeature(avfs.FeatSymlink) {
		return nil
	}

	nbSymlinks := randRange(rt.MinSymlinks, rt.MaxSymlinks)
	rt.SymLinks = make([]string, nbSymlinks)

	for i := 0; i < nbSymlinks; i++ {
		fileIdx := rand.Intn(len(rt.Files)) //nolint:gosec // No security-sensitive function.
		oldName := rt.Files[fileIdx]
		parent := rt.RandDir(len(rt.Dirs))
		newName := vfs.Join(parent, rt.RandName())

		err := rt.vfs.Symlink(oldName, newName)
		if err != nil {
			return err
		}

		rt.SymLinks[i] = newName
	}

	return nil
}

// RandDir returns a random directory.
func (rt *RndTree) RandDir(max int) string {
	if rt.OneLevel || max <= 0 {
		return rt.baseDir
	}

	return rt.Dirs[rand.Intn(max)] //nolint:gosec // No security-sensitive function.
}

// RandName generates a random name.
func (rt *RndTree) RandName() string {
	return randName(rt.MinName, rt.MaxName)
}

// randName generates a random name using different sets of runes (ASCII, Cyrillic, Devanagari).
func randName(minName, maxName int) string {
	nbRunes := randRange(minName, maxName)

	var name strings.Builder

	for i, s, e := 0, 0, 0; i < nbRunes; i++ {
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

// randRange returns a random integer between min and max.
func randRange(min, max int) int {
	val := min
	if min < max {
		val += rand.Intn(max - min) //nolint:gosec // No security-sensitive function.
	}

	return val
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
