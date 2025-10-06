//
//  Copyright 2025 The AVFS authors
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

//go:build darwin

package osidm

import (
	"fmt"
	"math"
	"os"
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

var (
	groupReadRE = regexp.MustCompile(`PrimaryGroupID: (\d+)\nRecordName: (\s+)`)
	userReadRE  = regexp.MustCompile(`PrimaryGroupID: (\d+)\nRecordName: (\s+)\nUniqueID: (\d+)`)
)

// AddGroup creates a new group with the specified name.
// If the group already exists, the returned error is of type avfs.AlreadyExistsGroupError.
func (idm *OsIdm) AddGroup(groupName string) (avfs.GroupReader, error) {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return nil, avfs.ErrPermDenied
	}

	if !idm.IsValidNameFunc(groupName) {
		return nil, avfs.InvalidNameError(groupName)
	}

	_, err := dsEditGroup("create", groupName)
	if err != nil {
		return nil, err
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

	if !idm.IsValidNameFunc(userName) {
		return nil, avfs.InvalidNameError(userName)
	}

	if !idm.IsValidNameFunc(groupName) {
		return nil, avfs.InvalidNameError(groupName)
	}

	g, err := lookupGroup(groupName)
	if err != nil {
		return nil, err
	}

	sGid := strconv.Itoa(g.Gid())

	_, err = sysAdminCtl("create", userName, "-GID", sGid)
	if err != nil {
		return nil, err
	}

	u, err := lookupUser(userName)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// AddUserToGroup adds the user to the group.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
func (idm *OsIdm) AddUserToGroup(userName, groupName string) error {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return avfs.ErrPermDenied
	}

	if !idm.IsValidNameFunc(userName) {
		return avfs.InvalidNameError(userName)
	}

	if !idm.IsValidNameFunc(groupName) {
		return avfs.InvalidNameError(groupName)
	}

	u, err := lookupUser(userName)
	if err != nil {
		return err
	}

	g, err := lookupGroup(groupName)
	if err != nil {
		return err
	}

	if u.IsInGroupId(g.Gid()) {
		return avfs.AlreadyExistsGroupError(groupName)
	}

	_, err = dsEditGroup("edit", "-a", u.Name(), "-t", "user", g.Name())
	if err != nil {
		return err
	}

	return nil
}

// DelGroup deletes an existing group with the specified name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *OsIdm) DelGroup(groupName string) error {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return avfs.ErrPermDenied
	}

	if !idm.IsValidNameFunc(groupName) {
		return avfs.InvalidNameError(groupName)
	}

	_, err := dsEditGroup("delete", groupName)
	if err != nil {
		return err
	}

	return nil
}

// DelUser deletes an existing user with the specified name.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *OsIdm) DelUser(userName string) error {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return avfs.ErrPermDenied
	}

	if !idm.IsValidNameFunc(userName) {
		return avfs.InvalidNameError(userName)
	}

	_, err := sysAdminCtl("delete", userName)
	if err != nil {
		return err
	}

	return nil
}

// DelUserFromGroup removes the user from the group.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError
// or avfs.UnknownGroupError respectively.
func (idm *OsIdm) DelUserFromGroup(userName, groupName string) error {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return avfs.ErrPermDenied
	}

	if !idm.IsValidNameFunc(userName) {
		return avfs.InvalidNameError(userName)
	}

	if !idm.IsValidNameFunc(groupName) {
		return avfs.InvalidNameError(groupName)
	}

	u, err := lookupUser(userName)
	if err != nil {
		return err
	}

	g, err := lookupGroup(groupName)
	if err != nil {
		return err
	}

	if !u.IsInGroupId(g.Gid()) {
		return avfs.UnknownGroupError(groupName)
	}

	_, err = dsEditGroup("edit", "-d", u.Name(), "-t", "user", g.Name())
	if err != nil {
		return err
	}

	return nil
}

// LookupGroup looks up a group by name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *OsIdm) LookupGroup(groupName string) (avfs.GroupReader, error) {
	if !idm.IsValidNameFunc(groupName) {
		return nil, avfs.InvalidNameError(groupName)
	}

	return lookupGroup(groupName)
}

// lookupGroup looks up a group by name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func lookupGroup(groupName string) (avfs.GroupReader, error) {
	buf, err := dscl("read", "Groups/"+groupName, "PrimaryGroupID", "RecordName")
	if err != nil {
		return nil, err
	}

	r := groupReadRE.FindStringSubmatch(buf)
	if r == nil {
		return nil, avfs.UnknownGroupError(groupName)
	}

	gid, _ := strconv.Atoi(r[1])
	name := r[2]

	g := &OsGroup{name: name, gid: gid}

	return g, nil
}

// LookupGroupId looks up a group by groupid.
// If the group is not found, the returned error is of type avfs.UnknownGroupIdError.
func (idm *OsIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	sGid := strconv.Itoa(gid)

	buf, err := dscl("search", "/Groups", "PrimaryGroupID", sGid)
	if err != nil {
		return nil, err
	}

	name := strings.Split(buf, "\t")[0]

	return lookupGroup(name)
}

// LookupUser looks up a user by username.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *OsIdm) LookupUser(userName string) (avfs.UserReader, error) {
	if !idm.IsValidNameFunc(userName) {
		return nil, avfs.InvalidNameError(userName)
	}

	return lookupUser(userName)
}

