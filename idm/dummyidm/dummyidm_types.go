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

package dummyidm

import "github.com/avfs/avfs"

var (
	// NotImplementedIdm is the default identity manager for all file systems.
	NotImplementedIdm = &DummyIdm{} //nolint:gochecknoglobals // Used as default Idm for other file systems.

	// RootUser represents a root user.
	RootUser = &User{name: avfs.UsrRoot, uid: 0, gid: 0} //nolint:gochecknoglobals // Used as user for other file systems.

	// NotImplementedUser represents a not implemented invalid user.
	NotImplementedUser = &User{name: avfs.NotImplemented, uid: -1, gid: -1} //nolint:gochecknoglobals // Used as user for other file systems.
)

// DummyIdm represent a non implementedy identity manager using the avfs.IdentityMgr interface.
type DummyIdm struct {
}

// User is the implementation of avfs.UserReader.
type User struct {
	name string
	uid  int
	gid  int
}

// Group is the implementation of avfs.GroupReader.
type Group struct {
	name string
	gid  int
}
