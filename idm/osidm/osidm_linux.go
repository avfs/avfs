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

package osidm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/avfs/avfs"
)

// To avoid flaky tests when executing commands or making system calls as root,
// the current goroutine is locked to the operating system thread just before calling the function.
// For details see https://github.com/golang/go/issues/1435

// GroupAdd adds a new group.
func (idm *OsIdm) GroupAdd(name string) (avfs.GroupReader, error) {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
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
		case errStr == "groupadd: group '"+name+"' already exists":
			return nil, avfs.AlreadyExistsGroupError(name)
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
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
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
		case errStr == "groupdel: group '"+name+"' does not exist":
			return avfs.UnknownGroupError(name)
		default:
			return avfs.UnknownError(err.Error() + errStr)
		}
	}

	return nil
}

// LookupGroup looks up a group by name. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func (idm *OsIdm) LookupGroup(name string) (avfs.GroupReader, error) {
	return getGroup(name, avfs.UnknownGroupError(name))
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (idm *OsIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	sGid := strconv.Itoa(gid)

	return getGroup(sGid, avfs.UnknownGroupIdError(gid))
}

func getGroup(nameOrId string, notFoundErr error) (*OsGroup, error) {
	line, err := getent("group", nameOrId, notFoundErr)
	if err != nil {
		return nil, err
	}

	cols := strings.Split(line, ":")
	gid, _ := strconv.Atoi(cols[2])

	g := &OsGroup{
		name: cols[0],
		gid:  gid,
	}

	return g, nil
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (idm *OsIdm) LookupUser(name string) (avfs.UserReader, error) {
	return lookupUser(name)
}

func lookupUser(name string) (avfs.UserReader, error) {
	return getUser(name, avfs.UnknownUserError(name))
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (idm *OsIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	return lookupUserId(uid)
}

func lookupUserId(uid int) (avfs.UserReader, error) {
	sUid := strconv.Itoa(uid)

	return getUser(sUid, avfs.UnknownUserIdError(uid))
}

func getUser(nameOrId string, notFoundErr error) (*OsUser, error) {
	line, err := getent("passwd", nameOrId, notFoundErr)
	if err != nil {
		return nil, err
	}

	cols := strings.Split(line, ":")
	uid, _ := strconv.Atoi(cols[2])
	gid, _ := strconv.Atoi(cols[3])

	u := &OsUser{
		name: cols[0],
		uid:  uid,
		gid:  gid,
	}

	return u, nil
}

// SetUser sets and returns the current user.
// If the user is not found, the returned error is of type UnknownUserError.
func (idm *OsIdm) SetUser(name string) (avfs.UserReader, error) {
	return SetUser(name)
}

func SetUser(name string) (avfs.UserReader, error) {
	const op = "user"

	u, err := lookupUser(name)
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

// User returns the current user.
func (idm *OsIdm) User() avfs.UserReader {
	return User()
}

// User returns the current user of the OS.
func User() avfs.UserReader {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	uid := syscall.Geteuid()

	user, err := lookupUserId(uid)
	if err != nil {
		return nil
	}

	return user
}

// UserAdd adds a new user.
func (idm *OsIdm) UserAdd(name, groupName string) (avfs.UserReader, error) {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return nil, avfs.ErrPermDenied
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("useradd", "-M", "-g", groupName, name)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())

		switch {
		case errStr == "useradd: user '"+name+"' already exists":
			return nil, avfs.AlreadyExistsUserError(name)
		case errStr == "useradd: group '"+groupName+"' does not exist":
			return nil, avfs.UnknownGroupError(groupName)
		default:
			return nil, avfs.UnknownError(err.Error() + errStr)
		}
	}

	u, err := idm.LookupUser(name)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// UserDel deletes an existing user.
func (idm *OsIdm) UserDel(name string) error {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
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
		case errStr == "userdel: user '"+name+"' does not exist":
			return avfs.UnknownUserError(name)
		default:
			return avfs.UnknownError(err.Error() + errStr)
		}
	}

	return nil
}

func getent(database, key string, notFoundErr error) (string, error) {
	cmd := exec.Command("getent", database, key)

	buf, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			switch e.ExitCode() {
			case 1:
				return "", avfs.UnknownError("Missing arguments, or database unknown.")
			case 2:
				return "", notFoundErr
			case 3:
				return "", avfs.UnknownError("Enumeration not supported on this database.")
			}
		}

		return "", err
	}

	return string(buf), nil
}

// IsUserAdmin returns true if the current user has admin privileges.
func isUserAdmin() bool {
	return os.Geteuid() == 0
}
