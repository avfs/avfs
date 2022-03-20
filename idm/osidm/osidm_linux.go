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
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

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
	return LookupGroup(name)
}

// LookupGroupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (idm *OsIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return LookupGroupId(gid)
}

// LookupUser looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownUserError.
func (idm *OsIdm) LookupUser(name string) (avfs.UserReader, error) {
	return LookupUser(name)
}

// LookupUserId looks up a user by userid. If the user cannot be found, the
// returned error is of type UnknownUserIdError.
func (idm *OsIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	return LookupUserId(uid)
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

	u, err := LookupUser(name)
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

type compareFunc func(line []string, value string) bool

func LookupGroup(name string) (*OsGroup, error) {
	return lookupGroupFunc(func(line []string, value string) bool { return line[0] == value },
		name,
		avfs.UnknownGroupError(name))
}

func LookupGroupId(gid int) (*OsGroup, error) {
	sGid := strconv.Itoa(gid)

	return lookupGroupFunc(func(line []string, value string) bool { return line[2] == value },
		sGid,
		avfs.UnknownGroupIdError(gid))
}

func lookupGroupFunc(compareFunc compareFunc, value string, notFoundErr error) (*OsGroup, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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

func LookupUser(name string) (*OsUser, error) {
	return lookupUserFunc(func(line []string, value string) bool { return line[0] == value },
		name,
		avfs.UnknownUserError(name))
}

func LookupUserId(uid int) (*OsUser, error) {
	sUid := strconv.Itoa(uid)

	return lookupUserFunc(func(line []string, value string) bool { return line[2] == value },
		sUid,
		avfs.UnknownUserIdError(uid))
}

func lookupUserFunc(compareFunc compareFunc, value string, notFoundErr error) (*OsUser, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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
