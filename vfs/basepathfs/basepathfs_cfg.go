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

package basepathfs

import (
	"errors"
	"io/fs"

	"github.com/avfs/avfs"
)

// New returns a new base path file system (BasePathFS).
func New(baseFS avfs.VFS, basePath string) *BasePathFS {
	const op = "basepath"

	absPath, _ := baseFS.Abs(basePath)

	info, err := baseFS.Stat(absPath)
	if err != nil {
		err = &fs.PathError{Op: op, Path: basePath, Err: errors.Unwrap(err)}
		panic(err)
	}

	if !info.IsDir() {
		err = &fs.PathError{Op: op, Path: basePath, Err: avfs.ErrNotADirectory}
		panic(err)
	}

	vfs := &BasePathFS{
		baseFS:   baseFS,
		basePath: absPath,
	}

	_ = vfs.SetFeatures(baseFS.Features() &^ avfs.FeatSymlink)

	return vfs
}

// Name returns the name of the fileSystem.
func (vfs *BasePathFS) Name() string {
	return vfs.baseFS.Name()
}

// OSType returns the operating system type of the file system.
func (vfs *BasePathFS) OSType() avfs.OSType {
	return vfs.baseFS.OSType()
}

// Type returns the type of the fileSystem or Identity manager.
func (*BasePathFS) Type() string {
	return "BasePathFS"
}
