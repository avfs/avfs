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

package osfs

import (
	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/dummyidm"
	"github.com/avfs/avfs/vfsutils"
)

// CurrentUser returns the current user.
func (vfs *OsFS) CurrentUser() avfs.UserReader {
	uc, ok := vfs.idm.(avfs.UserConnecter)
	if ok {
		return uc.CurrentUser()
	}

	return dummyidm.NotImplementedUser
}

// GroupAdd adds a new group.
func (vfs *OsFS) GroupAdd(name string) (avfs.GroupReader, error) {
	if !vfs.CurrentUser().IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	return vfs.idm.GroupAdd(name)
}

// GroupDel deletes an existing group.
func (vfs *OsFS) GroupDel(name string) error {
	if !vfs.CurrentUser().IsRoot() {
		return avfs.ErrPermDenied
	}

	return vfs.idm.GroupDel(name)
}

// LookupGroup looks up a group by name. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func (vfs *OsFS) LookupGroup(name string) (avfs.GroupReader, error) {
	return vfs.idm.LookupGroup(name)
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (vfs *OsFS) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return vfs.idm.LookupGroupId(gid)
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (vfs *OsFS) LookupUser(name string) (avfs.UserReader, error) {
	return vfs.idm.LookupUser(name)
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (vfs *OsFS) LookupUserId(uid int) (avfs.UserReader, error) {
	return vfs.idm.LookupUserId(uid)
}

// User sets the current user of the file system to uid.
// If the current user has not root privileges avfs.errPermDenied is returned.
func (vfs *OsFS) User(name string) (avfs.UserReader, error) {
	uc, ok := vfs.idm.(avfs.UserConnecter)
	if ok {
		return uc.User(name)
	}

	return nil, avfs.ErrPermDenied
}

// UserAdd adds a new user.
func (vfs *OsFS) UserAdd(name, groupName string) (avfs.UserReader, error) {
	if !vfs.CurrentUser().IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	u, err := vfs.idm.UserAdd(name, groupName)
	if err != nil {
		return nil, err
	}

	return vfsutils.CreateHomeDir(vfs, u)
}

// UserDel deletes an existing group.
func (vfs *OsFS) UserDel(name string) error {
	if !vfs.CurrentUser().IsRoot() {
		return avfs.ErrPermDenied
	}

	err := vfs.idm.UserDel(name)
	if err != nil {
		return err
	}

	userDir := vfsutils.Join(avfs.HomeDir, name)

	err = vfs.RemoveAll(userDir)
	if err != nil {
		return err
	}

	return nil
}
