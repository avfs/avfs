//
//  Copyright 2021 The AVFS authors
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

package avfs

// NewBaseFS creates a new NewBaseFS file system.
func NewBaseFS() (*BaseFS, error) {
	vfs := &BaseFS{
		osType:        OsLinux,
		pathSeparator: PathSeparator,
	}

	return vfs, nil
}

// Features returns the set of features provided by the file system or identity manager.
func (vfs *BaseFS) Features() Feature {
	return 0
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (vfs *BaseFS) HasFeature(feature Feature) bool {
	return false
}

// Name returns the name of the fileSystem.
func (vfs *BaseFS) Name() string {
	return ""
}

// OSType returns the operating system type of the file system.
func (vfs *BaseFS) OSType() OSType {
	return vfs.osType
}

// Type returns the type of the fileSystem or Identity manager.
func (vfs *BaseFS) Type() string {
	return "BaseFS"
}
