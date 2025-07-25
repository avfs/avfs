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
	"github.com/avfs/avfs/idm/osidm"
)

// New returns a new OS file system with the default Options.
// Don't use this for a production environment, prefer NewWithNoIdm.
func New() *OsFS {
	return NewWithOptions(&Options{Idm: osidm.New()})
}

// NewWithNoIdm returns a new OS file system with no identity management.
// Use this for production environments.
func NewWithNoIdm() *OsFS {
	return NewWithOptions(&Options{})
}

// NewWithOptions returns a new memory file system (MemFS) with the selected Options.
func NewWithOptions(opts *Options) *OsFS {
	if opts == nil {
		opts = &Options{}
	}

	idm := opts.Idm
	if idm == nil {
		idm = avfs.NotImplementedIdm
	}

	features := avfs.FeatRealFS | avfs.FeatSymlink | avfs.FeatHardlink | idm.Features()
	vfs := &OsFS{}

	_ = vfs.SetFeatures(features)
	_ = vfs.SetIdm(idm)
	vfs.err = avfs.ErrorsFor(vfs.OSType())

	return vfs
}

// Name returns the name of the fileSystem.
func (*OsFS) Name() string {
	return ""
}

// Type returns the type of the fileSystem or Identity manager.
func (*OsFS) Type() string {
	return "OsFS"
}
