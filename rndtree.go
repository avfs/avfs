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

package avfs

import (
	"math/rand"
	"strings"
)

// RndTreeError defines the generic error for out of range parameters RndTreeParams.
type RndTreeError string

// Error returns an RndTreeError error.
func (e RndTreeError) Error() string {
	return string(e) + " parameter out of range"
}

var (
	// ErrNameOutOfRange is the error when MinName or MaxName are out of range.
	ErrNameOutOfRange = RndTreeError("name")

	// ErrDirsOutOfRange is the error when MinDirs or MaxDirs are out of range.
	ErrDirsOutOfRange = RndTreeError("dirs")

	// ErrFilesOutOfRange is the error when MinFiles or MaxFiles are out of range.
	ErrFilesOutOfRange = RndTreeError("files")

	// ErrFileSizeOutOfRange is the error when MinFileSize or MaxFileSize are out of range.
	ErrFileSizeOutOfRange = RndTreeError("file size")

	// ErrSymlinksOutOfRange is the error when MinSymlinks or MaxSymlinks are out of range.
	ErrSymlinksOutOfRange = RndTreeError("symbolic links")
)

// RndTreeParams defines the parameters to generate a random file system tree
// of directories, files and symbolic links.
type RndTreeParams struct {
	MinName     int   // MinName is the minimum length of a name (must be >= 1).
	MaxName     int   // MaxName is the minimum length of a name (must be >= MinName).
	MinDirs     int   // MinDirs is the minimum number of directories (must be >= 0).
	MaxDirs     int   // MaxDirs is the maximum number of directories (must be >= MinDirs).
	MinFiles    int   // MinFiles is the minimum number of files (must be >= 0).
	MaxFiles    int   // MaxFiles is the maximum number of Files (must be >= MinFiles).
	MinFileSize int   // MinFileSize is minimum size of a file (must be >= 0).
	MaxFileSize int   // MaxFileSize is maximum size of a file (must be >= MinFileSize).
	MinSymlinks int   // MinSymlinks is the minimum number of symbolic links (must be >= 0).
	MaxSymlinks int   // MaxSymlinks is the maximum number of symbolic links (must be >= MinSymlinks).
	OneLevel    bool  // OneLevel set tree depth to one level.
	RandSeed    int64 // RandSeed is the seed used by random generating number functions.
}

// FileParams contains parameters to create a file.
type FileParams struct {
	Name string
	size int
}

// SymLinkParams contains parameters to create a symbolic link.
type SymLinkParams struct {
	OldName, NewName string
}

// RndTree is a random file system tree generator of directories, files and symbolic links.
type RndTree struct {
	vfs           VFSBase          // vfs is the virtual file system.
	rnd           *rand.Rand       // rnd is a source of random numbers.
	baseDir       string           // baseDir is the base directory of the random tree.
	Dirs          []string         // Dirs contains all directories.
	Files         []*FileParams    // Files contains all files.
	SymLinks      []*SymLinkParams // SymLinks contains all symbolic links.
	RndTreeParams                  // RndTreeParams regroups the parameters of the tree.
}

// NewRndTree returns a new random tree generator.
func NewRndTree(vfs VFSBase, baseDir string, p *RndTreeParams) (*RndTree, error) {
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

	if p.RandSeed == 0 {
		p.RandSeed = 42
	}

	rt := &RndTree{
		vfs:           vfs,
		baseDir:       baseDir,
		rnd:           rand.New(rand.NewSource(p.RandSeed)),
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
	vfs := rt.vfs

	err := vfs.MkdirAll(rt.baseDir, DefaultDirPerm)
	if err != nil {
		return err
	}

	if len(rt.Dirs) == 0 {
		nbDirs := rt.randRange(rt.MinDirs, rt.MaxDirs)
		rt.Dirs = make([]string, nbDirs)
	}

	for i, dirName := range rt.Dirs {
		if dirName == "" {
			parent := rt.randDir(i)
			dirName = vfs.Join(parent, rt.randName())
			rt.Dirs[i] = dirName
		}

		err = vfs.MkdirAll(dirName, DefaultDirPerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateFiles creates random files.
func (rt *RndTree) CreateFiles() error {
	err := rt.CreateDirs()
	if err != nil {
		return err
	}

	if len(rt.Files) == 0 {
		nbFiles := rt.randRange(rt.MinFiles, rt.MaxFiles)
		rt.Files = make([]*FileParams, nbFiles)
	}

	vfs := rt.vfs
	nbDirs := len(rt.Dirs)
	buf := make([]byte, rt.MaxFileSize)
	rt.rnd.Read(buf)

	for i, file := range rt.Files {
		if file == nil {
			parent := rt.randDir(nbDirs)
			fileName := vfs.Join(parent, rt.randName())
			size := rt.randRange(rt.MinFileSize, rt.MaxFileSize)
			file = &FileParams{Name: fileName, size: size}
			rt.Files[i] = file
		}

		err = vfs.WriteFile(file.Name, buf[:file.size], DefaultFilePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateSymlinks creates random symbolic links.
func (rt *RndTree) CreateSymlinks() error {
	err := rt.CreateFiles()
	if err != nil {
		return err
	}

	vfs := rt.vfs
	if !vfs.HasFeature(FeatSymlink) {
		return nil
	}

	if len(rt.SymLinks) == 0 {
		nbSymlinks := rt.randRange(rt.MinSymlinks, rt.MaxSymlinks)
		rt.SymLinks = make([]*SymLinkParams, nbSymlinks)
	}

	for i, symlink := range rt.SymLinks {
		if symlink == nil {
			oldName := rt.randFile(len(rt.Files)).Name
			newName := vfs.Join(rt.randDir(len(rt.Dirs)), rt.randName())
			symlink = &SymLinkParams{OldName: oldName, NewName: newName}
			rt.SymLinks[i] = symlink
		}

		oldName, err := vfs.EvalSymlinks(symlink.NewName)
		if err == nil && oldName == symlink.OldName {
			continue
		}

		err = vfs.Symlink(symlink.OldName, symlink.NewName)
		if err != nil {
			return err
		}
	}

	return nil
}

// randDir returns a random directory.
func (rt *RndTree) randDir(max int) string {
	if rt.OneLevel || max <= 0 {
		return rt.baseDir
	}

	return rt.Dirs[rt.rnd.Intn(max)]
}

// randFile returns a random file.
func (rt *RndTree) randFile(max int) *FileParams {
	return rt.Files[rt.rnd.Intn(max)]
}

// randName generates a random name using different sets of runes (ASCII, Cyrillic, Devanagari).
func (rt *RndTree) randName() string {
	const maxAlphabets = 4

	var name strings.Builder

	nbRunes := rt.randRange(rt.MinName, rt.MaxName)

	for i := 0; i < nbRunes; i++ {
		var s, e int

		switch rt.rnd.Intn(maxAlphabets) {
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

		r := rune(s + rt.rnd.Intn(e-s))

		name.WriteRune(r)
	}

	return name.String()
}

// randRange returns a random integer between min and max.
func (rt *RndTree) randRange(min, max int) int {
	val := min
	if min < max {
		val += rt.rnd.Intn(max - min)
	}

	return val
}
