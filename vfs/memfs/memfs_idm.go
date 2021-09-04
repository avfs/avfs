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

import "github.com/avfs/avfs"

// CurrentUser returns the current user of the file system.
func (vfs *MemFS) CurrentUser() avfs.UserReader {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return avfs.NotImplementedUser
	}

	return vfs.user
}

// GroupAdd adds a new group.
func (vfs *MemFS) GroupAdd(name string) (avfs.GroupReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	return vfs.memAttrs.idm.GroupAdd(name)
}

// GroupDel deletes an existing group.
func (vfs *MemFS) GroupDel(name string) error {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsRoot() {
		return avfs.ErrPermDenied
	}

	return vfs.memAttrs.idm.GroupDel(name)
}

// LookupGroup looks up a group by name. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func (vfs *MemFS) LookupGroup(name string) (avfs.GroupReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	return vfs.memAttrs.idm.LookupGroup(name)
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (vfs *MemFS) LookupGroupId(gid int) (avfs.GroupReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	return vfs.memAttrs.idm.LookupGroupId(gid)
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (vfs *MemFS) LookupUser(name string) (avfs.UserReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	return vfs.memAttrs.idm.LookupUser(name)
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (vfs *MemFS) LookupUserId(uid int) (avfs.UserReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	return vfs.memAttrs.idm.LookupUserId(uid)
}

// User sets the current user of the file system.
// If the current user has not root privileges avfs.errPermDenied is returned.
func (vfs *MemFS) User(name string) (avfs.UserReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return nil, avfs.ErrPermDenied
	}

	if vfs.user.Name() == name {
		return vfs.user, nil
	}

	user, err := vfs.memAttrs.idm.LookupUser(name)
	if err != nil {
		return nil, err
	}

	vfs.user = user
	vfs.curDir = avfs.HomeDirUser(vfs, user.Name())

	return user, nil
}

// UserAdd adds a new user.
func (vfs *MemFS) UserAdd(name, groupName string) (avfs.UserReader, error) {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	u, err := vfs.memAttrs.idm.UserAdd(name, groupName)
	if err != nil {
		return nil, err
	}

	return avfs.CreateHomeDir(vfs, u)
}

// UserDel deletes an existing group.
func (vfs *MemFS) UserDel(name string) error {
	if !vfs.HasFeature(avfs.FeatIdentityMgr) || !vfs.user.IsRoot() {
		return avfs.ErrPermDenied
	}

	err := vfs.memAttrs.idm.UserDel(name)
	if err != nil {
		return err
	}

	homeDir := avfs.HomeDirUser(vfs, name)

	err = vfs.RemoveAll(homeDir)
	if err != nil {
		return err
	}

	return nil
}
