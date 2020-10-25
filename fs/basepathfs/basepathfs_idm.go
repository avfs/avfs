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

package basepathfs

import (
	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
)

// CurrentUser returns the current user of the file system.
func (vfs *BasePathFs) CurrentUser() avfs.UserReader {
	return vfs.baseFs.CurrentUser()
}

// GroupAdd adds a new group.
func (vfs *BasePathFs) GroupAdd(name string) (avfs.GroupReader, error) {
	return vfs.baseFs.GroupAdd(name)
}

// GroupDel deletes an existing group.
func (vfs *BasePathFs) GroupDel(name string) error {
	return vfs.baseFs.GroupDel(name)
}

// LookupGroup looks up a group by name. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func (vfs *BasePathFs) LookupGroup(name string) (avfs.GroupReader, error) {
	return vfs.baseFs.LookupGroup(name)
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (vfs *BasePathFs) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return vfs.baseFs.LookupGroupId(gid)
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (vfs *BasePathFs) LookupUser(name string) (avfs.UserReader, error) {
	return vfs.baseFs.LookupUser(name)
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (vfs *BasePathFs) LookupUserId(uid int) (avfs.UserReader, error) {
	return vfs.baseFs.LookupUserId(uid)
}

// User sets the current user of the file system.
// If the current user has not root privileges avfs.errPermDenied is returned.
func (vfs *BasePathFs) User(name string) (avfs.UserReader, error) {
	return vfs.baseFs.User(name)
}

// UserAdd adds a new user.
func (vfs *BasePathFs) UserAdd(name, groupName string) (avfs.UserReader, error) {
	u, err := vfs.baseFs.UserAdd(name, groupName)
	if err != nil {
		return nil, err
	}

	return fsutil.CreateHomeDir(vfs, u)
}

// UserDel deletes an existing group.
func (vfs *BasePathFs) UserDel(name string) error {
	return vfs.baseFs.UserDel(name)
}
