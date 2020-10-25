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

package rofs

import "github.com/avfs/avfs"

// New creates a new readonly file system (RoFs) from a base file system.
func New(baseFs avfs.Fs) *RoFs {
	return &RoFs{baseFs: baseFs}
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *RoFs) Features() avfs.Feature {
	return (vfs.baseFs.Features() &^ avfs.FeatIdentityMgr) | avfs.FeatReadOnly
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (vfs *RoFs) HasFeature(feature avfs.Feature) bool {
	return ((vfs.baseFs.Features()|avfs.FeatReadOnly)&^avfs.FeatIdentityMgr)&feature == feature
}

// Name returns the name of the fileSystem.
func (vfs *RoFs) Name() string {
	return vfs.baseFs.Name()
}

// OSType returns the operating system type of the file system.
func (vfs *RoFs) OSType() avfs.OSType {
	return vfs.baseFs.OSType()
}

// Type returns the type of the fileSystem or Identity manager.
func (vfs *RoFs) Type() string {
	return "RoFs"
}
