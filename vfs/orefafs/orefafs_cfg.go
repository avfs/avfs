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
		nodes:   make(nodes),
		curDir:  string(avfs.PathSeparator),
		feature: avfs.FeatBasicFs | avfs.FeatHardlink,
		umask:   int32(avfs.UMask.Get()),
		utils:   avfs.OsUtils,
	}

	vfs.nodes[string(avfs.PathSeparator)] = &node{
		mtime: time.Now().UnixNano(),
		mode:  fs.ModeDir | 0o755,
	}

	for _, opt := range opts {
		opt(vfs)
	}

	vfs.currentUser = vfs.Idm().AdminUser()

	if vfs.feature&avfs.FeatMainDirs != 0 {
		um := vfs.umask
		vfs.umask = 0

		_ = vfs.utils.CreateBaseDirs(vfs, "")

		vfs.umask = um
		vfs.curDir = string(avfs.PathSeparator)
	}

	return vfs
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *OrefaFS) Features() avfs.Feature {
	return vfs.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (vfs *OrefaFS) HasFeature(feature avfs.Feature) bool {
	return vfs.feature&feature == feature
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
		vfs.feature |= avfs.FeatMainDirs
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
		vfs.feature |= avfs.FeatChownUser
	}
}
