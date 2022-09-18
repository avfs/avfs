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

package rofs

import (
	"io/fs"

	"github.com/avfs/avfs"
)

// New creates a new readonly file system (RoFS) from a base file system.
func New(baseFS avfs.VFS) *RoFS {
	vfs := &RoFS{
		baseFS:            baseFS,
		errOpNotPermitted: avfs.ErrOpNotPermitted,
		errPermDenied:     avfs.ErrPermDenied,
		features:          baseFS.Features()&^avfs.FeatIdentityMgr | avfs.FeatReadOnly,
	}

	if baseFS.OSType() == avfs.OsWindows {
		vfs.errOpNotPermitted = avfs.ErrWinNotSupported
		vfs.errPermDenied = avfs.ErrWinAccessDenied
	}

	return vfs
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *RoFS) Features() avfs.Features {
	return vfs.features
}

// HasFeature returns true if the file system or identity manager provides a given features.
func (vfs *RoFS) HasFeature(feature avfs.Features) bool {
	return vfs.features&feature == feature
}

// Name returns the name of the fileSystem.
func (vfs *RoFS) Name() string {
	return vfs.baseFS.Name()
}

// OSType returns the operating system type of the file system.
func (vfs *RoFS) OSType() avfs.OSType {
	return vfs.baseFS.OSType()
}

// Type returns the type of the fileSystem or Identity manager.
func (*RoFS) Type() string {
	return "RoFS"
}

// Configuration functions.

// CreateSystemDirs creates the system directories of a file system.
func (vfs *RoFS) CreateSystemDirs(basePath string) error {
	const op = "mkdir"

	return &fs.PathError{Op: op, Path: basePath, Err: vfs.errPermDenied}
}

// CreateHomeDir creates and returns the home directory of a user.
// If there is an error, it will be of type *PathError.
func (vfs *RoFS) CreateHomeDir(u avfs.UserReader) (string, error) {
	const op = "mkdir"

	return "", &fs.PathError{Op: op, Path: u.Name(), Err: vfs.errPermDenied}
}

// HomeDirUser returns the home directory of the user.
// If the file system does not have an identity manager, the root directory is returned.
func (vfs *RoFS) HomeDirUser(u avfs.UserReader) string {
	return vfs.baseFS.HomeDirUser(u)
}

// SystemDirs returns the system directories of the file system.
func (vfs *RoFS) SystemDirs(basePath string) []avfs.DirInfo {
	return vfs.baseFS.SystemDirs(basePath)
}
