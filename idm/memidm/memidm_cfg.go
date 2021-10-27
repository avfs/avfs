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

import (
	"github.com/avfs/avfs"
)

// New creates a new identity manager.
func New(opts ...Option) *MemIdm {
	idm := &MemIdm{
		groupsByName: make(groupsByName),
		groupsById:   make(groupsById),
		usersByName:  make(usersByName),
		usersById:    make(usersById),
		feature:      avfs.FeatIdentityMgr,
		maxGid:       minGid,
		maxUid:       minUid,
		utils:        avfs.Cfg.Utils(),
	}

	for _, opt := range opts {
		opt(idm)
	}

	adminGroupName := idm.utils.AdminGroupName()
	adminUserName := idm.utils.AdminUserName()

	idm.adminGroup = &MemGroup{
		name: adminGroupName,
		gid:  0,
	}

	idm.adminUser = &MemUser{
		name: adminUserName,
		uid:  0,
		gid:  0,
	}

	idm.groupsById[0] = idm.adminGroup
	idm.groupsByName[adminGroupName] = idm.adminGroup
	idm.usersById[0] = idm.adminUser
	idm.usersByName[adminUserName] = idm.adminUser

	return idm
}

// Type returns the type of the Identity manager.
func (idm *MemIdm) Type() string {
	return "MemIdm"
}

// Features returns the set of features provided by the file system or identity manager.
func (idm *MemIdm) Features() avfs.Features {
	return avfs.FeatIdentityMgr
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (idm *MemIdm) HasFeature(feature avfs.Features) bool {
	return (idm.feature & feature) == feature
}

// Options

// WithOSType returns a function setting the OS type of the file system.
func WithOSType(ost avfs.OSType) Option {
	return func(idm *MemIdm) {
		idm.utils = avfs.NewUtils(ost)
	}
}
