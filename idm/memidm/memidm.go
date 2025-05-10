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

// Package memidm implements an in memory identity manager.
//
// functions updating users or groups aren't safe for concurrent use
package memidm

import "github.com/avfs/avfs"

// AdminGroup returns the administrator (root) group.
func (idm *MemIdm) AdminGroup() avfs.GroupReader {
	return idm.adminGroup
}

// AdminUser returns the administrator (root) user.
func (idm *MemIdm) AdminUser() avfs.UserReader {
	return idm.adminUser
}

// AddGroup creates a new group with the specified name.
// If the group already exists, the returned error is of type avfs.AlreadyExistsGroupError.
func (idm *MemIdm) AddGroup(groupName string) (avfs.GroupReader, error) {
	if !idm.IsValidNameFunc(groupName) {
		return nil, avfs.InvalidNameError(groupName)
	}

	if _, ok := idm.groupsByName[groupName]; ok {
		return nil, avfs.AlreadyExistsGroupError(groupName)
	}

	idm.maxGid++
	gid := idm.maxGid

	g := &MemGroup{idm: idm, name: groupName, gid: gid}
	idm.groupsByName[groupName] = g
	idm.groupsById[gid] = g

	return g, nil
}

// AddUser creates a new user with the specified userName and the specified primary group groupName.
// If the user already exists, the returned error is of type avfs.AlreadyExistsUserError.
func (idm *MemIdm) AddUser(userName, groupName string) (avfs.UserReader, error) {
	if !idm.IsValidNameFunc(userName) {
		return nil, avfs.InvalidNameError(userName)
	}

	if !idm.IsValidNameFunc(groupName) {
		return nil, avfs.InvalidNameError(groupName)
	}

	g, err := idm.lookupGroup(groupName)
	if err != nil {
		return nil, err
	}

	if _, ok := idm.usersByName[userName]; ok {
		return nil, avfs.AlreadyExistsUserError(userName)
	}

	idm.maxUid++
	uid := idm.maxUid
	gid := g.Gid()

	u := &MemUser{idm: idm, name: userName, uid: uid, gid: gid, groupsById: make(groupsById)}

	idm.usersByName[userName] = u
	idm.usersById[uid] = u

	err = u.addGroup(g)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// AddUserToGroup adds the user to the group.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
func (idm *MemIdm) AddUserToGroup(userName, groupName string) error {
	if !idm.IsValidNameFunc(userName) {
		return avfs.InvalidNameError(userName)
	}

	if !idm.IsValidNameFunc(groupName) {
		return avfs.InvalidNameError(groupName)
	}

	u, err := idm.lookupUser(userName)
	if err != nil {
		return err
	}

	g, err := idm.lookupGroup(groupName)
	if err != nil {
		return err
	}

	err = u.addGroup(g)
	if err != nil {
		return err
	}

	return nil
}

// DelGroup deletes an existing group with the specified name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *MemIdm) DelGroup(groupName string) error {
	if !idm.IsValidNameFunc(groupName) {
		return avfs.InvalidNameError(groupName)
	}

	g, ok := idm.groupsByName[groupName]
	if !ok {
		return avfs.UnknownGroupError(groupName)
	}

	delete(idm.groupsByName, g.name)
	delete(idm.groupsById, g.gid)

	return nil
}

// DelUser deletes an existing user with the specified name.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *MemIdm) DelUser(userName string) error {
	if !idm.IsValidNameFunc(userName) {
		return avfs.InvalidNameError(userName)
	}

	u, ok := idm.usersByName[userName]
	if !ok {
		return avfs.UnknownUserError(userName)
	}

	u.groupsById = nil

	delete(idm.usersByName, u.name)
	delete(idm.usersById, u.uid)

	return nil
}

// DelUserFromGroup removes the user from the group.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError
// or avfs.UnknownGroupError respectively.
func (idm *MemIdm) DelUserFromGroup(userName, groupName string) error {
	if !idm.IsValidNameFunc(userName) {
		return avfs.InvalidNameError(userName)
	}

	if !idm.IsValidNameFunc(groupName) {
		return avfs.InvalidNameError(groupName)
	}

	g, err := idm.lookupGroup(groupName)
	if err != nil {
		return err
	}

	u, err := idm.lookupUser(userName)
	if err != nil {
		return err
	}

	err = u.delGroup(g)
	if err != nil {
		return err
	}

	return nil
}

// LookupGroup looks up a group by name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *MemIdm) LookupGroup(groupName string) (avfs.GroupReader, error) {
	if !idm.IsValidNameFunc(groupName) {
		return nil, avfs.InvalidNameError(groupName)
	}

	return idm.lookupGroup(groupName)
}

