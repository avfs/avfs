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

package dummyidm

import (
	"math"

	"github.com/avfs/avfs"
)

// New create a new identity manager.
func New() *DummyIdm {
	return &DummyIdm{
		adminGroup: &DummyGroup{name: avfs.DefaultName, gid: math.MaxInt},
		adminUser:  &DummyUser{name: avfs.DefaultName, uid: math.MaxInt, gid: math.MaxInt},
	}
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *DummyIdm) Type() string {
	return "DummyIdm"
}

// Features returns the set of features provided by the file system or identity manager.
func (idm *DummyIdm) Features() avfs.Features {
	return 0
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (idm *DummyIdm) HasFeature(feature avfs.Features) bool {
	return false
}

// AdminGroup returns the administrators (root) group.
func (idm *DummyIdm) AdminGroup() avfs.GroupReader {
	return idm.adminGroup
}

// AdminUser returns the administrator (root) user.
func (idm *DummyIdm) AdminUser() avfs.UserReader {
	return idm.adminUser
}

// GroupAdd adds a new group.
func (idm *DummyIdm) GroupAdd(name string) (avfs.GroupReader, error) {
	return nil, avfs.ErrPermDenied
}

// GroupDel deletes an existing group.
func (idm *DummyIdm) GroupDel(name string) error {
	return avfs.ErrPermDenied
}

// LookupGroup looks up a group by name.
// If the group cannot be found, the returned error is of type UnknownGroupError.
func (idm *DummyIdm) LookupGroup(name string) (avfs.GroupReader, error) {
	return nil, avfs.ErrPermDenied
}

// LookupGroupId looks up a group by groupid.
// If the group cannot be found, the returned error is of type UnknownGroupIdError.
func (idm *DummyIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return nil, avfs.ErrPermDenied
}

// LookupUser looks up a user by username.
// If the user cannot be found, the returned error is of type UnknownUserError.
func (idm *DummyIdm) LookupUser(name string) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// LookupUserId looks up a user by userid.
// If the user cannot be found, the returned error is of type UnknownUserIdError.
func (idm *DummyIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// OSType returns the operating system type of the identity manager.
func (idm *DummyIdm) OSType() avfs.OSType {
	return avfs.CurrentOSType()
}

// UserAdd adds a new user.
func (idm *DummyIdm) UserAdd(name, groupName string) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// UserDel deletes an existing group.
func (idm *DummyIdm) UserDel(name string) error {
	return avfs.ErrPermDenied
}

// DummyGroup

// Gid returns the Group ID.
func (g *DummyGroup) Gid() int {
	return g.gid
}

// Name returns the Group name.
func (g *DummyGroup) Name() string {
	return g.name
}

// DummyUser

// Gid returns the primary Group ID of the User.
func (u *DummyUser) Gid() int {
	return u.gid
}

// IsAdmin returns true if the user has administrator (root) privileges.
func (u *DummyUser) IsAdmin() bool {
	return u.uid == 0 || u.gid == 0
}

// Name returns the username.
func (u *DummyUser) Name() string {
	return u.name
}

// Uid returns the User ID.
func (u *DummyUser) Uid() int {
	return u.uid
}
