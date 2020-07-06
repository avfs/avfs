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
)

// CurrentUser returns the current user of the file system.
func (fs *MemFs) CurrentUser() avfs.UserReader {
	return fs.user
}

// GroupAdd adds a new group.
func (fs *MemFs) GroupAdd(name string) (avfs.GroupReader, error) {
	if !fs.user.IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	return fs.fsAttrs.idm.GroupAdd(name)
}

// GroupDel deletes an existing group.
func (fs *MemFs) GroupDel(name string) error {
	if !fs.user.IsRoot() {
		return avfs.ErrPermDenied
	}

	return fs.fsAttrs.idm.GroupDel(name)
}

// LookupGroup looks up a group by name. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func (fs *MemFs) LookupGroup(name string) (avfs.GroupReader, error) {
	return fs.fsAttrs.idm.LookupGroup(name)
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (fs *MemFs) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return fs.fsAttrs.idm.LookupGroupId(gid)
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (fs *MemFs) LookupUser(name string) (avfs.UserReader, error) {
	return fs.fsAttrs.idm.LookupUser(name)
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (fs *MemFs) LookupUserId(uid int) (avfs.UserReader, error) {
	return fs.fsAttrs.idm.LookupUserId(uid)
}

// User sets the current user of the file system.
// If the current user has not root privileges avfs.errPermDenied is returned.
func (fs *MemFs) User(name string) (avfs.UserReader, error) {
	if fs.user.Name() == name {
		return fs.user, nil
	}

	user, err := fs.fsAttrs.idm.LookupUser(name)
	if err != nil {
		return nil, err
	}

	fs.user = user
	fs.curDir = fsutil.Join(avfs.HomeDir, user.Name())

	return user, nil
}

// UserAdd adds a new user.
func (fs *MemFs) UserAdd(name, groupName string) (avfs.UserReader, error) {
	if !fs.user.IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	u, err := fs.fsAttrs.idm.UserAdd(name, groupName)
	if err != nil {
		return nil, err
	}

	return fsutil.CreateHomeDir(fs, u)
}

// UserDel deletes an existing group.
func (fs *MemFs) UserDel(name string) error {
	if !fs.user.IsRoot() {
		return avfs.ErrPermDenied
	}

	err := fs.fsAttrs.idm.UserDel(name)
	if err != nil {
		return err
	}

	userDir := fsutil.Join(avfs.HomeDir, name)

	err = fs.RemoveAll(userDir)
	if err != nil {
		return err
	}

	return nil
}
