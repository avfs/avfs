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

package dummyfs

import (
	"github.com/avfs/avfs"
)

// New creates a new DummyFS file system.
func New() (*DummyFS, error) {
	return &DummyFS{}, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *DummyFS) Features() avfs.Feature {
	return 0
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (vfs *DummyFS) HasFeature(feature avfs.Feature) bool {
	return false
}

// Name returns the name of the fileSystem.
func (vfs *DummyFS) Name() string {
	return avfs.NotImplemented
}

// OSType returns the operating system type of the file system.
func (vfs *DummyFS) OSType() avfs.OSType {
	return avfs.OsLinux
}

// Type returns the type of the fileSystem or Identity manager.
func (vfs *DummyFS) Type() string {
	return "DummyFS"
}
