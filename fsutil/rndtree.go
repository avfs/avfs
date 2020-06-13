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

package fsutil

import (
	"math/rand"
	"strings"

	"github.com/avfs/avfs"
)

// ErrOutOfRange
type ErrOutOfRange string

// Error
func (e ErrOutOfRange) Error() string {
	return string(e) + " parameter out of range"
}

// RndTreeParams defines the parameters to generate
// a random file system tree of directories, files and symbolic links.
type RndTreeParams struct {
	// MinDepth is the minimum depth of the file system tree (must be >= 1).
	MinDepth int

	// MaxDepth is the maximum depth of the file system tree (must be >= MinDepth).
	MaxDepth int

	// MinName is the minimum length of a name (must be >= 1).
	MinName int

	// MaxName is the minimum length of a name (must be >= MinName).
	MaxName int

	// MinDirs is the minimum number of directories of a parent directory (must be >= 0).
	MinDirs int

	// MaxDirs is the maximum number of directories of a parent directory (must be >= MinDirs).
	MaxDirs int

	// MinFiles is the minimum number of files of a parent directory (must be >= 0).
	MinFiles int

	// MaxFiles is the maximum number of Files of a parent directory (must be >= MinFiles).
	MaxFiles int

	// MinFileLen is minimum size of a file (must be >= 0).
	MinFileLen int

	// MaxFileLen is maximum size of a file (must be >= MinFileLen).
	MaxFileLen int

	// MinSymlinks is the minimum number of symbolic links of a parent directory (must be >= 0).
	MinSymlinks int

	// MaxSymlinks is the maximum number of symbolic links of a parent directory (must be >= MinSymlinks).
	MaxSymlinks int
}

// RndTree is a random file system tree generator of directories, files and symbolic links.
type RndTree struct {
	RndTreeParams
	fs       avfs.Fs
	Dirs     []string
	Files    []string
	SymLinks []string
}

// NewRndTree returns a new random tree generator.
func NewRndTree(fs avfs.Fs, p RndTreeParams) (*RndTree, error) { //nolint:gocritic
	if p.MinDepth < 1 || p.MinDepth > p.MaxDepth {
		return nil, ErrOutOfRange("depth")
	}

	if p.MinName < 1 || p.MinName > p.MaxName {
		return nil, ErrOutOfRange("name")
	}

	if p.MinDirs < 0 || p.MinDirs > p.MaxDirs {
		return nil, ErrOutOfRange("dirs")
	}

	if p.MinFiles < 0 || p.MinFiles > p.MaxFiles {
		return nil, ErrOutOfRange("files")
	}

	if p.MinFileLen < 0 || p.MinFileLen > p.MaxFileLen {
		return nil, ErrOutOfRange("file length")
	}

	if p.MinSymlinks < 0 || p.MinSymlinks > p.MaxSymlinks {
		return nil, ErrOutOfRange("symbolic links")
	}

	rt := &RndTree{
		RndTreeParams: p,
		fs:            fs,
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

	if rt.fs.HasFeature(avfs.FeatSymlink) {
		symLinks, err := rt.randSymlinks(parent) //nolint:govet
		if err != nil {
			return err
		}

		rt.SymLinks = append(rt.SymLinks, symLinks...)
	}

	maxDepth := rt.MinDepth
	if rt.MinDepth < rt.MaxDepth {
		maxDepth += rand.Intn(rt.MaxDepth - rt.MinDepth)
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
	nbDirs := rt.MinDirs
	if rt.MinDirs < rt.MaxDirs {
		nbDirs += rand.Intn(rt.MaxDirs - rt.MinDirs)
	}

	dirs := make([]string, 0, nbDirs)

	for i := 0; i < nbDirs; i++ {
		path := rt.fs.Join(parent, randName(rt.MinName, rt.MaxName))

		err := rt.fs.Mkdir(path, avfs.DefaultDirPerm)
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
		nbFiles += rand.Intn(rt.MaxFiles - rt.MinFiles)
	}

	files := make([]string, 0, nbFiles)

	for i := 0; i < nbFiles; i++ {
		size := rt.MinFileLen
		if rt.MinFileLen < rt.MaxFileLen {
			size += rand.Intn(rt.MaxFileLen - rt.MinFileLen)
		}

		buf := make([]byte, size)
		rand.Read(buf) //nolint:gosec

		path := rt.fs.Join(parent, randName(rt.MinName, rt.MaxName))

		err := rt.fs.WriteFile(path, buf, avfs.DefaultFilePerm)
		if err != nil {
			return nil, err
		}

		files = append(files, path)
	}

	return files, nil
}

// randSymlinks generates random symbolic links from a parent directory.
func (rt *RndTree) randSymlinks(parent string) ([]string, error) {
	if len(rt.Files) == 0 {
		return []string{}, nil
	}

	nbSl := rt.MinSymlinks
	if rt.MinSymlinks < rt.MaxSymlinks {
		nbSl += rand.Intn(rt.MaxSymlinks - rt.MinSymlinks)
	}

	sl := make([]string, 0, nbSl)

	for i := 0; i < nbSl; i++ {
		newName := rt.fs.Join(parent, randName(rt.MinName, rt.MaxName))
		fileIdx := rand.Intn(len(rt.Files))
		oldName := rt.Files[fileIdx]

		err := rt.fs.Symlink(oldName, newName)
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
		nbR += rand.Intn(maxName - minName)
	}

	var name strings.Builder

	for i, s, e := 0, 0, 0; i < nbR; i++ {
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
