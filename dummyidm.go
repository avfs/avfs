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

const maxInt = int(^uint(0) >> 1)

var (
	// NotImplementedGroup represents a not implemented group.
	NotImplementedGroup = &DummyGroup{ //nolint:gochecknoglobals // Used as default Idm for other file systems.
		name: NotImplemented,
		gid:  maxInt,
	}

	// NotImplementedIdm is the default identity manager for all file systems.
	NotImplementedIdm = &DummyIdm{} //nolint:gochecknoglobals // Used as default Idm for other file systems.

	// NotImplementedUser represents a not implemented user.
	NotImplementedUser = &DummyUser{ //nolint:gochecknoglobals // Used as not implemented user for other file systems.
		name: NotImplemented,
		uid:  maxInt,
		gid:  maxInt,
	}

	// AdminUser represents an administrator user.
	AdminUser = &DummyUser{ //nolint:gochecknoglobals // Used as Admin user for other file systems.
		name: OsUtils.AdminUserName(),
		uid:  0,
		gid:  0,
	}
)

// DummyIdm represent a non implemented identity manager using the avfs.IdentityMgr interface.
type DummyIdm struct{}

// DummyUser is the implementation of avfs.UserReader.
type DummyUser struct {
	name string
	uid  int
	gid  int
}

// DummyGroup is the implementation of avfs.GroupReader.
type DummyGroup struct {
	name string
	gid  int
}

// NewDummyIdm create a new identity manager.
func NewDummyIdm() *DummyIdm {
	return &DummyIdm{}
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *DummyIdm) Type() string {
	return "DummyIdm"
}

// Features returns the set of features provided by the file system or identity manager.
func (idm *DummyIdm) Features() Feature {
	return 0
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (idm *DummyIdm) HasFeature(feature Feature) bool {
	return false
}

// AdminGroup returns the administrators (root) group.
func (idm *DummyIdm) AdminGroup() GroupReader {
	return NotImplementedGroup
}

// AdminUser returns the administrator (root) user.
func (idm *DummyIdm) AdminUser() UserReader {
	return NotImplementedUser
}

// CurrentUser returns the current user.
func (idm *DummyIdm) CurrentUser() UserReader {
	return NotImplementedUser
}

// GroupAdd adds a new group.
func (idm *DummyIdm) GroupAdd(name string) (GroupReader, error) {
	return nil, ErrPermDenied
}

// GroupDel deletes an existing group.
func (idm *DummyIdm) GroupDel(name string) error {
	return ErrPermDenied
}

// LookupGroup looks up a group by name.
// If the group cannot be found, the returned error is of type UnknownGroupError.
func (idm *DummyIdm) LookupGroup(name string) (GroupReader, error) {
	return nil, ErrPermDenied
}

// LookupGroupId looks up a group by groupid.
// If the group cannot be found, the returned error is of type UnknownGroupIdError.
func (idm *DummyIdm) LookupGroupId(gid int) (GroupReader, error) {
	return nil, ErrPermDenied
}

// LookupUser looks up a user by username.
// If the user cannot be found, the returned error is of type UnknownUserError.
func (idm *DummyIdm) LookupUser(name string) (UserReader, error) {
	return nil, ErrPermDenied
}

// LookupUserId looks up a user by userid.
// If the user cannot be found, the returned error is of type UnknownUserIdError.
func (idm *DummyIdm) LookupUserId(uid int) (UserReader, error) {
	return nil, ErrPermDenied
}

// User sets the current user of the file system to uid.
// If the current user has not root privileges errPermDenied is returned.
func (idm *DummyIdm) User(name string) (UserReader, error) {
	return nil, ErrPermDenied
}

// UserAdd adds a new user.
func (idm *DummyIdm) UserAdd(name, groupName string) (UserReader, error) {
	return nil, ErrPermDenied
}

// UserDel deletes an existing group.
func (idm *DummyIdm) UserDel(name string) error {
	return ErrPermDenied
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

// IsRoot returns true if the User has root privileges.
func (u *DummyUser) IsRoot() bool {
	return u.uid == 0 || u.gid == 0
}

// Name returns the User name.
func (u *DummyUser) Name() string {
	return u.name
}

// Uid returns the User ID.
func (u *DummyUser) Uid() int {
	return u.uid
}
