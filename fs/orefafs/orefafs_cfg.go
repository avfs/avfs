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
	"os"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
)

// New returns a new memory file system (OrefaFs).
func New(opts ...Option) (*OrefaFs, error) {
	fs := &OrefaFs{
		nodes:   make(nodes),
		curDir:  string(avfs.PathSeparator),
		umask:   int32(fsutil.UMask.Get()),
		feature: avfs.FeatBasicFs,
		osType:  fsutil.RunTimeOS(),
	}

	fs.nodes[string(avfs.PathSeparator)] = &node{
		mtime: time.Now().UnixNano(),
		mode:  os.ModeDir | 0o755,
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

		_ = fsutil.CreateBaseDirs(fs)

		fs.umask = um
		fs.curDir = avfs.RootDir
	}

	return fs, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (fs *OrefaFs) Features() avfs.Feature {
	return fs.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (fs *OrefaFs) HasFeature(feature avfs.Feature) bool {
	return fs.feature&feature == feature
}

// Name returns the name of the fileSystem.
func (fs *OrefaFs) Name() string {
	return fs.name
}

// OSType returns the operating system type of the file system.
func (fs *OrefaFs) OSType() avfs.OSType {
	return fs.osType
}

// Type returns the type of the fileSystem or Identity manager.
func (fs *OrefaFs) Type() string {
	return "OrefaFs"
}

// Options

// OptMainDirs returns an option function to create main directories (/home, /root and /tmp).
func OptMainDirs() Option {
	return func(fs *OrefaFs) error {
		fs.feature |= avfs.FeatMainDirs

		return nil
	}
}
