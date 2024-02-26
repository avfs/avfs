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
func New() *OrefaFS {
	return NewWithOptions(nil)
}

// NewWithOptions returns a new memory file system (OrefaFS) with the selected Options.
func NewWithOptions(opts *Options) *OrefaFS {
	if opts == nil {
		opts = &Options{UMask: avfs.UMask(), OSType: avfs.OsUnknown}
	}

	features := avfs.FeatHardlink | avfs.BuildFeatures()
	idm := avfs.NotImplementedIdm

	user := opts.User
	if opts.User == nil {
		user = idm.AdminUser()
	}

	vfs := &OrefaFS{
		dirMode:  fs.ModeDir,
		fileMode: 0,
		lastId:   new(uint64),
		name:     opts.Name,
	}

	_ = vfs.SetFeatures(features)
	_ = vfs.SetOSType(opts.OSType)
	_ = vfs.SetUMask(opts.UMask)
	_ = vfs.SetIdm(idm)
	_ = vfs.SetUser(user)

	vfs.err.SetOSType(vfs.OSType())

	volumeName := ""
	curDir := "/"

	if vfs.OSType() == avfs.OsWindows {
		vfs.dirMode |= avfs.DefaultDirPerm
		vfs.fileMode |= avfs.DefaultFilePerm
		volumeName = avfs.DefaultVolume
		curDir = volumeName + string(vfs.PathSeparator())
	}

	vfs.nodes = make(nodes)
	vfs.nodes[volumeName] = &node{
		mode:  fs.ModeDir | 0o755,
		mtime: time.Now().UnixNano(),
		uid:   0,
		gid:   0,
	}

	_ = vfs.SetCurDir(curDir)

	if len(opts.SystemDirs) == 0 {
		opts.SystemDirs = avfs.SystemDirs(vfs, volumeName)
	}

	_ = avfs.MkSystemDirs(vfs, opts.SystemDirs)

	return vfs
}

// Name returns the name of the fileSystem.
func (vfs *OrefaFS) Name() string {
	return vfs.name
}

// Type returns the type of the fileSystem or Identity manager.
func (*OrefaFS) Type() string {
	return "OrefaFS"
}
