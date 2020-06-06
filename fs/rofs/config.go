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

// HasFeatures returns true if the file system provides all the given features.
func (fs *RoFs) HasFeatures(feature avfs.Feature) bool {
	return (feature&avfs.FeatReadOnly != 0) || fs.baseFs.HasFeatures(feature)
}

// Name returns the name of the fileSystem.
func (fs *RoFs) Name() string {
	return fs.baseFs.Name()
}

// Type returns the type of the fileSystem or Identity manager.
func (fs *RoFs) Type() string {
	return "RoFs"
}
