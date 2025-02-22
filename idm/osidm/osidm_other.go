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

//go:build !linux

package osidm

import (
	"math"

	"github.com/avfs/avfs"
)

// AddGroup creates a new group with the specified name.
// If the group already exists, the returned error is of type avfs.AlreadyExistsGroupError.
func (idm *OsIdm) AddGroup(groupName string) (avfs.GroupReader, error) {
	return nil, avfs.ErrPermDenied
}

// AddUser creates a new user with the specified userName and the specified primary group groupName.
// If the user already exists, the returned error is of type avfs.AlreadyExistsUserError.
func (idm *OsIdm) AddUser(userName, groupName string) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// AddUserToGroup adds the user to the group.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
func (idm *OsIdm) AddUserToGroup(userName, groupName string) error {
	return avfs.ErrPermDenied
}

// DelGroup deletes an existing group with the specified name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *OsIdm) DelGroup(groupName string) error {
	return avfs.ErrPermDenied
}

// DelUser deletes an existing user with the specified name.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *OsIdm) DelUser(userName string) error {
	return avfs.ErrPermDenied
}

// DelUserFromGroup removes the user from the group.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError
// or avfs.UnknownGroupError respectively.
func (idm *OsIdm) DelUserFromGroup(userName, groupName string) error {
	return avfs.ErrPermDenied
}

// LookupGroup looks up a group by name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *OsIdm) LookupGroup(groupName string) (avfs.GroupReader, error) {
	return nil, avfs.ErrPermDenied
}

// LookupGroupId looks up a group by groupid.
// If the group is not found, the returned error is of type avfs.UnknownGroupIdError.
func (idm *OsIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return nil, avfs.ErrPermDenied
}

// LookupUser looks up a user by username.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *OsIdm) LookupUser(userName string) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// LookupUserId looks up a user by userid.
// If the user is not found, the returned error is of type avfs.UnknownUserIdError.
func (idm *OsIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// SetUserPrimaryGroup sets the primary group of a user to the specified group name.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
// If the operation fails, the returned error is of type avfs.UnknownError.
func (idm *OsIdm) SetUserPrimaryGroup(userName, groupName string) error {
	return avfs.ErrPermDenied
}

// Groups returns a slice of strings representing the group names that the user belongs to.
// If an error occurs while fetching the group names, it returns nil.
func (u *OsUser) Groups() []string {
	return nil
}

// GroupsId returns a slice group IDs that the user belongs to.
// If an error occurs while fetching the group IDs, it returns nil.
func (u *OsUser) GroupsId() []int {
	return nil
}

// IsInGroupId returns true if the user is in the specified group ID.
func (u *OsUser) IsInGroupId(gid int) bool {
	return false
}

// PrimaryGroup returns the primary group name of the user.
func (u *OsUser) PrimaryGroup() string {
	return ""
}

// PrimaryGroupId returns the primary group ID of the OsUser.
// If an error occurs, it returns the maximum integer value.
func (u *OsUser) PrimaryGroupId() int {
	return math.MaxInt
}

// SetUser sets the current user.
// If the user can't be changed an error is returned.
func SetUser(user avfs.UserReader) error {
	return avfs.ErrPermDenied
}

// SetUserByName sets the current user by name.
// If the user is not found, the returned error is of type UnknownUserError.
func SetUserByName(userName string) error {
	return avfs.ErrPermDenied
}

// User returns the current user of the OS.
func User() avfs.UserReader {
	return avfs.NotImplementedIdm.AdminUser()
}

// IsUserAdmin returns true if the current user has admin privileges.
func isUserAdmin() bool {
	return false
}
