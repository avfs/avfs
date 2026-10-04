//
//  Copyright 2024 The AVFS authors
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
)

// CloneWithUser returns a shallow copy of the current file system (see MemFs)
// acting as user and emulating ost.
//
// The clone shares the Storage (the nodes, the volumes and the identity views)
// of the file system it is copied from, so a file created through either of them
// is visible from both. Cloning twice for the same user and OS type returns two
// file systems sharing the same identity view, hence the same current
// directory: Chdir on one of them is visible in the other.
//
// The home and temporary directories of user are not created: a view only refers
// to them, so they must already exist in the shared content, unless user is the
// administrator.
//
// If user is nil, the administrator of the identity manager is used. If ost is
// avfs.OsUnknown, the OS type of the current file system is kept. If ost can't
// be set (see avfs.ErrSetOSType), it returns that error.
//
// The content is not converted to the new OS type: a view emulating another OS
// resolves paths with the rules, and reports the errors, of that OS, but the
// directories already created keep the names and the layout of the OS the
// content was built for.
func (vfs *MemFS) CloneWithUser(user avfs.UserReader, ost avfs.OSType) (avfs.VFS, error) {
	return vfs.cloneWithUser(user, ost)
}

// CloneWithUserName returns a shallow copy of the current file system (see
// MemFs) acting as the user userName and emulating ost.
//
// If the user is not found, the returned error is of type avfs.UnknownUserError,
// or avfs.ErrPermDenied if the identity manager of the file system holds no
// user.
func (vfs *MemFS) CloneWithUserName(userName string, ost avfs.OSType) (avfs.VFS, error) {
	// A file system whose identity manager holds no user answers
	// ErrPermDenied, as LookupUser does.
	user, err := vfs.Idm().LookupUser(userName)
	if err != nil {
		return nil, err
	}

	return vfs.cloneWithUser(user, ost)
}

// cloneWithUser returns a view of the file system acting as user and emulating
// ost, sharing its Storage.
func (vfs *MemFS) cloneWithUser(user avfs.UserReader, ost avfs.OSType) (*MemFS, error) {
	if ost == avfs.OsUnknown {
		ost = vfs.OSType()
	}

	if user == nil {
		user = vfs.Idm().AdminUser()
	}

	userDir, err := vfs.storage.userDirFor(vfs.Idm(), user, ost, vfs.Features(), vfs.UMask())
	if err != nil {
		return nil, err
	}

	return &MemFS{
		err:     avfs.ErrorsFor(userDir.OSType()),
		storage: vfs.storage,
		userDir: userDir,
	}, nil
}
