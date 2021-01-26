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
// Copied from standard library os/user/*_unix.go with few modifications.
package osidm

import "github.com/avfs/avfs"

// CurrentUser returns the current user.
func (idm *OsIdm) CurrentUser() avfs.UserReader {
	return currentUser()
}

// group

// Gid returns the group ID.
func (g *Group) Gid() int {
	return g.gid
}

// Name returns the group name.
func (g *Group) Name() string {
	return g.name
}

// user

// Gid returns the primary group ID of the user.
func (u *User) Gid() int {
	return u.gid
}

// IsRoot returns true if the user has root privileges.
func (u *User) IsRoot() bool {
	return u.uid == 0 || u.gid == 0
}

// Name returns the user name.
func (u *User) Name() string {
	return u.name
}

// Uid returns the user ID.
func (u *User) Uid() int {
	return u.uid
}
