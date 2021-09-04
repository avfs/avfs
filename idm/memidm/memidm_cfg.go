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

package memidm

import "github.com/avfs/avfs"

// New creates a new identity manager.
func New() *MemIdm {
	groupRoot := &MemGroup{
		name: avfs.UsrRoot,
		gid:  0,
	}

	userRoot := &MemUser{
		name: avfs.UsrRoot,
		uid:  0,
		gid:  0,
	}

	idm := &MemIdm{
		groupsByName: make(groupsByName),
		groupsById:   make(groupsById),
		usersByName:  make(usersByName),
		usersById:    make(usersById),
		maxGid:       minGid,
		maxUid:       minUid,
	}

	idm.groupsById[0] = groupRoot
	idm.groupsByName[avfs.UsrRoot] = groupRoot
	idm.usersById[0] = userRoot
	idm.usersByName[avfs.UsrRoot] = userRoot

	return idm
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *MemIdm) Type() string {
	return "MemIdm"
}

// Features returns the set of features provided by the file system or identity manager.
func (idm *MemIdm) Features() avfs.Feature {
	return avfs.FeatIdentityMgr
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (idm *MemIdm) HasFeature(feature avfs.Feature) bool {
	return avfs.FeatIdentityMgr&feature == feature
}
