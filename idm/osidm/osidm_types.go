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

// OsIdm implements a rudimentary identity manager using the avfs.IdentityMgr interface.
type OsIdm struct {
	initUser avfs.UserReader
	feature  avfs.Feature
}

// OsGroup is the implementation of avfs.GroupReader.
type OsGroup struct {
	name string
	gid  int
}

// OsUser is the implementation of avfs.UserReader.
type OsUser struct {
	name string
	uid  int
	gid  int
}
