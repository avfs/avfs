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

package memfs

import (
	"os"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
	"github.com/avfs/avfs/idm/dummyidm"
)

// New returns a new memory file system (MemFs).
func New(opts ...Option) (*MemFs, error) {
	fsa := &fsAttrs{
		idm: dummyidm.NotImplementedIdm,
		feature: avfs.FeatBasicFs |
			avfs.FeatChroot |
			avfs.FeatClonable |
			avfs.FeatHardlink |
			avfs.FeatSymlink,
		umask: int32(fsutil.UMask.Get()),
	}

	fs := &MemFs{
		rootNode: &dirNode{
			baseNode: baseNode{
				mtime: time.Now().UnixNano(),
				mode:  os.ModeDir | 0o755,
				uid:   0,
				gid:   0,
			},
		},
		fsAttrs: fsa,
		user:    dummyidm.RootUser,
		curDir:  string(avfs.PathSeparator),
	}

	for _, opt := range opts {
		err := opt(fs)
		if err != nil {
			return nil, err
		}
	}

	if fs.fsAttrs.feature&avfs.FeatMainDirs != 0 {
		um := fsa.umask
		fsa.umask = 0

		_ = fsutil.CreateBaseDirs(fs)

		fsa.umask = um
		fs.curDir = avfs.RootDir
	}

	return fs, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (fs *MemFs) Features() avfs.Feature {
	return fs.fsAttrs.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (fs *MemFs) HasFeature(feature avfs.Feature) bool {
	return fs.fsAttrs.feature&feature == feature
}

// Name returns the name of the fileSystem.
func (fs *MemFs) Name() string {
	return fs.fsAttrs.name
}

// Type returns the type of the fileSystem or Identity manager.
func (fs *MemFs) Type() string {
	return "MemFs"
}

// Options

// OptMainDirs returns an option function to create main directories.
func OptMainDirs() Option {
	return func(fs *MemFs) error {
		fs.fsAttrs.feature |= avfs.FeatMainDirs
		return nil
	}
}

// OptIdm returns an option function which sets the identity manager.
func OptIdm(idm avfs.IdentityMgr) Option {
	return func(fs *MemFs) error {
		u, err := idm.LookupUser(avfs.UsrRoot)
		if err != nil {
			return err
		}

		fs.fsAttrs.idm = idm
		fs.fsAttrs.feature |= idm.Features()
		fs.user = u

		return nil
	}
}

// OptName returns an option function which sets the name of the file system.
func OptName(name string) Option {
	return func(fs *MemFs) error {
		fs.fsAttrs.name = name

		return nil
	}
}

func OptAbsPath() Option {
	return func(fs *MemFs) error {
		fs.fsAttrs.feature |= avfs.FeatAbsPath

		return nil
	}
}
