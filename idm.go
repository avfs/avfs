//
//  Copyright 2024 The AVFS authors
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

// IdentityMgr interface manages identities (users and groups).
type IdentityMgr interface {
	Featurer
	OSTyper
	Typer

	// AdminGroup returns the administrator (root) group.
	AdminGroup() GroupReader

	// AdminUser returns the administrator (root) user.
	AdminUser() UserReader

	// AddGroup creates a new group with the specified name.
	// If the group already exists, the returned error is of type avfs.AlreadyExistsGroupError.
	AddGroup(groupName string) (GroupReader, error)

	// AddUser creates a new user with the specified userName and the specified primary group groupName.
	// If the user already exists, the returned error is of type avfs.AlreadyExistsUserError.
	AddUser(userName, groupName string) (UserReader, error)

	// AddUserToGroup adds the user to the group.
	// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
	AddUserToGroup(userName, groupName string) error

	// DelGroup deletes an existing group with the specified name.
	// If the group is not found, the returned error is of type avfs.UnknownGroupError.
	DelGroup(groupName string) error

	// DelUser deletes an existing user with the specified name.
	// If the user is not found, the returned error is of type avfs.UnknownUserError.
	DelUser(userName string) error

	// DelUserFromGroup removes the user from the group.
	// If the user or group is not found, the returned error is of type avfs.UnknownUserError
	// or avfs.UnknownGroupError respectively.
	DelUserFromGroup(userName, groupName string) error

	// LookupGroup looks up a group by name.
	// If the group is not found, the returned error is of type avfs.UnknownGroupError.
	LookupGroup(groupName string) (GroupReader, error)

	// LookupGroupId looks up a group by groupid.
	// If the group is not found, the returned error is of type avfs.UnknownGroupIdError.
	LookupGroupId(gid int) (GroupReader, error)

	// LookupUser looks up a user by username.
	// If the user is not found, the returned error is of type avfs.UnknownUserError.
	LookupUser(userName string) (UserReader, error)

	// LookupUserId looks up a user by userid.
	// If the user is not found, the returned error is of type avfs.UnknownUserIdError.
	LookupUserId(uid int) (UserReader, error)

	// SetUserPrimaryGroup sets the primary group of a user to the specified group name.
	// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
	// If the operation fails, the returned error is of type avfs.UnknownError.
	SetUserPrimaryGroup(userName, groupName string) error
}

// UserReader reads user information.
type UserReader interface {
	GroupIdentifier
	UserIdentifier
	Namer

	// Groups returns a slice of strings representing the group names that the user belongs to.
	// If an error occurs while fetching the group names, it returns nil.
	Groups() []string

	// GroupsId returns a slice group IDs that the user belongs to.
	// If an error occurs while fetching the group IDs, it returns nil.
	GroupsId() []int

	// IsInGroupId returns true if the user is in the specified group ID.
	IsInGroupId(gid int) bool

	// IsAdmin returns true if the user has administrator (root) privileges.
	IsAdmin() bool

	// PrimaryGroup returns the primary group name of the user.
	PrimaryGroup() string

	// PrimaryGroupId returns the primary group ID of the OsUser.
	// If an error occurs, it returns the maximum integer value.
	PrimaryGroupId() int
}

// GroupIdentifier is the interface that wraps the Gid method.
type GroupIdentifier interface {
	// Gid returns the primary group id.
	Gid() int
}

// GroupReader interface reads group information.
type GroupReader interface {
	GroupIdentifier
	Namer
}

// UserIdentifier is the interface that wraps the Uid method.
type UserIdentifier interface {
	// Uid returns the user id.
	Uid() int
}

// IdmMgr is the interface that wraps Identity manager setting methods for file systems.
type IdmMgr interface {
	// Idm returns the identity manager of the file system.
	Idm() IdentityMgr

	// SetIdm set the current identity manager.
	// If the identity manager provider is nil, the idm dummyidm.NotImplementedIdm is set.
	SetIdm(idm IdentityMgr) error
}

// IdmFn provides identity manager functions to a file system.
type IdmFn struct {
	idm IdentityMgr // idm is the identity manager of the file system.
}

// Idm returns the identity manager of the file system.
func (idf *IdmFn) Idm() IdentityMgr {
	return idf.idm
}

// SetIdm set the current identity manager.
// If the identity manager provider is nil, the idm NotImplementedIdm is set.
func (idf *IdmFn) SetIdm(idm IdentityMgr) error {
	if idm == nil {
		idm = NotImplementedIdm
	}

	idf.idm = idm

	return nil
}

// AdminGroupName returns the name of the administrator group of the file system.
func AdminGroupName(osType OSType) string {
	switch osType {
	case OsWindows:
		return "Administrators"
	default:
		return "root"
	}
}

// AdminUserName returns the name of the administrator of the file system.
func AdminUserName(osType OSType) string {
	switch osType {
	case OsWindows:
		return "ContainerAdministrator"
	default:
		return "root"
	}
}
