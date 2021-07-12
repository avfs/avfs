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
	"bufio"
	"bytes"
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

// To avoid flaky tests when executing commands or making system calls as root,
// the current goroutine is locked to the operating system thread just before calling the function.
// For details see https://github.com/golang/go/issues/1435

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
		case errStr == "userdel: user '"+name+"' does not exist":
			return avfs.UnknownUserError(name)
		default:
			return avfs.UnknownError(err.Error() + errStr)
		}
	}

	return nil
}

const (
	groupFile = "/etc/group"
	userFile  = "/etc/passwd"
)

var colon = []byte{':'} //nolint:gochecknoglobals // Used in matchGroupIndexValue and matchUserIndexValue.

// lineFunc returns a value, an error, or (nil, nil) to skip the row.
type lineFunc func(line []byte) (v interface{}, err error)

// readColonFile parses r as an /etc/group or /etc/passwd style file, running
// fn for each row. readColonFile returns a value, an error, or (nil, nil) if
// the end of the file is reached without a match.
func readColonFile(r io.Reader, fn lineFunc) (v interface{}, err error) {
	bs := bufio.NewScanner(r)
	for bs.Scan() {
		line := bs.Bytes()
		// There's no spec for /etc/passwd or /etc/group, but we try to follow
		// the same rules as the glibc parser, which allows comments and blank
		// space at the beginning of a line.
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		v, err = fn(line)
		if v != nil || err != nil {
			return
		}
	}

	return nil, bs.Err()
}

func matchGroupIndexValue(value string, idx int) lineFunc {
	var leadColon string
	if idx > 0 {
		leadColon = ":"
	}

	substr := []byte(leadColon + value + ":")

	return func(line []byte) (v interface{}, err error) {
		if !bytes.Contains(line, substr) || bytes.Count(line, colon) < 3 {
			return
		}
		// wheel:*:0:root
		parts := strings.SplitN(string(line), ":", 4)
		if len(parts) < 4 || parts[0] == "" || parts[idx] != value ||
			// If the file contains +foo and you search for "foo", glibc
			// returns an "invalid argument" error. Similarly, if you search
			// for a gid for a row where the group name starts with "+" or "-",
			// glibc fails to find the record.
			parts[0][0] == '+' || parts[0][0] == '-' {
			return
		}

		gid, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, nil
		}

		return &Group{name: parts[0], gid: gid}, nil
	}
}

func findGroupId(gid int, r io.Reader) (*Group, error) {
	sGid := strconv.Itoa(gid)
	if v, err := readColonFile(r, matchGroupIndexValue(sGid, 2)); err != nil {
		return nil, err
	} else if v != nil {
		return v.(*Group), nil
	}

	return nil, avfs.UnknownGroupIdError(gid)
}

func findGroupName(name string, r io.Reader) (*Group, error) {
	if v, err := readColonFile(r, matchGroupIndexValue(name, 0)); err != nil {
		return nil, err
	} else if v != nil {
		return v.(*Group), nil
	}

	return nil, avfs.UnknownGroupError(name)
}

// returns a *User for a row if that row's has the given value at the
// given index.
func matchUserIndexValue(value string, idx int) lineFunc {
	var leadColon string
	if idx > 0 {
		leadColon = ":"
	}

	substr := []byte(leadColon + value + ":")

	return func(line []byte) (v interface{}, err error) {
		if !bytes.Contains(line, substr) || bytes.Count(line, colon) < 6 {
			return
		}

		// kevin:x:1005:1006::/home/kevin:/usr/bin/zsh
		parts := strings.SplitN(string(line), ":", 7)
		if len(parts) < 6 || parts[idx] != value || parts[0] == "" ||
			parts[0][0] == '+' || parts[0][0] == '-' {
			return
		}

		uid, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, nil
		}

		gid, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, nil
		}

		u := &User{
			name: parts[0],
			uid:  uid,
			gid:  gid,
		}

		// The pw_gecos field isn't quite standardized. Some docs
		// say: "It is expected to be a comma separated list of
		// personal data where the first item is the full name of the
		// user."
		if i := strings.Index(u.name, ","); i >= 0 {
			u.name = u.name[:i]
		}

		return u, nil
	}
}

func findUserId(uid int, r io.Reader) (*User, error) {
	sUid := strconv.Itoa(uid)
	if v, err := readColonFile(r, matchUserIndexValue(sUid, 2)); err != nil {
		return nil, err
	} else if v != nil {
		return v.(*User), nil
	}

	return nil, avfs.UnknownUserIdError(uid)
}

func findUsername(name string, r io.Reader) (*User, error) {
	if v, err := readColonFile(r, matchUserIndexValue(name, 0)); err != nil {
		return nil, err
	} else if v != nil {
		return v.(*User), nil
	}

	return nil, avfs.UnknownUserError(name)
}

func lookupGroup(groupname string) (*Group, error) {
	f, err := os.Open(groupFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return findGroupName(groupname, f)
}

func lookupGroupId(gid int) (*Group, error) {
	f, err := os.Open(groupFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return findGroupId(gid, f)
}

func lookupUser(username string) (*User, error) {
	f, err := os.Open(userFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return findUsername(username, f)
}

func lookupUserId(uid int) (*User, error) {
	f, err := os.Open(userFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return findUserId(uid, f)
}

func currentUser() *User {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	uid := syscall.Geteuid()

	user, err := lookupUserId(uid)
	if err != nil {
		return nil
	}

	return user
}
