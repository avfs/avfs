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
	"io/fs"
	"time"

	"github.com/avfs/avfs"
)

// New returns a new memory file system (MemFS).
func New(opts ...Option) *MemFS {
	ma := &memAttrs{
		idm: avfs.NotImplementedIdm,
		feature: avfs.FeatBasicFs |
			avfs.FeatChroot |
			avfs.FeatHardlink |
			avfs.FeatSymlink,
		umask: int32(avfs.UMask.Get()),
	}

	vfs := &MemFS{
		user: avfs.NotImplementedUser,
		rootNode: &dirNode{
			baseNode: baseNode{
				mtime: time.Now().UnixNano(),
				mode:  fs.ModeDir | 0o755,
				uid:   0,
				gid:   0,
			},
		},
		memAttrs: ma,
	}

	for _, opt := range opts {
		opt(vfs)
	}

	if vfs.utils.OSType() == avfs.OsUnknown {
		vfs.utils = avfs.NewUtils(avfs.OsLinux)
	}

	if vfs.HasFeature(avfs.FeatMainDirs) {
		u := vfs.user
		um := ma.umask

		vfs.user = avfs.AdminUser
		ma.umask = 0

		err := vfs.utils.CreateBaseDirs(vfs, "")
		if err != nil {
			panic("CreateBaseDirs " + err.Error())
		}

		ma.umask = um
		vfs.user = u
		vfs.curDir = vfs.utils.HomeDirUser(u.Name())
	}

	return vfs
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *MemFS) Features() avfs.Feature {
	return vfs.memAttrs.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (vfs *MemFS) HasFeature(feature avfs.Feature) bool {
	return vfs.memAttrs.feature&feature == feature
}

// Name returns the name of the fileSystem.
func (vfs *MemFS) Name() string {
	return vfs.memAttrs.name
}

// OSType returns the operating system type of the file system.
func (vfs *MemFS) OSType() avfs.OSType {
	return vfs.utils.OSType()
}

// Type returns the type of the fileSystem or Identity manager.
func (vfs *MemFS) Type() string {
	return "MemFS"
}

// Options

// WithMainDirs returns an option function to create main directories.
func WithMainDirs() Option {
	return func(vfs *MemFS) {
		vfs.memAttrs.feature |= avfs.FeatMainDirs
	}
}

// WithIdm returns an option function which sets the identity manager.
func WithIdm(idm avfs.IdentityMgr) Option {
	return func(vfs *MemFS) {
		vfs.memAttrs.idm = idm
		vfs.memAttrs.feature |= idm.Features()
		vfs.user = idm.AdminUser()
	}
}

// WithName returns an option function which sets the name of the file system.
func WithName(name string) Option {
	return func(vfs *MemFS) {
		vfs.memAttrs.name = name
	}
}

// WithOS returns an option function which sets the OS type.
func WithOS(osType avfs.OSType) Option {
	return func(vfs *MemFS) {
		vfs.utils = avfs.NewUtils(osType)
	}
}
