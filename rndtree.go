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
	"strconv"
)

// RndTreeOpts defines the parameters to generate a random file system tree
// of directories, files and symbolic links.
type RndTreeOpts struct {
	NbDirs      int // NbDirs is the number of directories.
	NbFiles     int // NbFiles is the number of files.
	NbSymlinks  int // NbSymlinks is the number of symbolic links.
	MaxFileSize int // MaxFileSize is maximum size of a file.
	MaxDepth    int // MaxDepth is the maximum depth of the tree.
}

// RndTreeDir contains parameters to create a directory.
type RndTreeDir struct {
	Name  string
	Depth int
}

// RndTreeFile contains parameters to create a file.
type RndTreeFile struct {
	Name string
	Size int
}

// RndTreeSymLink contains parameters to create a symbolic link.
type RndTreeSymLink struct {
	OldName, NewName string
}

// RndTree is a random file system tree generator of directories, files and symbolic links.
type RndTree struct {
	vfs         VFSBase           // vfs is the virtual file system.
	rnd         *rand.Rand        // rnd is a source of random numbers.
	dirs        []*RndTreeDir     // Dirs contains all directories.
	files       []*RndTreeFile    // Files contains all files.
	symLinks    []*RndTreeSymLink // SymLinks contains all symbolic links.
	RndTreeOpts                   // RndTreeOpts regroups the options of the tree.
}

// NewRndTree returns a new random tree generator.
func NewRndTree(vfs VFSBase, opts *RndTreeOpts) *RndTree {
	if opts.NbDirs < 0 {
		opts.NbDirs = 0
	}

	if opts.NbFiles < 0 {
		opts.NbFiles = 0
	}

	if opts.NbSymlinks < 0 {
		opts.NbSymlinks = 0
	}

	if opts.MaxDepth < 0 {
		opts.MaxDepth = 0
	}

	if opts.MaxFileSize < 0 {
		opts.MaxFileSize = 0
	}

	rt := &RndTree{
		vfs: vfs,
		rnd: rand.New(rand.NewSource(42)),
		RndTreeOpts: RndTreeOpts{
			NbDirs:      opts.NbDirs,
			NbFiles:     opts.NbFiles,
			NbSymlinks:  opts.NbSymlinks,
			MaxFileSize: opts.MaxFileSize,
			MaxDepth:    opts.MaxDepth,
		},
	}

	return rt
}

// GenTree generates a random tree and populates RndTree.Dirs, RndTree.Files and RndTree.SymLinks.
func (rt *RndTree) GenTree() {
	nameIdx := 0
	name := func(prefix string) string {
		nameIdx++

		return prefix + "-" + strconv.Itoa(nameIdx)
	}

	if rt.dirs != nil {
		return
	}

	nbDirs := rt.NbDirs
	dirs := make([]*RndTreeDir, nbDirs)

	parents := make([]*RndTreeDir, 1, 10)
	parents[0] = &RndTreeDir{}

	for i := 0; i < nbDirs; i++ {
		parent := parents[rand.Intn(len(parents))]
		path := parent.Name + "/" + name("dir")
		depth := parent.Depth + 1

		dir := &RndTreeDir{Name: path, Depth: depth}
		dirs[i] = dir

		if depth < rt.MaxDepth {
			parents = append(parents, dir)
		}
	}

	rt.dirs = dirs

	if rt.NbFiles == 0 {
		return
	}

	nbParents := len(parents)
	nbFiles := rt.NbFiles
	files := make([]*RndTreeFile, nbFiles)

	for i := 0; i < nbFiles; i++ {
		parent := parents[rand.Intn(nbParents)]
		fileName := parent.Name + "/" + name("file")

		size := 0
		if rt.MaxFileSize > 0 {
			size = rand.Intn(rt.MaxFileSize)
		}

		file := &RndTreeFile{Name: fileName, Size: size}
		files[i] = file
	}

	rt.files = files

	if !rt.vfs.HasFeature(FeatSymlink) {
		return
	}

	nbSymlinks := rt.NbSymlinks
	symLinks := make([]*RndTreeSymLink, nbSymlinks)

	for i := 0; i < nbSymlinks; i++ {
		oldName := files[rand.Intn(nbFiles)].Name
		newDir := parents[rand.Intn(nbParents)].Name
		newName := newDir + "/" + name("symlink")

		sl := &RndTreeSymLink{OldName: oldName, NewName: newName}
		symLinks[i] = sl
	}

	rt.symLinks = symLinks
}

// CreateDirs creates random directories.
func (rt *RndTree) CreateDirs(baseDir string) error {
	vfs := rt.vfs

	rt.GenTree()

	for _, dir := range rt.dirs {
		path := vfs.Join(baseDir, dir.Name)

		err := vfs.MkdirAll(path, DefaultDirPerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateFiles creates random files.
func (rt *RndTree) CreateFiles(baseDir string) error {
	err := rt.CreateDirs(baseDir)
	if err != nil {
		return err
	}

	buf := make([]byte, rt.MaxFileSize)
	rt.rnd.Read(buf)

	vfs := rt.vfs

	for _, file := range rt.files {
		path := vfs.Join(baseDir, file.Name)

		err = vfs.WriteFile(path, buf[:file.Size], DefaultFilePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateSymlinks creates random symbolic links.
func (rt *RndTree) CreateSymlinks(baseDir string) error {
	err := rt.CreateFiles(baseDir)
	if err != nil {
		return err
	}

	vfs := rt.vfs
	if !vfs.HasFeature(FeatSymlink) {
		return nil
	}

	for _, symlink := range rt.symLinks {
		oldPath := vfs.Join(baseDir, symlink.OldName)
		newPath := vfs.Join(baseDir, symlink.NewName)

		err = vfs.Symlink(oldPath, newPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateTree creates a random tree structure.
func (rt *RndTree) CreateTree(baseDir string) error {
	return rt.CreateSymlinks(baseDir)
}

func (rt *RndTree) Dirs() []*RndTreeDir {
	return rt.dirs
}

func (rt *RndTree) Files() []*RndTreeFile {
	return rt.files
}

func (rt *RndTree) SymLinks() []*RndTreeSymLink {
	return rt.symLinks
}
