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

// ErrRndTreeOutOfRange defines the generic error for out of range parameters RndTreeParams.
type ErrRndTreeOutOfRange string

// Error returns an ErrRndTreeOutOfRange error.
func (e ErrRndTreeOutOfRange) Error() string {
	return string(e) + " parameter out of range"
}

var (
	// ErrNameOutOfRange is the error when MinName or MaxName is out of range.
	ErrNameOutOfRange = ErrRndTreeOutOfRange("name")

	// ErrDirsOutOfRange is the error when MinDirs or MaxDirs is out of range.
	ErrDirsOutOfRange = ErrRndTreeOutOfRange("dirs")

	// ErrFilesOutOfRange is the error when MinFiles or MaxFiles is out of range.
	ErrFilesOutOfRange = ErrRndTreeOutOfRange("files")

	// ErrFileSizeOutOfRange is the error when MinFileSize or MaxFileSize is out of range.
	ErrFileSizeOutOfRange = ErrRndTreeOutOfRange("file size")

	// ErrSymlinksOutOfRange is the error when MinSymlinks or MaxSymlinks is out of range.
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

// SymLinkParams contains parameters to create a symbolic link.
type SymLinkParams struct {
	OldName, NewName string
}

// RndTree is a random file system tree generator of directories, files and symbolic links.
type RndTree struct {
	vfs           VFS             // virtual file system.
	baseDir       string          // base directory of the random tree.
	Dirs          []string        // all directories.
	Files         []string        // all files.
	SymLinks      []SymLinkParams // all symbolic links.
	RndTreeParams                 // parameters of the tree.
}

// NewRndTree returns a new random tree generator.
func NewRndTree(vfs VFS, baseDir string, p *RndTreeParams) (*RndTree, error) {
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

	rt.generateDirs()
	rt.generateFiles()
	rt.generateSymlinks()

	return rt, nil
}

// generateDirs generates random directories.
func (rt *RndTree) generateDirs() {
	nbDirs := randRange(rt.MinDirs, rt.MaxDirs)
	rt.Dirs = make([]string, nbDirs)
	vfs := rt.vfs

	for i := 0; i < nbDirs; i++ {
		parent := rt.randDir(i)
		newDir := vfs.Join(parent, rt.randName())
		rt.Dirs[i] = newDir
	}
}

// generateFiles generates random files from existing directories.
func (rt *RndTree) generateFiles() {
	nbFiles := randRange(rt.MinFiles, rt.MaxFiles)
	rt.Files = make([]string, nbFiles)
	vfs := rt.vfs

	for i := 0; i < nbFiles; i++ {
		parent := rt.randDir(len(rt.Dirs))
		newFile := vfs.Join(parent, rt.randName())
		rt.Files[i] = newFile
	}
}

// generateSymlinks generate random symbolic links from existing random files and directories.
func (rt *RndTree) generateSymlinks() {
	vfs := rt.vfs
	if !vfs.HasFeature(FeatSymlink) {
		return
	}

	nbSymlinks := randRange(rt.MinSymlinks, rt.MaxSymlinks)
	rt.SymLinks = make([]SymLinkParams, nbSymlinks)

	for i := 0; i < nbSymlinks; i++ {
		oldName := rt.randFile(len(rt.Files))
		newName := vfs.Join(rt.randDir(len(rt.Dirs)), rt.randName())
		rt.SymLinks[i] = SymLinkParams{OldName: oldName, NewName: newName}
	}
}

// CreateTree creates a random tree structure.
func (rt *RndTree) CreateTree() error {
	err := rt.CreateDirs()
	if err != nil {
		return err
	}

	err = rt.CreateFiles()
	if err != nil {
		return err
	}

	return rt.CreateSymlinks()
}

// CreateDirs creates random directories.
func (rt *RndTree) CreateDirs() error {
	vfs := rt.vfs

	err := vfs.MkdirAll(rt.baseDir, DefaultDirPerm)
	if err != nil {
		return err
	}

	for _, dirName := range rt.Dirs {
		err = vfs.Mkdir(dirName, DefaultDirPerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateFiles creates random files.
func (rt *RndTree) CreateFiles() error {
	buf := make([]byte, rt.MaxFileSize)
	rand.Read(buf)

	vfs := rt.vfs

	for _, fileName := range rt.Files {
		size := randRange(rt.MinFileSize, rt.MaxFileSize)

		err := vfs.WriteFile(fileName, buf[:size], DefaultFilePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateSymlinks creates random symbolic links.
func (rt *RndTree) CreateSymlinks() error {
	vfs := rt.vfs
	if !vfs.HasFeature(FeatSymlink) {
		return nil
	}

	for _, symlink := range rt.SymLinks {
		err := vfs.Symlink(symlink.OldName, symlink.NewName)
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

	return rt.Dirs[rand.Intn(max)]
}

// randFile returns a random file.
func (rt *RndTree) randFile(max int) string {
	return rt.Files[rand.Intn(max)]
}

// randName generates a random name.
func (rt *RndTree) randName() string {
	return randName(rt.MinName, rt.MaxName)
}

// randRange returns a random integer between min and max.
func randRange(min, max int) int {
	val := min
	if min < max {
		val += rand.Intn(max - min)
	}

	return val
}

// randName generates a random name using different sets of runes (ASCII, Cyrillic, Devanagari).
func randName(minName, maxName int) string {
	nbRunes := randRange(minName, maxName)

	var name strings.Builder

	for i, s, e := 0, 0, 0; i < nbRunes; i++ {
		switch rand.Intn(4) {
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

		r := rune(s + rand.Intn(e-s))

		name.WriteRune(r)
	}

	return name.String()
}
