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

package osfs

import (
	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/dummyidm"
	"github.com/avfs/avfs/idm/osidm"
)

// New returns a new OS file system with the default Options.
// Don't use this for a production environment, prefer NewWithNoIdm.
func New() *OsFS {
	return NewWithOptions(nil)
}

// NewWithNoIdm returns a new OS file system with no identity management.
// Use this for production environments.
func NewWithNoIdm() *OsFS {
	return NewWithOptions(&Options{Idm: dummyidm.NotImplementedIdm})
}

// NewWithOptions returns a new memory file system (MemFS) with the selected Options.
func NewWithOptions(opts *Options) *OsFS {
	if opts == nil {
		opts = &Options{Idm: osidm.New()}
	}

	features := avfs.FeatRealFS | avfs.FeatSystemDirs | avfs.FeatSymlink | avfs.FeatHardlink | opts.Idm.Features()
	vfs := &OsFS{
		idm: opts.Idm,
	}

	_ = vfs.SetFeatures(features)
	_ = vfs.SetOSType(avfs.CurrentOSType())
	vfs.setErrors()

	return vfs
}

// setErrors sets OsFS errors depending on the operating system.
func (vfs *OsFS) setErrors() {
	switch vfs.OSType() {
	case avfs.OsWindows:
		vfs.err.PermDenied = avfs.ErrWinAccessDenied
	default:
		vfs.err.PermDenied = avfs.ErrPermDenied
	}
}

// Name returns the name of the fileSystem.
func (*OsFS) Name() string {
	return ""
}

// Type returns the type of the fileSystem or Identity manager.
func (*OsFS) Type() string {
	return "OsFS"
}

// Configuration functions.

// CreateSystemDirs creates the system directories of a file system.
func (vfs *OsFS) CreateSystemDirs(basePath string) error {
	return vfs.Utils.CreateSystemDirs(vfs, basePath)
}

// CreateHomeDir creates and returns the home directory of a user.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) CreateHomeDir(u avfs.UserReader) (string, error) {
	return vfs.Utils.CreateHomeDir(vfs, u)
}
