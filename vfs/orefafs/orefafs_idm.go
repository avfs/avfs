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

package orefafs

import (
	"github.com/avfs/avfs"
)

// CurrentUser returns the current User.
func (vfs *OrefaFS) CurrentUser() avfs.UserReader {
	return vfs.currentUser
}

// GroupAdd adds a new Group.
func (vfs *OrefaFS) GroupAdd(name string) (avfs.GroupReader, error) {
	return nil, avfs.ErrPermDenied
}

// GroupDel deletes an existing Group.
func (vfs *OrefaFS) GroupDel(name string) error {
	return avfs.ErrPermDenied
}

// LookupGroup looks up a Group by name. If the Group cannot be found, the
// returned error is of type UnknownGroupError.
func (vfs *OrefaFS) LookupGroup(name string) (avfs.GroupReader, error) {
	return nil, avfs.ErrPermDenied
}

// LookupGroupId looks up a Group by groupid. If the Group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (vfs *OrefaFS) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return nil, avfs.ErrPermDenied
}

// LookupUser looks up a User by username. If the User cannot be found, the
// returned error is of type UnknownUserError.
func (vfs *OrefaFS) LookupUser(name string) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// LookupId looks up a User by userid. If the User cannot be found, the
// returned error is of type UnknownUserIdError.
func (vfs *OrefaFS) LookupUserId(uid int) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// User sets the current User.
func (vfs *OrefaFS) User(name string) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// UserAdd adds a new User.
func (vfs *OrefaFS) UserAdd(name, groupName string) (avfs.UserReader, error) {
	return nil, avfs.ErrPermDenied
}

// UserDel deletes an existing Group.
func (vfs *OrefaFS) UserDel(name string) error {
	return avfs.ErrPermDenied
}
