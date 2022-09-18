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
)

// Open opens the named file for reading. If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func (vfs *MemIOFS) Open(name string) (fs.File, error) {
	return vfs.OpenFile(name, os.O_RDONLY, 0)
}

// Sub returns an FS corresponding to the subtree rooted at dir.
func (vfs *MemIOFS) Sub(dir string) (fs.FS, error) {
	const op = "sub"

	_, child, _, err := vfs.searchNode(dir, slmEval)
	if err != vfs.err.FileExists || child == nil {
		return nil, &fs.PathError{Op: op, Path: dir, Err: err}
	}

	c, ok := child.(*dirNode)
	if !ok {
		return nil, &fs.PathError{Op: op, Path: dir, Err: vfs.err.NotADirectory}
	}

	subFS := *vfs
	subFS.rootNode = c

	return &subFS, nil
}
