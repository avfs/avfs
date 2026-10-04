//
//  Copyright 2022 The AVFS authors
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

package memfs

import (
	"io/fs"
	"os"

	"github.com/avfs/avfs"
)

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode [O_RDONLY].
// If there is an error, it will be of type [*PathError].
func (vfs *MemIOFS) Open(name string) (fs.File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (vfs *MemIOFS) Sub(dir string) (fs.FS, error) {
	// vfs.MemFS.Sub, not vfs.Sub: this method shadows the embedded one.
	vfsSub, err := vfs.MemFS.Sub(dir)
	if err != nil {
		return nil, err
	}

	return &memIOPathFS{vfs: vfsSub}, nil
}

// memIOPathFS is the io/fs projection of a file system returned by Sub.
type memIOPathFS struct {
	vfs avfs.VFS // vfs is the file system of the subtree.
}

// Open opens the named file for reading.
func (pfs *memIOPathFS) Open(name string) (fs.File, error) {
	return pfs.vfs.Open(name)
}
