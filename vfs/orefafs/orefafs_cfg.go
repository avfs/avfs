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

package orefafs

import (
	"io/fs"
	"time"

	"github.com/avfs/avfs"
)

// New returns a new memory file system (OrefaFS).
func New(opts ...Option) *OrefaFS {
	vfs := &OrefaFS{
		nodes:    make(nodes),
		features: avfs.FeatBasicFs | avfs.FeatHardlink,
		umask:    int32(avfs.Cfg.UMask()),
		utils:    avfs.Cfg.Utils(),
	}

	for _, opt := range opts {
		opt(vfs)
	}

	volumeName := string(avfs.PathSeparator)
	if vfs.utils.OSType() == avfs.OsWindows {
		volumeName = "C:\\"
	}

	rootNode := &node{
		mtime: time.Now().UnixNano(),
		mode:  fs.ModeDir | 0o755,
	}

	vfs.nodes[volumeName] = rootNode
	vfs.user = avfs.AdminUser
	vfs.curDir = volumeName

	if vfs.features&avfs.FeatMainDirs != 0 {
		um := vfs.umask
		vfs.umask = 0

		err := vfs.utils.CreateBaseDirs(vfs, volumeName)
		if err != nil {
			panic("CreateBaseDirs " + err.Error())
		}

		vfs.umask = um
	}

	return vfs
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *OrefaFS) Features() avfs.Features {
	return vfs.features
}

// HasFeature returns true if the file system or identity manager provides a given features.
func (vfs *OrefaFS) HasFeature(feature avfs.Features) bool {
	return vfs.features&feature == feature
}

// Name returns the name of the fileSystem.
func (vfs *OrefaFS) Name() string {
	return vfs.name
}

// OSType returns the operating system type of the file system.
func (vfs *OrefaFS) OSType() avfs.OSType {
	return vfs.utils.OSType()
}

// Type returns the type of the fileSystem or Identity manager.
func (vfs *OrefaFS) Type() string {
	return "OrefaFS"
}

// Options

// WithMainDirs returns an option function to create main directories (/home, /root and /tmp).
func WithMainDirs() Option {
	return func(vfs *OrefaFS) {
		vfs.features |= avfs.FeatMainDirs
	}
}

// WithOSType returns a function setting the OS type of the file system.
func WithOSType(ost avfs.OSType) Option {
	return func(idm *OrefaFS) {
		idm.utils = avfs.NewUtils(ost)
	}
}

// WithChownUser returns an option function.
func WithChownUser() Option {
	return func(vfs *OrefaFS) {
		vfs.features |= avfs.FeatChownUser
	}
}
