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
	"sync"
	"time"

	"github.com/avfs/avfs"
)

// New returns a new memory file system (OrefaFS).
func New(opts ...Option) (*OrefaFS, error) {
	vfs := &OrefaFS{
		currentUser: avfs.RootUser,
		nodes:       make(nodes),
		curDir:      string(avfs.PathSeparator),
		feature:     avfs.FeatBasicFs | avfs.FeatHardlink,
		mu:          sync.RWMutex{},
		umask:       int32(avfs.UMask.Get()),
	}

	vfs.nodes[string(avfs.PathSeparator)] = &node{
		mtime: time.Now().UnixNano(),
		mode:  fs.ModeDir | 0o755,
	}

	for _, opt := range opts {
		err := opt(vfs)
		if err != nil {
			return nil, err
		}
	}

	if vfs.feature&avfs.FeatMainDirs != 0 {
		um := vfs.umask
		vfs.umask = 0

		_ = avfs.CreateBaseDirs(vfs, "")

		vfs.umask = um
		vfs.curDir = avfs.RootDir
	}

	return vfs, nil
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

// Type returns the type of the fileSystem or Identity manager.
func (vfs *OrefaFS) Type() string {
	return "OrefaFS"
}

// Options

// WithMainDirs returns an option function to create main directories (/home, /root and /tmp).
func WithMainDirs() Option {
	return func(vfs *OrefaFS) error {
		vfs.feature |= avfs.FeatMainDirs

		return nil
	}
}
