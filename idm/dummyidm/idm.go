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

package dummyidm

import (
	"github.com/avfs/avfs"
)

// New create a new identity manager.
func New() *DummyIdm {
	return &DummyIdm{}
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *DummyIdm) Type() string {
	return "DummyIdm"
}

// CurrentUser returns the current user of the identity manager.
func (idm *DummyIdm) CurrentUser() avfs.UserReader {
	return NotImplementedUser
}

// GroupAdd adds a new group.
func (idm *DummyIdm) GroupAdd(name string) (avfs.GroupReader, error) {
	_ = name
	return nil, avfs.ErrNotImplemented
}

// GroupDel deletes an existing group.
func (idm *DummyIdm) GroupDel(name string) error {
	_ = name
	return avfs.ErrNotImplemented
}

// LookupGroup looks up a group by name.
// If the group cannot be found, the returned error is of type UnknownGroupError.
func (idm *DummyIdm) LookupGroup(name string) (avfs.GroupReader, error) {
	_ = name
	return nil, avfs.ErrNotImplemented
}

// LookupGroupId looks up a group by groupid.
// If the group cannot be found, the returned error is of type UnknownGroupIdError.
func (idm *DummyIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	_ = gid
	return nil, avfs.ErrNotImplemented
}

// LookupUser looks up a user by username.
// If the user cannot be found, the returned error is of type UnknownUserError.
func (idm *DummyIdm) LookupUser(name string) (avfs.UserReader, error) {
	_ = name
	return nil, avfs.ErrNotImplemented
}

// LookupUserId looks up a user by userid.
// If the user cannot be found, the returned error is of type UnknownUserIdError.
func (idm *DummyIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	_ = uid
	return nil, avfs.ErrNotImplemented
}

// User sets the current user of the identity manager.
// If the current user has not root privileges avfs.errPermDenied is returned.
func (idm *DummyIdm) User(name string) (avfs.UserReader, error) {
	_ = name
	return nil, avfs.ErrNotImplemented
}

// UserAdd adds a new user.
func (idm *DummyIdm) UserAdd(name, groupName string) (avfs.UserReader, error) {
	_, _ = name, groupName
	return nil, avfs.ErrNotImplemented
}

// UserDel deletes an existing group.
func (idm *DummyIdm) UserDel(name string) error {
	_ = name
	return avfs.ErrNotImplemented
}

// Group

// Gid returns the Group ID.
func (g *Group) Gid() int {
	return g.gid
}

// Name returns the Group name.
func (g *Group) Name() string {
	return g.name
}

// User

// Gid returns the primary Group ID of the User.
func (u *User) Gid() int {
	return u.gid
}

// IsRoot returns true if the User has root privileges.
func (u *User) IsRoot() bool {
	return u.uid == 0 || u.gid == 0
}

// Name returns the User name.
func (u *User) Name() string {
	return u.name
}

// Uid returns the User ID.
func (u *User) Uid() int {
	return u.uid
}
