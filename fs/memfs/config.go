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
	fs := &MemFs{
		rootNode: &dirNode{
			baseNode: baseNode{
				mode:  os.ModeDir | 0o755,
				mtime: time.Now().UnixNano(),
				uid:   0,
				gid:   0,
			},
		},
		curDir: string(avfs.PathSeparator),
		feature: avfs.FeatBasicFs |
			avfs.FeatChroot |
			avfs.FeatHardlink |
			avfs.FeatInescapableChroot |
			avfs.FeatSymlink,
		idm:   dummyidm.NotImplementedIdm,
		umask: int32(fsutil.UMask.Get()),
		user:  dummyidm.RootUser,
	}

	for _, opt := range opts {
		err := opt(fs)
		if err != nil {
			return nil, err
		}
	}

	if fs.feature&avfs.FeatMainDirs != 0 {
		um := fs.umask
		fs.umask = 0

		_ = fs.createDir(fs.rootNode, avfs.HomeDir[1:], 0o755)
		_ = fs.createDir(fs.rootNode, avfs.RootDir[1:], 0o700)
		_ = fs.createDir(fs.rootNode, avfs.TmpDir[1:], 0o1777)

		fs.umask = um
		fs.curDir = avfs.RootDir
	}

	return fs, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (fs *MemFs) Features() avfs.Feature {
	return fs.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (fs *MemFs) HasFeature(feature avfs.Feature) bool {
	return fs.feature&feature == feature
}

// Name returns the name of the fileSystem.
func (fs *MemFs) Name() string {
	return fs.name
}

// Type returns the type of the fileSystem or Identity manager.
func (fs *MemFs) Type() string {
	return "MemFs"
}

// Options

// OptMainDirs returns an option function to create main directories.
func OptMainDirs() Option {
	return func(fs *MemFs) error {
		fs.feature |= avfs.FeatMainDirs
		return nil
	}
}

// OptIdm returns an option function which sets the identity manager.
func OptIdm(idm avfs.IdentityMgr) Option {
	return func(fs *MemFs) error {
		fs.idm = idm

		u, err := idm.LookupUser(avfs.UsrRoot)
		if err != nil {
			return err
		}

		fs.feature |= idm.Features()
		fs.user = u

		return nil
	}
}

// OptName returns an option function which sets the name of the file system.
func OptName(name string) Option {
	return func(fs *MemFs) error {
		fs.name = name

		return nil
	}
}

// OptAbsPath returns an option function which sets
func OptAbsPath() Option {
	return func(fs *MemFs) error {
		fs.feature |= avfs.FeatAbsPath

		return nil
	}
}
