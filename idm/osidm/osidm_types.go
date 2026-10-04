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

package osidm

import "github.com/avfs/avfs"

// OsIdm implements a rudimentary identity manager using the avfs.IdmMgr interface.
type OsIdm struct {
	avfs.IdmMixin
	adminGroup         *OsGroup // Administrator group.
	adminUser          *OsUser  // Administrator user.
	avfs.FeaturesMixin          // FeaturesMixin is an embeddable default implementation of the Featurer interface.
}

// OsGroup is the implementation of avfs.GroupReader.
type OsGroup struct {
	name string // name is the name of the group.
	gid  int    // gid represents the group ID of the OsGroup.
}

// OsUser is the implementation of avfs.UserReader.
type OsUser struct {
	name string // name is the name of the user.
	uid  int    // uid represents the user ID of the OsUser.
	gid  int    // gid represents the primary group ID of the OsUser.
}
