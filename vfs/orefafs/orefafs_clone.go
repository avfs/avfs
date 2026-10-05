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

package orefafs

import (
	"github.com/avfs/avfs"
)

// CloneWithUser returns a shallow copy of the current file system (see OrefaFS)
// acting as user and emulating ost.
//
// The clone starts in the home directory of user, which is created in the
// shared content if it does not exist yet.
//
// If user is nil, the administrator of the identity manager is used. If ost is
// avfs.OsUnknown, the OS type of the current file system is kept. If ost can't
// be set (see avfs.ErrSetOSType), it returns that error.
//
// The content is not converted to the new OS type: a clone emulating another OS
// resolves paths with the rules, and reports the errors, of that OS, but the
// directories already created keep the names and the layout of the OS the
// content was built for.
func (vfs *OrefaFS) CloneWithUser(user avfs.UserReader, ost avfs.OSType) (avfs.VFS, error) {
	return vfs.cloneWithUser(user, ost)
}

// CloneWithUserName returns a shallow copy of the current file system (see
// OrefaFS) acting as the user userName and emulating ost.
//
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (vfs *OrefaFS) CloneWithUserName(userName string, ost avfs.OSType) (avfs.VFS, error) {
	user, err := vfs.Idm().LookupUser(userName)
	if err != nil {
		return nil, err
	}

	return vfs.cloneWithUser(user, ost)
}

// cloneWithUser returns a copy of the file system acting as user and emulating
// ost, sharing its Storage.
func (vfs *OrefaFS) cloneWithUser(user avfs.UserReader, ost avfs.OSType) (*OrefaFS, error) {
	if ost == avfs.OsUnknown {
		ost = vfs.OSType()
	}

	if user == nil {
		user = vfs.Idm().AdminUser()
	}

	// The clone is initialized from scratch, not copied: an OrefaFS holds an
	// atomic pointer (its current directory), so copying one would copy a lock.
	c := &OrefaFS{storage: vfs.storage}

	err := c.userDir.Init(ost, vfs.Idm(), user)
	if err != nil {
		return nil, err
	}

	// The features and the umask are those of the original file system,
	// whatever the OS type of the clone: they describe the content and the
	// creation policy, not the identity of the view.
	_ = c.SetFeatures(vfs.Features())
	_ = c.SetUMask(vfs.UMask())

	c.err = avfs.ErrorsFor(c.OSType())

	// The clone starts in the home directory of user, which is created in the
	// shared content if it does not exist yet, along with the user-specific
	// directories of the OS, such as the temporary directory on Windows and
	// Darwin. Nothing is created when the clone emulates another OS: the content
	// is not converted to it (see the AVFS specification), so the directories of
	// that OS are not either.
	if c.OSType() == vfs.OSType() {
		err = avfs.MkDirs(c, avfs.UserDirsInfo(c, user), "")
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}
