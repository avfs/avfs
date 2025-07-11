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
	"github.com/avfs/avfs"
)

// New creates a new readonly file system (RoFS) from a base file system.
func New(baseFS avfs.VFS) *RoFS {
	vfs := &RoFS{baseFS: baseFS}

	_ = vfs.SetFeatures(baseFS.Features()&^avfs.FeatIdentityMgr | avfs.FeatReadOnly)
	vfs.err = avfs.ErrorsFor(vfs.OSType())

	return vfs
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
