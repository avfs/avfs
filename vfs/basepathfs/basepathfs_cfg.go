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
		features: baseFS.Features() &^ avfs.FeatSymlink,
	}

	vfs.InitUtils(avfs.CurrentOSType())

	if baseFS.HasFeature(avfs.FeatSystemDirs) {
		err = vfs.baseFS.CreateSystemDirs(vfs.basePath)
		if err != nil {
			panic(err)
		}
	}

	return vfs
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *BasePathFS) Features() avfs.Features {
	return vfs.features
}

// HasFeature returns true if the file system or identity manager provides a given features.
func (vfs *BasePathFS) HasFeature(feature avfs.Features) bool {
	return (vfs.features & feature) == feature
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
func (vfs *BasePathFS) Type() string {
	return "BasePathFS"
}

// Configuration functions.

// CreateSystemDirs creates the system directories of a file system.
func (vfs *BasePathFS) CreateSystemDirs(basePath string) error {
	return vfs.Utils.CreateSystemDirs(vfs, basePath)
}

// CreateHomeDir creates and returns the home directory of a user.
// If there is an error, it will be of type *PathError.
func (vfs *BasePathFS) CreateHomeDir(u avfs.UserReader) (string, error) {
	return vfs.Utils.CreateHomeDir(vfs, u)
}
