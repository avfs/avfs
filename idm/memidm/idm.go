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

package memidm

import (
	"github.com/avfs/avfs"
)

// New creates a new identity manager.
func New() *MemIdm {
	groupRoot := &Group{
		name: avfs.UsrRoot,
		gid:  0,
	}

	userRoot := &User{
		name: avfs.UsrRoot,
		uid:  0,
		gid:  0,
	}

	idm := &MemIdm{
		groupsByName: make(groupsByName),
		groupsById:   make(groupsById),
		usersByName:  make(usersByName),
		usersById:    make(usersById),
		maxGid:       minGid,
		maxUid:       minUid,
	}

	idm.groupsById[0] = groupRoot
	idm.groupsByName[avfs.UsrRoot] = groupRoot
	idm.usersById[0] = userRoot
	idm.usersByName[avfs.UsrRoot] = userRoot

	return idm
}

// Type returns the type of the fileSystem or Identity manager.
func (idm *MemIdm) Type() string {
	return "MemIdm"
}

// GroupAdd adds a new group.
func (idm *MemIdm) GroupAdd(name string) (avfs.GroupReader, error) {
	idm.grpMu.Lock()
	defer idm.grpMu.Unlock()

	if _, ok := idm.groupsByName[name]; ok {
		return nil, avfs.AlreadyExistsGroupError(name)
	}

	idm.maxGid++
	gid := idm.maxGid

	g := &Group{name: name, gid: gid}
	idm.groupsByName[name] = g
	idm.groupsById[gid] = g

	return g, nil
}

// GroupDel deletes an existing group.
func (idm *MemIdm) GroupDel(name string) error {
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

// LookupGroup looks up a group by name.
// If the group cannot be found, the returned error is of type UnknownGroupError.
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
// If the group cannot be found, the returned error is of type UnknownGroupIdError.
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
// If the user cannot be found, the returned error is of type UnknownUserError.
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
// If the user cannot be found, the returned error is of type UnknownUserIdError.
func (idm *MemIdm) LookupUserId(uid int) (avfs.UserReader, error) {
	idm.usrMu.RLock()
	defer idm.usrMu.RUnlock()

	u, ok := idm.usersById[uid]
	if !ok {
		return nil, avfs.UnknownUserIdError(uid)
	}

	return u, nil
}

// UserAdd adds a new user.
func (idm *MemIdm) UserAdd(name, groupName string) (avfs.UserReader, error) {
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

	u := &User{
		name: name,
		uid:  uid,
		gid:  g.Gid(),
	}

	idm.usersByName[name] = u
	idm.usersById[uid] = u

	return u, nil
}

// UserDel deletes an existing group.
func (idm *MemIdm) UserDel(name string) error {
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

// User

// Name returns the user name.
func (u *User) Name() string {
	return u.name
}

// Gid returns the primary group ID of the user.
func (u *User) Gid() int {
	return u.gid
}

// IsRoot returns true if the user has root privileges.
func (u *User) IsRoot() bool {
	return u.uid == 0 || u.gid == 0
}

// Uid returns the user ID.
func (u *User) Uid() int {
	return u.uid
}

// Group

// Gid returns the group ID.
func (g *Group) Gid() int {
	return g.gid
}

// Name returns the group name.
func (g *Group) Name() string {
	return g.name
}
