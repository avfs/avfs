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

import "math"

// NotImplementedIdm is the default identity manager for all file systems.
var NotImplementedIdm = NewDummyIdm() //nolint:gochecknoglobals // Used as default Idm for other file systems.

// DummyIdm represent a non implemented identity manager using the avfs.IdentityMgr interface.
type DummyIdm struct {
	adminGroup GroupReader
	adminUser  UserReader
}

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
	return &DummyIdm{
		adminGroup: &DummyGroup{name: DefaultName, gid: math.MaxInt},
		adminUser:  &DummyUser{name: DefaultName, uid: math.MaxInt, gid: math.MaxInt},
	}
}

// NewGroup creates and returns a pointer to a DummyGroup with the specified group name and group ID.
func NewGroup(groupName string, gid int) *DummyGroup {
	return &DummyGroup{name: groupName, gid: gid}
}

// NewUser creates a new instance of DummyUser with the specified username, user ID (uid), and group ID (gid).
func NewUser(userName string, uid, gid int) *DummyUser {
	return &DummyUser{name: userName, uid: uid, gid: gid}
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *DummyIdm) Type() string {
	return "DummyIdm"
}

// Features returns the set of features provided by the file system or identity manager.
func (idm *DummyIdm) Features() Features {
	return 0
}

// HasFeature returns true if the file system or identity manager provides a given feature.
func (idm *DummyIdm) HasFeature(feature Features) bool {
	return false
}

// AdminGroup returns the administrators (root) group.
func (idm *DummyIdm) AdminGroup() GroupReader {
	return idm.adminGroup
}

// AdminUser returns the administrator (root) user.
func (idm *DummyIdm) AdminUser() UserReader {
	return idm.adminUser
}

// AddGroup creates a new group with the specified name.
// If the group already exists, the returned error is of type avfs.AlreadyExistsGroupError.
func (idm *DummyIdm) AddGroup(groupName string) (GroupReader, error) {
	return nil, ErrPermDenied
}

// AddUser creates a new user with the specified userName and the specified primary group groupName.
// If the user already exists, the returned error is of type avfs.AlreadyExistsUserError.
func (idm *DummyIdm) AddUser(userName, groupName string) (UserReader, error) {
	return nil, ErrPermDenied
}

// AddUserToGroup adds the user to the group.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
func (idm *DummyIdm) AddUserToGroup(userName, groupName string) error {
	return ErrPermDenied
}

// DelGroup deletes an existing group with the specified name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *DummyIdm) DelGroup(groupName string) error {
	return ErrPermDenied
}

// DelUser deletes an existing user with the specified name.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *DummyIdm) DelUser(userName string) error {
	return ErrPermDenied
}

// DelUserFromGroup removes the user from the group.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError
// or avfs.UnknownGroupError respectively.
func (idm *DummyIdm) DelUserFromGroup(userName, groupName string) error {
	return ErrPermDenied
}

// LookupGroup looks up a group by name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *DummyIdm) LookupGroup(groupName string) (GroupReader, error) {
	return nil, ErrPermDenied
}

// LookupGroupId looks up a group by groupid.
// If the group is not found, the returned error is of type avfs.UnknownGroupIdError.
func (idm *DummyIdm) LookupGroupId(gid int) (GroupReader, error) {
	return nil, ErrPermDenied
}

// LookupUser looks up a user by username.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *DummyIdm) LookupUser(userName string) (UserReader, error) {
	return nil, ErrPermDenied
}

// LookupUserId looks up a user by userid.
// If the user cannot be found, the returned error is of type avfs.UnknownUserIdError.
func (idm *DummyIdm) LookupUserId(uid int) (UserReader, error) {
	return nil, ErrPermDenied
}

// SetUserPrimaryGroup sets the primary group of a user to the specified group name.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
// If the operation fails, the returned error is of type avfs.UnknownError.
func (idm *DummyIdm) SetUserPrimaryGroup(userName, groupName string) error {
	return ErrPermDenied
}

// OSType returns the operating system type of the identity manager.
func (idm *DummyIdm) OSType() OSType {
	return CurrentOSType()
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

// Groups returns a slice of strings representing the group names that the user belongs to.
// If an error occurs while fetching the group names, it returns nil.
func (u *DummyUser) Groups() []string {
	return nil
}

// GroupsId returns a slice group IDs that the user belongs to.
// If an error occurs while fetching the group IDs, it returns nil.
func (u *DummyUser) GroupsId() []int {
	return nil
}

// IsInGroupId returns true if the user is in the specified group ID.
func (u *DummyUser) IsInGroupId(gid int) bool {
	return false
}

// IsAdmin returns true if the user has administrator (root) privileges.
func (u *DummyUser) IsAdmin() bool {
	return u.uid == 0 || u.gid == 0
}

// Name returns the username.
func (u *DummyUser) Name() string {
	return u.name
}

// PrimaryGroup returns the primary group name of the user.
func (u *DummyUser) PrimaryGroup() string {
	return ""
}

// PrimaryGroupId returns the primary group ID of the OsUser.
// If an error occurs, it returns the maximum integer value.
func (u *DummyUser) PrimaryGroupId() int {
	return math.MaxInt
}

// Uid returns the User ID.
func (u *DummyUser) Uid() int {
	return u.uid
}
