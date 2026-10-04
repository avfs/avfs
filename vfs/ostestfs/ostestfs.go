//
//  Copyright 2026 The AVFS authors
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

// Package ostestfs implements a file system using functions from os and path/filepath packages.
//
// Most functions are just calls to the original ones from os and filepath packages.
package ostestfs

import (
	"io/fs"
	"os"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/osidm"
)

// Chown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link's target.
// A uid or gid of -1 means to not change that value.
// If there is an error, it will be of type *PathError.
//
// On Windows or Plan 9, Chown always returns the syscall.EWINDOWS or
// EPLAN9 error, wrapped in *PathError.
func (vfs *OsTestFS) Chown(name string, uid, gid int) error {
	const op = "chown"

	if !vfs.HasFeature(avfs.FeatIdentityMgr) && vfs.OSType() != avfs.OsWindows {
		return &fs.PathError{Op: op, Path: name, Err: avfs.ErrOpNotPermitted}
	}

	return os.Chown(name, uid, gid)
}

// Lchown changes the numeric uid and gid of the named file.
// If the file is a symbolic link, it changes the uid and gid of the link itself.
// If there is an error, it will be of type *PathError.
//
// On Windows, it always returns the syscall.EWINDOWS error, wrapped
// in *PathError.
func (vfs *OsTestFS) Lchown(name string, uid, gid int) error {
	const op = "lchown"

	if !vfs.HasFeature(avfs.FeatIdentityMgr) && vfs.OSType() != avfs.OsWindows {
		return &os.PathError{Op: op, Path: name, Err: avfs.ErrOpNotPermitted}
	}

	return os.Lchown(name, uid, gid)
}

// SetUser sets the user of the process running the file system.
//
// This is **not** a file system operation: OsTestFS is a real file system, so
// its user is the user of the process, and there is no view of it acting as
// another user. Changing the user changes the credentials of the whole process
// (see osidm.SetUser), which is why this exists only to run tests, and why such
// a file system is not an avfs.Cloner.
//
// If the user can't be changed, an error is returned.
func (vfs *OsTestFS) SetUser(user avfs.UserReader) error {
	return osidm.SetUser(user)
}

// SetUserByName sets the user of the process running the file system.
//
// See SetUser: this changes the credentials of the process, not the identity of
// a file system. If the user is not found, the returned error is of type
// avfs.UnknownUserError.
func (vfs *OsTestFS) SetUserByName(name string) error {
	return osidm.SetUserByName(name)
}

// User returns the current user.
func (vfs *OsTestFS) User() avfs.UserReader {
	return osidm.User()
}
