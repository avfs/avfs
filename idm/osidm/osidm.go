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

// Package osidm implements an identity manager using os functions.
//
// For testing only, should not be used in a production environment.
package osidm

import "github.com/avfs/avfs"

// AdminGroup returns the administrator (root) group.
func (idm *OsIdm) AdminGroup() avfs.GroupReader {
	return idm.adminGroup
}

// AdminUser returns the administrator (root) user.
func (idm *OsIdm) AdminUser() avfs.UserReader {
	return idm.adminUser
}

// OsGroup

// Gid returns the group ID.
func (g *OsGroup) Gid() int {
	return g.gid
}

// Name returns the group name.
func (g *OsGroup) Name() string {
	return g.name
}

// OsUser

// Gid returns the primary group ID of the user.
func (u *OsUser) Gid() int {
	return u.gid
}

// IsRoot returns true if the user has root privileges.
func (u *OsUser) IsRoot() bool {
	return u.uid == 0 || u.gid == 0
}

// Name returns the user name.
func (u *OsUser) Name() string {
	return u.name
}

// Uid returns the user ID.
func (u *OsUser) Uid() int {
	return u.uid
}
