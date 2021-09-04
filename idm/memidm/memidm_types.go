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
	"sync"
)

const (
	// minUid is the minimum uid for a user.
	minUid = 1000

	// minGid is the minimum gid for a group.
	minGid = 1000
)

// MemIdm implements an in memory identity manager using the avfs.IdentityMgr interface.
type MemIdm struct {
	groupsByName   groupsByName
	groupsById     groupsById
	usersByName    usersByName
	usersById      usersById
	maxGid, maxUid int
	grpMu, usrMu   sync.RWMutex
}

// groupsByName is the map of groups by group name.
type groupsByName map[string]*MemGroup

// groupsById is the map of the groups by group id.
type groupsById map[int]*MemGroup

// usersByName is the map of the users by user name.
type usersByName map[string]*MemUser

// usersById is the map of the users by user id.
type usersById map[int]*MemUser

// MemUser is the implementation of avfs.UserReader.
type MemUser struct {
	name string
	uid  int
	gid  int
}

// MemGroup is the implementation of avfs.GroupReader.
type MemGroup struct {
	name string
	gid  int
}
