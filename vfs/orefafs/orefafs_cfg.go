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

// New returns a new memory file system (OrefaFS) with the default Options.
func New() (*OrefaFS, error) {
	return NewWithOptions(nil)
}

// NewWithOptions returns a new memory file system (OrefaFS) with the selected Options.
func NewWithOptions(opts *Options) (*OrefaFS, error) {
	if opts == nil {
		opts = &Options{}
	}

	features := avfs.FeatHardlink | avfs.BuildFeatures()
	idm := avfs.DefaultIdm
	user := opts.User

	vfs := &OrefaFS{storage: &Storage{name: opts.Name}}

	_ = vfs.SetUMask(avfs.UMask())

	// The current directory of a file system is the home directory of its
	// user (see avfs.UserDirMixin.Init), which is created with the system
	// directories below when it is the one of the administrator.
	_ = vfs.userDir.Init(opts.OSType, idm, user)

	_ = vfs.SetFeatures(features)

	vfs.err = avfs.ErrorsFor(vfs.OSType())

	volumeName := ""
	if vfs.OSType() == avfs.OsWindows {
		volumeName = avfs.DefaultVolume
	}

	vfs.storage.nodes = make(nodes)
	vfs.storage.nodes[volumeName] = &node{
		mode:  fs.ModeDir | 0o755,
		mtime: time.Now(),
		uid:   0,
		gid:   0,
	}

	if len(opts.SystemDirs) == 0 {
		opts.SystemDirs = avfs.SystemDirs(vfs)
	}

	err := avfs.MkDirs(vfs, opts.SystemDirs, "")
	if err != nil {
		return nil, err
	}

	return vfs, nil
}

// Name returns the name of the fileSystem.
func (vfs *OrefaFS) Name() string {
	return vfs.storage.name
}

// Type returns the type of the fileSystem or Identity manager.
func (*OrefaFS) Type() string {
	return "OrefaFS"
}
