//
//  Copyright 2023 The AVFS authors
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

// NotImplementedIdm is the default identity manager for all file systems.
var NotImplementedIdm = New() //nolint:gochecknoglobals // Used as default Idm for other file systems.

// DummyIdm represent a non implemented identity manager using the avfs.IdentityMgr interface.
type DummyIdm struct {
	adminGroup avfs.GroupReader
	adminUser  avfs.UserReader
}

// DummyUser is the implementation of avfs.UserReader.
type DummyUser struct {
	name string
	uid  int
	gid  int
}

// DummyGroup is the implementation of avfs.GroupReader.
type DummyGroup struct {
	name string
	gid  int
}
