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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/avfs/avfs"
)

// To avoid flaky tests when executing commands or making system calls as root,
// the current goroutine is locked to the operating system thread just before calling the function.
// For details see https://github.com/golang/go/issues/1435

// AddGroup creates a new group with the specified name.
// If the group already exists, the returned error is of type avfs.AlreadyExistsGroupError.
func (idm *OsIdm) AddGroup(groupName string) (avfs.GroupReader, error) {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return nil, avfs.ErrPermDenied
	}

	if !isValidLinuxName(groupName) {
		return nil, avfs.InvalidNameError(groupName)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("groupadd", groupName)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())

		switch {
		case errStr == "groupadd: group '"+groupName+"' already exists":
			return nil, avfs.AlreadyExistsGroupError(groupName)
		default:
			return nil, avfs.UnknownError(err.Error() + errStr)
		}
	}

	g, err := idm.LookupGroup(groupName)
	if err != nil {
		return nil, err
	}

	return g, nil
}

// AddUser creates a new user with the specified userName and the specified primary group groupName.
// If the user already exists, the returned error is of type avfs.AlreadyExistsUserError.
func (idm *OsIdm) AddUser(userName, groupName string) (avfs.UserReader, error) {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return nil, avfs.ErrPermDenied
	}

	if !isValidLinuxName(userName) {
		return nil, avfs.InvalidNameError(userName)
	}

	if !isValidLinuxName(groupName) {
		return nil, avfs.InvalidNameError(groupName)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("useradd", "-M", "-g", groupName, userName)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errStr := strings.TrimSpace(stderr.String())

		switch {
		case errStr == "useradd: user '"+userName+"' already exists":
			return nil, avfs.AlreadyExistsUserError(userName)
		case errStr == "useradd: group '"+groupName+"' does not exist":
			return nil, avfs.UnknownGroupError(groupName)
		default:
			return nil, avfs.UnknownError(err.Error() + errStr)
		}
	}

	u, err := idm.LookupUser(userName)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// DelGroup deletes an existing group with the specified name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *OsIdm) DelGroup(groupName string) error {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return avfs.ErrPermDenied
	}

	if !isValidLinuxName(groupName) {
		return avfs.InvalidNameError(groupName)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("groupdel", groupName)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())

		switch {
		case errStr == "groupdel: group '"+groupName+"' does not exist":
			return avfs.UnknownGroupError(groupName)
		default:
			return avfs.UnknownError(err.Error() + errStr)
		}
	}

	return nil
}

// DelUser deletes an existing user with the specified name.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *OsIdm) DelUser(userName string) error {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return avfs.ErrPermDenied
	}

	if !isValidLinuxName(userName) {
		return avfs.InvalidNameError(userName)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cmd := exec.Command("userdel", userName)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errStr := strings.TrimSpace(stderr.String())

		switch {
		case errStr == "userdel: user '"+userName+"' does not exist":
			return avfs.UnknownUserError(userName)
		default:
			return avfs.UnknownError(err.Error() + errStr)
		}
	}

	return nil
}

// LookupGroup looks up a group by name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *OsIdm) LookupGroup(groupName string) (avfs.GroupReader, error) {
	return getGroup(groupName, avfs.UnknownGroupError(groupName))
}

// LookupGroupId looks up a group by groupid.
// If the group is not found, the returned error is of type avfs.UnknownGroupIdError.
func (idm *OsIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	sGid := strconv.Itoa(gid)

	return getGroup(sGid, avfs.UnknownGroupIdError(gid))
}

// getGroup retrieves a group based on the provided name or ID.
// It returns an OsGroup struct or an error if the group is not found.
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

// LookupUser looks up a user by username.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *OsIdm) LookupUser(userName string) (avfs.UserReader, error) {
	return lookupUser(userName)
}

func lookupUser(userName string) (avfs.UserReader, error) {
	if !isValidLinuxName(userName) {
		return nil, avfs.InvalidNameError(userName)
	}

	return getUser(userName, avfs.UnknownUserError(userName))
}

// LookupUserId looks up a user by userid.
// If the user is not found, the returned error is of type avfs.UnknownUserIdError.
func (idm *OsIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	return lookupUserId(uid)
}

// lookupUserId retrieves user information by user ID. It returns an avfs.UserReader and an error if the user is not found.
func lookupUserId(uid int) (avfs.UserReader, error) {
	sUid := strconv.Itoa(uid)

	return getUser(sUid, avfs.UnknownUserIdError(uid))
}

// getUser retrieves user information based on either a username or user ID.
// It returns an OsUser pointer and an error if any occurs during retrieval.
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

// SetUser sets the current user.
// If the user can't be changed an error is returned.
func SetUser(user avfs.UserReader) error {
	const op = "user"

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// If the current user is the target user there is nothing to do.
	curUid := syscall.Geteuid()
	if curUid == user.Uid() {
		return nil
	}

	runtime.LockOSThread()

	curGid := syscall.Getegid()

	// If the current user is not root, root privileges must be restored
	// before setting the new uid and gid.
	if curGid != 0 {
		runtime.LockOSThread()

		if err := syscall.Setresgid(0, 0, 0); err != nil {
			return avfs.UnknownError(fmt.Sprintf("%s : can't change gid to %d : %v", op, 0, err))
		}
	}

	if curUid != 0 {
		runtime.LockOSThread()

		if err := syscall.Setresuid(0, 0, 0); err != nil {
			return avfs.UnknownError(fmt.Sprintf("%s : can't change uid to %d : %v", op, 0, err))
		}
	}

	if user.Uid() == 0 {
		return nil
	}

	runtime.LockOSThread()

	if err := syscall.Setresgid(user.Gid(), user.Gid(), 0); err != nil {
		return avfs.UnknownError(fmt.Sprintf("%s : can't change gid to %d : %v", op, user.Gid(), err))
	}

	runtime.LockOSThread()

	if err := syscall.Setresuid(user.Uid(), user.Uid(), 0); err != nil {
		return avfs.UnknownError(fmt.Sprintf("%s : can't change uid to %d : %v", op, user.Uid(), err))
	}

	return nil
}

// SetUserByName sets the current user by name.
// If the user is not found, the returned error is of type UnknownUserError.
func SetUserByName(userName string) error {
	u, err := lookupUser(userName)
	if err != nil {
		return err
	}

	return SetUser(u)
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

// getent retrieves an entry from the specified database using the `getent` command.
// If the entry is not found or if there's an error, it returns a corresponding error.
// Parameters:
// - database: The name of the database to query (e.g., "passwd", "group").
// - key: The key to look up in the database.
// - notFoundErr: The error to return if the key is not found.
// Returns the retrieved entry as a string or an error if any occurs.
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

// isValidLinuxName checks if the input string is a valid username or group name.
func isValidLinuxName(name string) bool {
	return regexp.MustCompile("^[a-zA-Z0-9_-]+$").MatchString(name)
}
