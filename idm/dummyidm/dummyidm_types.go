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

const maxInt = int(^uint(0) >> 1)

var (
	// NotImplementedIdm is the default identity manager for all file systems.
	NotImplementedIdm = &DummyIdm{} //nolint:gochecknoglobals // Used as default Idm for other file systems.

	// RootUser represents a root user.
	RootUser = &User{ //nolint:gochecknoglobals // Used as root user for other file systems.
		name: avfs.UsrRoot,
		uid:  0,
		gid:  0,
	}

	// NotImplementedUser represents a not implemented invalid user.
	NotImplementedUser = &User{ //nolint:gochecknoglobals // Used as not implemented user for other file systems.
		name: avfs.NotImplemented,
		uid:  maxInt,
		gid:  maxInt,
	}
)

// DummyIdm represent a non implemented identity manager using the avfs.IdentityMgr interface.
type DummyIdm struct{}

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
