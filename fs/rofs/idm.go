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

package rofs

import "github.com/avfs/avfs"

// CurrentUser returns the current user.
func (fs *RoFs) CurrentUser() avfs.UserReader {
	return fs.baseFs.CurrentUser()
}

// GroupAdd adds a new group.
func (fs *RoFs) GroupAdd(name string) (avfs.GroupReader, error) {
	_ = name
	return nil, avfs.ErrPermDenied
}

// GroupDel deletes an existing group.
func (fs *RoFs) GroupDel(name string) error {
	_ = name
	return avfs.ErrPermDenied
}

// LookupGroup looks up a group by name. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func (fs *RoFs) LookupGroup(name string) (avfs.GroupReader, error) {
	return fs.baseFs.LookupGroup(name)
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (fs *RoFs) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return fs.baseFs.LookupGroupId(gid)
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (fs *RoFs) LookupUser(name string) (avfs.UserReader, error) {
	return fs.baseFs.LookupUser(name)
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (fs *RoFs) LookupUserId(uid int) (avfs.UserReader, error) {
	return fs.baseFs.LookupUserId(uid)
}

// User sets the current User.
func (fs *RoFs) User(name string) (avfs.UserReader, error) {
	_ = name
	return nil, avfs.ErrPermDenied
}

// UserAdd adds a new user.
func (fs *RoFs) UserAdd(name, groupName string) (avfs.UserReader, error) {
	_, _ = name, groupName
	return nil, avfs.ErrPermDenied
}

// UserDel deletes an existing user.
func (fs *RoFs) UserDel(name string) error {
	_ = name
	return avfs.ErrPermDenied
}