func (idm *MemIdm) lookupGroup(groupName string) (*MemGroup, error) {
	g, ok := idm.groupsByName[groupName]
	if !ok {
		return nil, avfs.UnknownGroupError(groupName)
	}

	return g, nil
}

// LookupGroupId looks up a group by groupid.
// If the group is not found, the returned error is of type avfs.UnknownGroupIdError.
func (idm *MemIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	g, ok := idm.groupsById[gid]
	if !ok {
		return nil, avfs.UnknownGroupIdError(gid)
	}

	return g, nil
}

// LookupUser looks up a user by username.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *MemIdm) LookupUser(userName string) (avfs.UserReader, error) {
	if !idm.IsValidNameFunc(userName) {
		return nil, avfs.InvalidNameError(userName)
	}

	return idm.lookupUser(userName)
}

func (idm *MemIdm) lookupUser(userName string) (*MemUser, error) {
	u, ok := idm.usersByName[userName]
	if !ok {
		return nil, avfs.UnknownUserError(userName)
	}

	return u, nil
}

// LookupUserId looks up a user by userid.
// If the user is not found, the returned error is of type avfs.UnknownUserIdError.
func (idm *MemIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	u, ok := idm.usersById[uid]
	if !ok {
		return nil, avfs.UnknownUserIdError(uid)
	}

	return u, nil
}

// SetUserPrimaryGroup sets the primary group of a user to the specified group name.
// If the user or group is not found, the returned error is of type avfs.UnknownUserError or avfs.UnknownGroupError respectively.
// If the operation fails, the returned error is of type avfs.UnknownError.
func (idm *MemIdm) SetUserPrimaryGroup(userName, groupName string) error {
	if !idm.IsValidNameFunc(userName) {
		return avfs.InvalidNameError(userName)
	}

	if !idm.IsValidNameFunc(groupName) {
		return avfs.InvalidNameError(groupName)
	}

	g, err := idm.lookupGroup(groupName)
	if err != nil {
		return err
	}

	u, err := idm.lookupUser(userName)
	if err != nil {
		return err
	}

	u.gid = g.Gid()

	return nil
}

// MemUser

func (u *MemUser) addGroup(g *MemGroup) error {
	_, ok := u.groupsById[g.gid]
	if ok {
		return avfs.AlreadyExistsGroupError(g.name)
	}

	u.groupsById[g.gid] = g

	return nil
}

func (u *MemUser) delGroup(g *MemGroup) error {
	_, ok := u.groupsById[g.gid]
	if !ok {
		return avfs.UnknownGroupError(g.name)
	}

	delete(u.groupsById, g.gid)

	return nil
}

// Gid returns the primary group ID of the user.
func (u *MemUser) Gid() int {
	return u.gid
}

// Groups returns a slice of strings representing the group names that the user belongs to.
// If an error occurs while fetching the group names, it returns nil.
func (u *MemUser) Groups() []string {
	groups := make([]string, 0, len(u.groupsById))

	for _, g := range u.groupsById {
		groups = append(groups, g.Name())
	}

	return groups
}

// GroupsId returns a slice group IDs that the user belongs to.
// If an error occurs while fetching the group IDs, it returns nil.
func (u *MemUser) GroupsId() []int {
	gids := make([]int, 0, len(u.groupsById))

	for _, g := range u.groupsById {
		gids = append(gids, g.Gid())
	}

	return gids
}

// IsAdmin returns true if the user has administrator (root) privileges.
func (u *MemUser) IsAdmin() bool {
	return u.uid == 0 || u.IsInGroupId(0)
}

// IsInGroupId returns true if the user is in the specified group ID.
func (u *MemUser) IsInGroupId(gid int) bool {
	_, ok := u.groupsById[gid]

	return ok
}

// Name returns the username.
func (u *MemUser) Name() string {
	return u.name
}

// PrimaryGroup returns the primary group name of the user.
func (u *MemUser) PrimaryGroup() string {
	g, err := u.idm.LookupGroupId(u.gid)
	if err != nil {
		return ""
	}

	return g.Name()
}

// PrimaryGroupId returns the primary group ID of the OsUser.
// If an error occurs, it returns the maximum integer value.
func (u *MemUser) PrimaryGroupId() int {
	return u.gid
}

// Uid returns the user ID.
func (u *MemUser) Uid() int {
	return u.uid
}

// MemGroup

// Gid returns the group ID.
func (g *MemGroup) Gid() int {
	return g.gid
}

// Name returns the group name.
func (g *MemGroup) Name() string {
	return g.name
}
