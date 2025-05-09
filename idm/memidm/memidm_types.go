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

import "github.com/avfs/avfs"

const (
	// minUid is the minimum uid for a user.
	minUid = 1000

	// minGid is the minimum gid for a group.
	minGid = 1000
)

// MemIdm implements an in memory identity manager using the avfs.IdentityMgr interface.
type MemIdm struct {
	adminGroup      *MemGroup            // adminGroup is the Administrator Group.
	adminUser       *MemUser             // adminUser is the Administrator User.
	groupsByName    groupsByName         // groupsByName is the groups map by Name.
	groupsById      groupsById           // groupsById is the groups map by Id.
	usersByName     usersByName          // usersByName is the users map by Name.
	usersById       usersById            // usersById is users map by Id.
	IsValidNameFunc avfs.IsValidNameFunc // IsValidNameFunc is a function that checks if the input string is a valid username or group name.
	maxGid          int                  // maxGid is the current maximum Gid.
	maxUid          int                  // maxUid is the current maximum Uid.
	avfs.FeaturesFn                      // FeaturesFn provides features functions to a file system or an identity manager.
	avfs.OSTypeFn                        // OSTypeFn provides OS type functions to a file system or an identity manager.
}

// groupsByName is the map of groups by group name.
type groupsByName map[string]*MemGroup

// groupsById is the map of the groups by group id.
type groupsById map[int]*MemGroup

// usersByName is the map of the users by username.
type usersByName map[string]*MemUser

// usersById is the map of the users by user id.
type usersById map[int]*MemUser

// MemUser is the implementation of avfs.UserReader.
type MemUser struct {
	idm        *MemIdm // idm is the identity manager.
	groupsById         // groupsById is the groups map by Id.
	name       string  // name is the username.
	uid        int     // uid is the user id.
	gid        int     // gid is the primary group ID of the user.
}

// MemGroup is the implementation of avfs.GroupReader.
type MemGroup struct {
	idm  *MemIdm // idm is the identity manager.
	name string  // name is the group name.
	gid  int     // gid is the group id.
}

// Options defines the initialization options of MemIdm.
type Options struct {
	OSType avfs.OSType // OSType defines the operating system type.
}
