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
func (idm *MemIdm) AddGroup(name string) (avfs.GroupReader, error) {
	idm.grpMu.Lock()
	defer idm.grpMu.Unlock()

	if _, ok := idm.groupsByName[name]; ok {
		return nil, avfs.AlreadyExistsGroupError(name)
	}

	idm.maxGid++
	gid := idm.maxGid

	g := &MemGroup{name: name, gid: gid}
	idm.groupsByName[name] = g
	idm.groupsById[gid] = g

	return g, nil
}

// AddUser creates a new user with the specified userName and the specified primary group groupName.
// If the user already exists, the returned error is of type avfs.AlreadyExistsUserError.
func (idm *MemIdm) AddUser(name, groupName string) (avfs.UserReader, error) {
	g, err := idm.LookupGroup(groupName)
	if err != nil {
		return nil, err
	}

	idm.usrMu.Lock()
	defer idm.usrMu.Unlock()

	if _, ok := idm.usersByName[name]; ok {
		return nil, avfs.AlreadyExistsUserError(name)
	}

	idm.maxUid++
	uid := idm.maxUid

	u := &MemUser{
		name: name,
		uid:  uid,
		gid:  g.Gid(),
	}

	idm.usersByName[name] = u
	idm.usersById[uid] = u

	return u, nil
}

// DelGroup deletes an existing group with the specified name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *MemIdm) DelGroup(name string) error {
	idm.grpMu.Lock()
	defer idm.grpMu.Unlock()

	g, ok := idm.groupsByName[name]
	if !ok {
		return avfs.UnknownGroupError(name)
	}

	delete(idm.groupsByName, g.name)
	delete(idm.groupsById, g.gid)

	return nil
}

// DelUser deletes an existing user with the specified name.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *MemIdm) DelUser(name string) error {
	idm.usrMu.Lock()
	defer idm.usrMu.Unlock()

	u, ok := idm.usersByName[name]
	if !ok {
		return avfs.UnknownUserError(name)
	}

	delete(idm.usersByName, u.name)
	delete(idm.usersById, u.uid)

	return nil
}

// LookupGroup looks up a group by name.
// If the group is not found, the returned error is of type avfs.UnknownGroupError.
func (idm *MemIdm) LookupGroup(name string) (avfs.GroupReader, error) {
	idm.grpMu.RLock()
	defer idm.grpMu.RUnlock()

	g, ok := idm.groupsByName[name]
	if !ok {
		return nil, avfs.UnknownGroupError(name)
	}

	return g, nil
}

// LookupGroupId looks up a group by groupid.
// If the group is not found, the returned error is of type avfs.UnknownGroupIdError.
func (idm *MemIdm) LookupGroupId(gid int) (avfs.GroupReader, error) {
	idm.grpMu.RLock()
	defer idm.grpMu.RUnlock()

	g, ok := idm.groupsById[gid]
	if !ok {
		return nil, avfs.UnknownGroupIdError(gid)
	}

	return g, nil
}

// LookupUser looks up a user by username.
// If the user is not found, the returned error is of type avfs.UnknownUserError.
func (idm *MemIdm) LookupUser(name string) (avfs.UserReader, error) {
	idm.usrMu.RLock()
	defer idm.usrMu.RUnlock()

	u, ok := idm.usersByName[name]
	if !ok {
		return nil, avfs.UnknownUserError(name)
	}

	return u, nil
}

// LookupUserId looks up a user by userid.
// If the user is not found, the returned error is of type avfs.UnknownUserIdError.
func (idm *MemIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	idm.usrMu.RLock()
	defer idm.usrMu.RUnlock()

	u, ok := idm.usersById[uid]
	if !ok {
		return nil, avfs.UnknownUserIdError(uid)
	}

	return u, nil
}

// MemUser

// Name returns the user name.
func (u *MemUser) Name() string {
	return u.name
}

// Gid returns the primary group ID of the user.
func (u *MemUser) Gid() int {
	return u.gid
}

// IsAdmin returns true if the user has administrator (root) privileges.
func (u *MemUser) IsAdmin() bool {
	return u.uid == 0 || u.gid == 0
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
