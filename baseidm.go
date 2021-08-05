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

// CurrentUser returns the current user.
func (idm *BaseIdm) CurrentUser() UserReader {
	return NotImplementedUser
}

// GroupAdd adds a new group.
func (idm *BaseIdm) GroupAdd(name string) (GroupReader, error) {
	return nil, ErrPermDenied
}

// GroupDel deletes an existing group.
func (idm *BaseIdm) GroupDel(name string) error {
	return ErrPermDenied
}

// LookupGroup looks up a group by name.
// If the group cannot be found, the returned error is of type UnknownGroupError.
func (idm *BaseIdm) LookupGroup(name string) (GroupReader, error) {
	return nil, ErrPermDenied
}

// LookupGroupId looks up a group by groupid.
// If the group cannot be found, the returned error is of type UnknownGroupIdError.
func (idm *BaseIdm) LookupGroupId(gid int) (GroupReader, error) {
	return nil, ErrPermDenied
}

// LookupUser looks up a user by username.
// If the user cannot be found, the returned error is of type UnknownUserError.
func (idm *BaseIdm) LookupUser(name string) (UserReader, error) {
	return nil, ErrPermDenied
}

// LookupUserId looks up a user by userid.
// If the user cannot be found, the returned error is of type UnknownUserIdError.
func (idm *BaseIdm) LookupUserId(uid int) (UserReader, error) {
	return nil, ErrPermDenied
}

// User sets the current user of the file system to uid.
// If the current user has not root privileges errPermDenied is returned.
func (idm *BaseIdm) User(name string) (UserReader, error) {
	return nil, ErrPermDenied
}

// UserAdd adds a new user.
func (idm *BaseIdm) UserAdd(name, groupName string) (UserReader, error) {
	return nil, ErrPermDenied
}

// UserDel deletes an existing group.
func (idm *BaseIdm) UserDel(name string) error {
	return ErrPermDenied
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
