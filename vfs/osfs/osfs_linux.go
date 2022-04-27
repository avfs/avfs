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

//go:build linux

package osfs

import (
	"fmt"
	"io/fs"
	"runtime"
	"syscall"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/osidm"
)

// Chroot changes the root to that specified in path.
// If there is an error, it will be of type *PathError.
func (vfs *OsFS) Chroot(path string) error {
	const op = "chroot"

	if !vfs.HasFeature(avfs.FeatIdentityMgr) {
		return &fs.PathError{Op: op, Path: path, Err: avfs.ErrOpNotPermitted}
	}

	err := syscall.Chroot(path)
	if err != nil {
		return &fs.PathError{Op: op, Path: path, Err: err}
	}

	return nil
}

// SetUser sets and returns the current user.
// If the user is not found, the returned error is of type UnknownUserError.
func SetUser(name string) (avfs.UserReader, error) {
	const op = "user"

	u, err := osidm.LookupUser(name)
	if err != nil {
		return nil, err
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// If the current user is the target user there is nothing to do.
	curUid := syscall.Geteuid()
	if curUid == u.Uid() {
		return u, nil
	}

	runtime.LockOSThread()

	curGid := syscall.Getegid()

	// If the current user is not root, root privileges must be restored
	// before setting the new uid and gid.
	if curGid != 0 {
		runtime.LockOSThread()

		if err := syscall.Setresgid(0, 0, 0); err != nil {
			return nil, avfs.UnknownError(fmt.Sprintf("%s : can't change gid to %d : %v", op, 0, err))
		}
	}

	if curUid != 0 {
		runtime.LockOSThread()

		if err := syscall.Setresuid(0, 0, 0); err != nil {
			return nil, avfs.UnknownError(fmt.Sprintf("%s : can't change uid to %d : %v", op, 0, err))
		}
	}

	if u.Uid() == 0 {
		return u, nil
	}

	runtime.LockOSThread()

	if err := syscall.Setresgid(u.Gid(), u.Gid(), 0); err != nil {
		return nil, avfs.UnknownError(fmt.Sprintf("%s : can't change gid to %d : %v", op, u.Gid(), err))
	}

	runtime.LockOSThread()

	if err := syscall.Setresuid(u.Uid(), u.Uid(), 0); err != nil {
		return nil, avfs.UnknownError(fmt.Sprintf("%s : can't change uid to %d : %v", op, u.Uid(), err))
	}

	return u, nil
}

// ToSysStat takes a value from fs.FileInfo.Sys() and returns a value that implements interface avfs.SysStater.
func (vfs *OsFS) ToSysStat(info fs.FileInfo) avfs.SysStater {
	return &LinuxSysStat{Sys: info.Sys().(*syscall.Stat_t)} // nolint:forcetypeassert // type assertion must be checked
}

// User returns the current user of the OS.
func User() avfs.UserReader {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	uid := syscall.Geteuid()

	user, err := osidm.LookupUserId(uid)
	if err != nil {
		return nil
	}

	return user
}

// LinuxSysStat implements SysStater interface returned by fs.FileInfo.Sys() for a Linux file system.
type LinuxSysStat struct {
	Sys *syscall.Stat_t
}

// Gid returns the group id.
func (lst *LinuxSysStat) Gid() int {
	return int(lst.Sys.Gid)
}

// Uid returns the user id.
func (lst *LinuxSysStat) Uid() int {
	return int(lst.Sys.Uid)
}

// Nlink returns the number of hard links.
func (lst *LinuxSysStat) Nlink() uint64 {
	return uint64(lst.Sys.Nlink) //nolint:unconvert // required for 32 bits systems.
}
