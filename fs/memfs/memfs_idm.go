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

package memfs

import (
	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
	"github.com/avfs/avfs/idm/dummyidm"
)

// CurrentUser returns the current user of the file system.
func (vfs *MemFs) CurrentUser() avfs.UserReader {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return dummyidm.NotImplementedUser
	}

	return vfs.user
}

// GroupAdd adds a new group.
func (vfs *MemFs) GroupAdd(name string) (avfs.GroupReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	return vfs.fsAttrs.idm.GroupAdd(name)
}

// GroupDel deletes an existing group.
func (vfs *MemFs) GroupDel(name string) error {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsRoot() {
		return avfs.ErrPermDenied
	}

	return vfs.fsAttrs.idm.GroupDel(name)
}

// LookupGroup looks up a group by name. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func (vfs *MemFs) LookupGroup(name string) (avfs.GroupReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	return vfs.fsAttrs.idm.LookupGroup(name)
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (vfs *MemFs) LookupGroupId(gid int) (avfs.GroupReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	return vfs.fsAttrs.idm.LookupGroupId(gid)
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (vfs *MemFs) LookupUser(name string) (avfs.UserReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	return vfs.fsAttrs.idm.LookupUser(name)
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (vfs *MemFs) LookupUserId(uid int) (avfs.UserReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	return vfs.fsAttrs.idm.LookupUserId(uid)
}

// User sets the current user of the file system.
// If the current user has not root privileges avfs.errPermDenied is returned.
func (vfs *MemFs) User(name string) (avfs.UserReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	if vfs.user.Name() == name {
		return vfs.user, nil
	}

	user, err := vfs.fsAttrs.idm.LookupUser(name)
	if err != nil {
		return nil, err
	}

	vfs.user = user
	vfs.curDir = fsutil.Join(avfs.HomeDir, user.Name())

	return user, nil
}

// UserAdd adds a new user.
func (vfs *MemFs) UserAdd(name, groupName string) (avfs.UserReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	u, err := vfs.fsAttrs.idm.UserAdd(name, groupName)
	if err != nil {
		return nil, err
	}

	return fsutil.CreateHomeDir(vfs, u)
}

// UserDel deletes an existing group.
func (vfs *MemFs) UserDel(name string) error {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsRoot() {
		return avfs.ErrPermDenied
	}

	err := vfs.fsAttrs.idm.UserDel(name)
	if err != nil {
		return err
	}

	userDir := fsutil.Join(avfs.HomeDir, name)

	err = vfs.RemoveAll(userDir)
	if err != nil {
		return err
	}

	return nil
}
