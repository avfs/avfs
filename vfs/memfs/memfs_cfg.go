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
	"github.com/avfs/avfs/idm/dummyidm"
	"github.com/avfs/avfs/vfsutils"
)

// New returns a new memory file system (MemFS).
func New(opts ...Option) (*MemFS, error) {
	fsa := &fsAttrs{
		idm: dummyidm.NotImplementedIdm,
		feature: avfs.FeatBasicFs |
			avfs.FeatChroot |
			avfs.FeatClonable |
			avfs.FeatHardlink |
			avfs.FeatSymlink,
		umask: int32(vfsutils.UMask.Get()),
	}

	vfs := &MemFS{
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
		err := opt(vfs)
		if err != nil {
			return nil, err
		}
	}

	if vfs.fsAttrs.feature&avfs.FeatMainDirs != 0 {
		um := fsa.umask
		fsa.umask = 0

		_ = vfsutils.CreateBaseDirs(vfs, "")

		fsa.umask = um
		vfs.curDir = avfs.RootDir
	}

	return vfs, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *MemFS) Features() avfs.Feature {
	return vfs.fsAttrs.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (vfs *MemFS) HasFeature(feature avfs.Feature) bool {
	return vfs.fsAttrs.feature&feature == feature
}

// Name returns the name of the fileSystem.
func (vfs *MemFS) Name() string {
	return vfs.fsAttrs.name
}

// OSType returns the operating system type of the file system.
func (vfs *MemFS) OSType() avfs.OSType {
	return avfs.OsLinux
}

// Type returns the type of the fileSystem or Identity manager.
func (vfs *MemFS) Type() string {
	return "MemFS"
}

// Options

// WithMainDirs returns an option function to create main directories.
func WithMainDirs() Option {
	return func(vfs *MemFS) error {
		vfs.fsAttrs.feature |= avfs.FeatMainDirs

		return nil
	}
}

// WithIdm returns an option function which sets the identity manager.
func WithIdm(idm avfs.IdentityMgr) Option {
	return func(vfs *MemFS) error {
		u, err := idm.LookupUser(avfs.UsrRoot)
		if err != nil {
			return err
		}

		vfs.fsAttrs.idm = idm
		vfs.fsAttrs.feature |= idm.Features()
		vfs.user = u

		return nil
	}
}

// WithName returns an option function which sets the name of the file system.
func WithName(name string) Option {
	return func(vfs *MemFS) error {
		vfs.fsAttrs.name = name

		return nil
	}
}

func WithAbsPath() Option {
	return func(vfs *MemFS) error {
		vfs.fsAttrs.feature |= avfs.FeatAbsPath

		return nil
	}
}
