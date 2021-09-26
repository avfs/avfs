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
// +build linux

package osidm

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/avfs/avfs"
)

const (
	groupFile = "/etc/group"
	userFile  = "/etc/passwd"
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

	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
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

	u, err := lookupUser(name)
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

func currentUser() *OsUser {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	uid := syscall.Geteuid()

	user, err := lookupUserId(uid)
	if err != nil {
		return nil
	}

	return user
}

type compareFunc func(line []string, value string) bool

func lookupGroup(name string) (*OsGroup, error) {
	return lookupGroupFunc(func(line []string, value string) bool { return line[0] == value },
		name,
		avfs.UnknownGroupError(name))
}

func lookupGroupId(gid int) (*OsGroup, error) {
	sGid := strconv.Itoa(gid)

	return lookupGroupFunc(func(line []string, value string) bool { return line[2] == value },
		sGid,
		avfs.UnknownGroupIdError(gid))
}

func lookupGroupFunc(compareFunc compareFunc, value string, notFoundErr error) (*OsGroup, error) {
	f, err := os.Open(groupFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	// Line format :
	// groupname:x:gid:
	r := csv.NewReader(f)
	r.Comma = ':'
	r.Comment = '#'
	r.FieldsPerRecord = 4

	for {
		line, err := r.Read()
		if err == io.EOF {
			return nil, notFoundErr
		}

		if err != nil {
			return nil, err
		}

		if compareFunc(line, value) {
			gid, _ := strconv.Atoi(line[2])

			g := &OsGroup{
				name: line[0],
				gid:  gid,
			}

			return g, nil
		}
	}
}

func lookupUser(name string) (*OsUser, error) {
	return lookupUserFunc(func(line []string, value string) bool { return line[0] == value },
		name,
		avfs.UnknownUserError(name))
}

func lookupUserId(uid int) (*OsUser, error) {
	sUid := strconv.Itoa(uid)

	return lookupUserFunc(func(line []string, value string) bool { return line[2] == value },
		sUid,
		avfs.UnknownUserIdError(uid))
}

func lookupUserFunc(compareFunc compareFunc, value string, notFoundErr error) (*OsUser, error) {
	f, err := os.Open(userFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	// Line format :
	// username:x:uid:gid::/home/username:/bin/bash
	r := csv.NewReader(f)
	r.Comma = ':'
	r.Comment = '#'
	r.FieldsPerRecord = 7

	for {
		line, err := r.Read()
		if err == io.EOF {
			return nil, notFoundErr
		}

		if err != nil {
			return nil, err
		}

		if compareFunc(line, value) {
			uid, _ := strconv.Atoi(line[2])
			gid, _ := strconv.Atoi(line[3])

			u := &OsUser{
				name: line[0],
				uid:  uid,
				gid:  gid,
			}

			return u, nil
		}
	}
}
