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
	"github.com/avfs/avfs/vfsutils"
)

// New returns a new OsFS file system.
func New(opts ...Option) (*OsFS, error) {
	vfs := &OsFS{
		idm:     dummyidm.NotImplementedIdm,
		feature: avfs.FeatBasicFs | avfs.FeatMainDirs,
	}

	if vfsutils.RunTimeOS() != avfs.OsWindows {
		vfs.feature |= avfs.FeatChroot | avfs.FeatHardlink | avfs.FeatSymlink
	}

	for _, opt := range opts {
		err := opt(vfs)
		if err != nil {
			return nil, err
		}
	}

	return vfs, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *OsFS) Features() avfs.Feature {
	return vfs.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (vfs *OsFS) HasFeature(feature avfs.Feature) bool {
	return vfs.feature&feature == feature
}

// Name returns the name of the fileSystem.
func (vfs *OsFS) Name() string {
	return ""
}

// OSType returns the operating system type of the file system.
func (vfs *OsFS) OSType() avfs.OSType {
	return vfsutils.RunTimeOS()
}

// Type returns the type of the fileSystem or Identity manager.
func (vfs *OsFS) Type() string {
	return "OsFS"
}

// Options

// WithIdm returns a function setting the identity manager for the file system.
func WithIdm(idm avfs.IdentityMgr) Option {
	return func(vfs *OsFS) error {
		vfs.idm = idm
		vfs.feature |= idm.Features()

		return nil
	}
}
