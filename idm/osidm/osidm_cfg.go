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
	var (
		features   avfs.Features
		adminGroup *OsGroup
		adminUser  *OsUser
	)

	ost := avfs.Cfg.OSType()
	ut := avfs.NewUtils(ost)

	switch ost {
	case avfs.OsWindows:
		adminGroup = &OsGroup{name: ut.DefaultGroupName(), gid: avfs.DefaultGroup.Gid()}
		adminUser = &OsUser{name: ut.DefaultUserName(), uid: avfs.DefaultUser.Uid(), gid: avfs.DefaultUser.Gid()}
	default:
		adminGroup = &OsGroup{name: ut.AdminGroupName(), gid: 0}
		adminUser = &OsUser{name: ut.AdminUserName(), uid: 0, gid: 0}

		features = avfs.FeatIdentityMgr
		if !User().IsAdmin() {
			features |= avfs.FeatReadOnlyIdm
		}
	}

	idm := &OsIdm{
		adminGroup: adminGroup,
		adminUser:  adminUser,
		features:   features,
	}

	return idm
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
