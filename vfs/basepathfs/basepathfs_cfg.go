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
func New(baseFs avfs.VFS, basePath string) (*BasePathFS, error) {
	const op = "basepath"

	absPath, _ := baseFs.Abs(basePath)

	info, err := baseFs.Stat(absPath)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: basePath, Err: errors.Unwrap(err)}
	}

	if !info.IsDir() {
		return nil, &fs.PathError{Op: op, Path: basePath, Err: avfs.ErrNotADirectory}
	}

	vfs := &BasePathFS{
		baseFS:   baseFs,
		basePath: absPath,
		feature:  baseFs.Features() &^ (avfs.FeatSymlink | avfs.FeatChroot),
	}

	if baseFs.HasFeature(avfs.FeatMainDirs) {
		err = avfs.CreateBaseDirs(vfs.baseFS, vfs.basePath)
		if err != nil {
			return nil, err
		}
	}

	return vfs, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *BasePathFS) Features() avfs.Feature {
	return vfs.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (vfs *BasePathFS) HasFeature(feature avfs.Feature) bool {
	return (vfs.feature & feature) == feature
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
