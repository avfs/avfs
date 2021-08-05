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

package osidm

import "github.com/avfs/avfs"

// New creates a new OsIdm identity manager.
func New() *OsIdm {
	var feature avfs.Feature

	switch avfs.RunTimeOS() {
	case avfs.OsWindows:
		feature = 0
	default:
		feature = avfs.FeatIdentityMgr
	}

	return &OsIdm{
		feature:  feature,
		initUser: currentUser(),
	}
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *OsIdm) Type() string {
	return "OsIdm"
}

// Features returns the set of features provided by the file system or identity manager.
func (idm *OsIdm) Features() avfs.Feature {
	return idm.feature
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (idm *OsIdm) HasFeature(feature avfs.Feature) bool {
	return idm.feature == feature
}
