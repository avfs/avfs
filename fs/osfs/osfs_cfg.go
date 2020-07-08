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
)

// New returns a new OsFs file system.
func New(opts ...Option) (*OsFs, error) {
	fs := &OsFs{
		idm: dummyidm.NotImplementedIdm,
		feature: avfs.FeatBasicFs |
			avfs.FeatChroot |
			avfs.FeatMainDirs |
			avfs.FeatHardlink |
			avfs.FeatSymlink,
	}

	for _, opt := range opts {
		err := opt(fs)
		if err != nil {
			return nil, err
		}
	}

	return fs, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (fs *OsFs) Features() avfs.Feature {
	return fs.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (fs *OsFs) HasFeature(feature avfs.Feature) bool {
	return fs.feature&feature == feature
}

// Name returns the name of the fileSystem.
func (fs *OsFs) Name() string {
	return ""
}

// Type returns the type of the fileSystem or Identity manager.
func (fs *OsFs) Type() string {
	return "OsFs"
}

// Options

// OptIdm returns a function setting the identity manager for the file system.
func OptIdm(idm avfs.IdentityMgr) Option {
	return func(fs *OsFs) error {
		fs.idm = idm
		fs.feature |= idm.Features()

		return nil
	}
}
