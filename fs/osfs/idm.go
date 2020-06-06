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
	"github.com/avfs/avfs/fsutil"
	"github.com/avfs/avfs/idm/dummyidm"
)

// CurrentUser returns the current user.
func (fs *OsFs) CurrentUser() avfs.UserReader {
	uc, ok := fs.idm.(avfs.UserConnecter)
	if ok {
		return uc.CurrentUser()
	}

	return dummyidm.NotImplementedUser
}

// GroupAdd adds a new group.
func (fs *OsFs) GroupAdd(name string) (avfs.GroupReader, error) {
	if !fs.CurrentUser().IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	return fs.idm.GroupAdd(name)
}

// GroupDel deletes an existing group.
func (fs *OsFs) GroupDel(name string) error {
	if !fs.CurrentUser().IsRoot() {
		return avfs.ErrPermDenied
	}

	return fs.idm.GroupDel(name)
}

// LookupGroup looks up a group by name. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func (fs *OsFs) LookupGroup(name string) (avfs.GroupReader, error) {
	return fs.idm.LookupGroup(name)
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (fs *OsFs) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return fs.idm.LookupGroupId(gid)
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (fs *OsFs) LookupUser(name string) (avfs.UserReader, error) {
	return fs.idm.LookupUser(name)
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (fs *OsFs) LookupUserId(uid int) (avfs.UserReader, error) {
	return fs.idm.LookupUserId(uid)
}

// User sets the current user of the file system to uid.
// If the current user has not root privileges avfs.errPermDenied is returned.
func (fs *OsFs) User(name string) (avfs.UserReader, error) {
	uc, ok := fs.idm.(avfs.UserConnecter)
	if ok {
		return uc.User(name)
	}

	return nil, avfs.ErrNotImplemented
}

// UserAdd adds a new user.
func (fs *OsFs) UserAdd(name, groupName string) (avfs.UserReader, error) {
	if !fs.CurrentUser().IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	u, err := fs.idm.UserAdd(name, groupName)
	if err != nil {
		return nil, err
	}

	return fsutil.CreateHomeDir(fs, u)
}

// UserDel deletes an existing group.
func (fs *OsFs) UserDel(name string) error {
	if !fs.CurrentUser().IsRoot() {
		return avfs.ErrPermDenied
	}

	err := fs.idm.UserDel(name)
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
