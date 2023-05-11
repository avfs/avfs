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

import (
	"math"

	"github.com/avfs/avfs"
)

// New creates a new OsIdm identity manager.
func New() *OsIdm {
	osType := avfs.CurrentOSType()
	uid, gid := 0, 0
	GroupName, UserName := avfs.AdminGroupName(osType), avfs.AdminUserName(osType)
	features := avfs.FeatIdentityMgr

	if !isUserAdmin() {
		features |= avfs.FeatReadOnlyIdm
	}

	if osType == avfs.OsWindows {
		features = 0
		uid, gid = math.MaxInt, math.MaxInt
		GroupName, UserName = avfs.DefaultName, avfs.DefaultName
	}

	adminGroup := &OsGroup{name: GroupName, gid: gid}
	adminUser := &OsUser{name: UserName, uid: uid, gid: gid}

	idm := &OsIdm{
		adminGroup: adminGroup,
		adminUser:  adminUser,
		features:   features,
	}

	return idm
}

// OSType returns the operating system type of the identity manager.
func (idm *OsIdm) OSType() avfs.OSType {
	return avfs.CurrentOSType()
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *OsIdm) Type() string {
	return "OsIdm"
}

// Features returns the set of features provided by the file system or identity manager.
func (idm *OsIdm) Features() avfs.Features {
	return idm.features
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (idm *OsIdm) HasFeature(feature avfs.Features) bool {
	return (idm.features & feature) == feature
}
