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

package dummyfs

import (
	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/dummyidm"
)

// CurrentUser returns the current User.
func (fs *DummyFs) CurrentUser() avfs.UserReader {
	return dummyidm.NotImplementedUser
}

// GroupAdd adds a new Group.
func (fs *DummyFs) GroupAdd(name string) (avfs.GroupReader, error) {
	return nil, avfs.ErrNotImplemented
}

// GroupDel deletes an existing Group.
func (fs *DummyFs) GroupDel(name string) error {
	return avfs.ErrNotImplemented
}

// LookupGroup looks up a Group by name. If the Group cannot be found, the
// returned error is of type UnknownGroupError.
func (fs *DummyFs) LookupGroup(name string) (avfs.GroupReader, error) {
	return nil, avfs.ErrNotImplemented
}

// LookupGroupId looks up a Group by groupid. If the Group cannot be found, the
// returned error is of type UnknownGroupIdError.
func (fs *DummyFs) LookupGroupId(gid int) (avfs.GroupReader, error) {
	return nil, avfs.ErrNotImplemented
}

// LookupUser looks up a User by username. If the User cannot be found, the
// returned error is of type UnknownUserError.
func (fs *DummyFs) LookupUser(name string) (avfs.UserReader, error) {
	return nil, avfs.ErrNotImplemented
}

// LookupUserId looks up a User by userid. If the User cannot be found, the
// returned error is of type UnknownUserIdError.
func (fs *DummyFs) LookupUserId(uid int) (avfs.UserReader, error) {
	return nil, avfs.ErrNotImplemented
}

// User sets the current User.
func (fs *DummyFs) User(name string) (avfs.UserReader, error) {
	return nil, avfs.ErrNotImplemented
}

// UserAdd adds a new User.
func (fs *DummyFs) UserAdd(name, groupName string) (avfs.UserReader, error) {
	return nil, avfs.ErrNotImplemented
}

// UserDel deletes an existing Group.
func (fs *DummyFs) UserDel(name string) error {
	return avfs.ErrNotImplemented
}
