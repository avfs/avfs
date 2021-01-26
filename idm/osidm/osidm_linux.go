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

// +build linux

package osidm

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/avfs/avfs"
)

// To avoid flaky tests when executing commands or making system calls as root,
// the current goroutine is locked to the operating system thread just before calling the function.
// For details see https://github.com/golang/go/issues/1435

// CurrentUser returns the current user.
func (idm *OsIdm) CurrentUser() avfs.UserReader {
	return currentUser()
}

// GroupAdd adds a new group.
func (idm *OsIdm) GroupAdd(name string) (avfs.GroupReader, error) {
	if !idm.initUser.IsRoot() || !idm.CurrentUser().IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("groupadd", name)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())

		switch {
		case err == exec.ErrNotFound:
			return nil, err
		case errStr == "groupadd: group '"+name+"' already exists":
			return nil, avfs.AlreadyExistsGroupError(name)
		case strings.HasPrefix(errStr, "groupadd: Permission denied."):
			return nil, avfs.ErrPermDenied
		default:
			return nil, avfs.UnknownError(err.Error() + errStr)
		}
	}

	g, err := idm.LookupGroup(name)
	if err != nil {
		return nil, err
	}

	return g, nil
}

// GroupDel deletes an existing group.
func (idm *OsIdm) GroupDel(name string) error {
	if !idm.initUser.IsRoot() || !idm.CurrentUser().IsRoot() {
		return avfs.ErrPermDenied
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("groupdel", name)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())

		switch {
		case err == exec.ErrNotFound:
			return err
		case errStr == "groupdel: group '"+name+"' does not exist":
			return avfs.UnknownGroupError(name)
		case strings.HasPrefix(errStr, "groupdel: Permission denied."):
			return avfs.ErrPermDenied
		default:
			return avfs.UnknownError(err.Error() + errStr)
		}
	}

	return nil
}

// LookupGroup looks up a group by name. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func (idm *OsIdm) LookupGroup(name string) (avfs.GroupReader, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	return lookupGroup(name)
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (idm *OsIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	return lookupGroupId(gid)
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (idm *OsIdm) LookupUser(name string) (avfs.UserReader, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	return lookupUser(name)
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (idm *OsIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	return lookupUserId(uid)
}

// User sets the current user of the file system to uid.
// If the current user has not root privileges avfs.errPermDenied is returned.
func (idm *OsIdm) User(name string) (avfs.UserReader, error) {
	const op = "user"

	if !idm.initUser.IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	u, err := lookupUser(name)
	if err != nil {
		return nil, err
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// If the current user is the target user there is nothing to do.
	curUid := syscall.Geteuid()
	if curUid == u.uid {
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

	if u.uid == 0 {
		return u, nil
	}

	runtime.LockOSThread()

	if err := syscall.Setresgid(u.gid, u.gid, 0); err != nil {
		return nil, avfs.UnknownError(fmt.Sprintf("%s : can't change gid to %d : %v", op, u.gid, err))
	}

	runtime.LockOSThread()

	if err := syscall.Setresuid(u.uid, u.uid, 0); err != nil {
		return nil, avfs.UnknownError(fmt.Sprintf("%s : can't change uid to %d : %v", op, u.uid, err))
	}

	return u, nil
}

// UserAdd adds a new user.
func (idm *OsIdm) UserAdd(name, groupName string) (avfs.UserReader, error) {
	if !idm.initUser.IsRoot() || !idm.CurrentUser().IsRoot() {
		return nil, avfs.ErrPermDenied
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("useradd", "-M", "-g", groupName, "-s", "/usr/sbin/nologin", name)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())

		switch {
		case err == exec.ErrNotFound:
			return nil, err
		case errStr == "useradd: user '"+name+"' already exists":
			return nil, avfs.AlreadyExistsUserError(name)
		case errStr == "useradd: group '"+groupName+"' does not exist":
			return nil, avfs.UnknownGroupError(groupName)
		case strings.HasPrefix(errStr, "useradd: Permission denied."):
			return nil, avfs.ErrPermDenied
		default:
			return nil, avfs.UnknownError(err.Error() + errStr)
		}
	}

	u, err := lookupUser(name)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// UserDel deletes an existing user.
func (idm *OsIdm) UserDel(name string) error {
	if !idm.initUser.IsRoot() || !idm.CurrentUser().IsRoot() {
		return avfs.ErrPermDenied
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("userdel", name)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())

		switch {
		case err == exec.ErrNotFound:
			return err
		case errStr == "userdel: user '"+name+"' does not exist":
			return avfs.UnknownUserError(name)
		case strings.HasPrefix(errStr, "userdel: Permission denied."):
			return avfs.ErrPermDenied
		default:
			return avfs.UnknownError(err.Error() + errStr)
		}
	}

	return nil
}
