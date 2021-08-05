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

// CurrentUser returns the current User.
func (vfs *BaseFS) CurrentUser() UserReader {
	return NotImplementedUser
}

// GroupAdd adds a new Group.
func (vfs *BaseFS) GroupAdd(name string) (GroupReader, error) {
	return nil, ErrPermDenied
}

// GroupDel deletes an existing Group.
func (vfs *BaseFS) GroupDel(name string) error {
	return ErrPermDenied
}

// LookupGroup looks up a Group by name. If the Group cannot be found, the
// returned error is of type UnknownGroupError.
func (vfs *BaseFS) LookupGroup(name string) (GroupReader, error) {
	return nil, ErrPermDenied
}

// LookupGroupId looks up a Group by groupid. If the Group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (vfs *BaseFS) LookupGroupId(gid int) (GroupReader, error) {
	return nil, ErrPermDenied
}

// LookupUser looks up a User by username. If the User cannot be found, the
// returned error is of type UnknownUserError.
func (vfs *BaseFS) LookupUser(name string) (UserReader, error) {
	return nil, ErrPermDenied
}

// LookupUserId looks up a User by userid. If the User cannot be found, the
// returned error is of type UnknownUserIdError.
func (vfs *BaseFS) LookupUserId(uid int) (UserReader, error) {
	return nil, ErrPermDenied
}

// User sets the current User.
func (vfs *BaseFS) User(name string) (UserReader, error) {
	return nil, ErrPermDenied
}

// UserAdd adds a new User.
func (vfs *BaseFS) UserAdd(name, groupName string) (UserReader, error) {
	return nil, ErrPermDenied
}

// UserDel deletes an existing Group.
func (vfs *BaseFS) UserDel(name string) error {
	return ErrPermDenied
}
