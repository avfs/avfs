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

import "github.com/avfs/avfs"

// New creates a new DummyFs file system.
func New() (*DummyFs, error) {
	return &DummyFs{}, nil
}

// HasFeatures returns true if the file system provides all the given features.
func (fs *DummyFs) HasFeatures(feature avfs.Feature) bool {
	return false
}

// Name returns the name of the fileSystem.
func (fs *DummyFs) Name() string {
	return avfs.NotImplemented
}

// Type returns the type of the fileSystem or Identity manager.
func (fs *DummyFs) Type() string {
	return "DummyFs"
}