func lookupUser(userName string) (*OsUser, error) {
	buf, err := dscl("read", "/Users/"+userName, "PrimaryGroupID", "RecordName", "UniqueID")
	if err != nil {
		return nil, err
	}

	r := userReadRE.FindStringSubmatch(buf)
	if r == nil {
		return nil, avfs.UnknownUserError(userName)
	}

	gid, _ := strconv.Atoi(r[1])
	name := r[2]
	uid, _ := strconv.Atoi(r[3])

	u := &OsUser{name: name, uid: uid, gid: gid}

	return u, nil
}

// LookupUserId looks up a user by userid.
// If the user is not found, the returned error is of type avfs.UnknownUserIdError.
func (idm *OsIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	return lookupUserId(uid)
}

// lookupUserId retrieves user information by user ID. It returns an avfs.UserReader and an error if the user is not found.
func lookupUserId(uid int) (avfs.UserReader, error) {
	sUid := strconv.Itoa(uid)

	buf, err := dscl("search", "/Users", "UniqueID", sUid)
	if err != nil {
		return nil, err
	}

	name := strings.Split(buf, "\t")[0]

	return lookupUser(name)
}

// SetUserPrimaryGroup sets the primary group of a user to the specified group name.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
// If the operation fails, the returned error is of type avfs.UnknownError.
func (idm *OsIdm) SetUserPrimaryGroup(userName, groupName string) error {
	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return avfs.ErrPermDenied
	}

	if !idm.IsValidNameFunc(userName) {
		return avfs.InvalidNameError(userName)
	}

	if !idm.IsValidNameFunc(groupName) {
		return avfs.InvalidNameError(groupName)
	}

	u, err := lookupUser(userName)
	if err != nil {
		return err
	}

	g, err := lookupGroup(groupName)
	if err != nil {
		return err
	}

	sGid := strconv.Itoa(g.Gid())

	_, err = dscl("create", "/Users/"+u.Name(), "PrimaryGroupID", sGid)
	if err != nil {
		return err
	}

	return nil
}

// Groups returns a slice of strings representing the group names that the user belongs to.
// If an error occurs while fetching the group names, it returns nil.
func (u *OsUser) Groups() []string {
	groupNames, err := id(u.name, "-Gn")
	if err != nil {
		return nil
	}

	return strings.Split(groupNames, " ")
}

// GroupsId returns a slice group IDs that the user belongs to.
// If an error occurs while fetching the group IDs, it returns nil.
func (u *OsUser) GroupsId() []int {
	groupIds, err := id(u.name, "-G")
	if err != nil {
		return nil
	}

	var (
		gids []int
		pos  int
	)

	for s := groupIds; ; s = s[pos+1:] {
		pos = strings.IndexByte(s, ' ')
		if pos == -1 {
			gid, err := strconv.Atoi(s)
			if err == nil {
				gids = append(gids, gid)
			}

			return gids
		}

		gid, err := strconv.Atoi(s[:pos])
		if err == nil {
			gids = append(gids, gid)
		}
	}
}

// IsInGroupId returns true if the user is in the specified group ID.
func (u *OsUser) IsInGroupId(gid int) bool {
	sGids, err := id(u.name, "-G")
	if err != nil {
		return false
	}

	sGid := strconv.Itoa(gid) + " "

	return strings.Contains(sGids+" ", sGid)
}

// PrimaryGroup returns the primary group name of the user.
func (u *OsUser) PrimaryGroup() string {
	groupName, err := id(u.name, "-gn")
	if err != nil {
		return ""
	}

	return groupName
}

// PrimaryGroupId returns the primary group ID of the OsUser.
// If an error occurs, it returns the maximum integer value.
func (u *OsUser) PrimaryGroupId() int {
	sGid, err := id(u.name, "-g")
	if err != nil {
		return math.MaxInt
	}

	gid, err := strconv.Atoi(sGid)
	if err != nil {
		return math.MaxInt
	}

	return gid
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

		if err := syscall.Setgid(0); err != nil {
			return avfs.UnknownError(fmt.Sprintf("%s : can't change gid to %d : %v", op, 0, err))
		}
	}

	if curUid != 0 {
		runtime.LockOSThread()

		if err := syscall.Setuid(0); err != nil {
			return avfs.UnknownError(fmt.Sprintf("%s : can't change uid to %d : %v", op, 0, err))
		}
	}

	if user.Uid() == 0 {
		return nil
	}

	runtime.LockOSThread()

	if err := syscall.Setgid(user.Gid()); err != nil {
		return avfs.UnknownError(fmt.Sprintf("%s : can't change gid to %d : %v", op, user.Gid(), err))
	}

	runtime.LockOSThread()

	if err := syscall.Setuid(user.Uid()); err != nil {
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

// id returns the result of the "id" command for the given username and options.
// The result is returned as a string, and an error is returned if the command fails.
func id(username, options string) (string, error) {
	buf, err := output("id", options, username)
	if err != nil {
		return "", err
	}

	return buf, nil
}

// dscl calls the Directory Service command line utility.
func dscl(command string, args ...string) (string, error) {
	args = append([]string{"-q", ".", "-" + command}, args...)
	buf, err := output("dscl", args...)

	return buf, err
}

// dsEditGroup calls the Directory Service group record manipulation tool.
func dsEditGroup(command string, args ...string) (string, error) {
	args = append([]string{"-q", "-" + command}, args...)
	buf, err := output("desedseditgrou", args...)

	return buf, err
}

// sysAdminCtl calls the sysadminctl command line utility.
func sysAdminCtl(command string, args ...string) (string, error) {
	args = append([]string{"-" + command}, args...)
	buf, err := output("sysadminctl", args...)

	return buf, err
}

// IsUserAdmin returns true if the current user has admin privileges.
func isUserAdmin() bool {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	return os.Geteuid() == 0
}
