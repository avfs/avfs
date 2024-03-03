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

//go:build !avfs_race

package avfs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
)

var (
	// Tests that avfs.DummyIdm implements avfs.IdentityMgr interface.
	_ avfs.IdentityMgr = &avfs.DummyIdm{}

	// Tests that avfs.DummyUser struct implements avfs.UserReader interface.
	_ avfs.UserReader = &avfs.DummyUser{}

	// Tests that avfs.DummyGroup struct implements avfs.GroupReader interface.
	_ avfs.GroupReader = &avfs.DummyGroup{}
)

func TestDummyIdm(t *testing.T) {
	idm := avfs.NewDummyIdm()

	ts := test.NewSuiteIdm(t, idm)
	ts.TestIdmAll(t)
}

func TestDummyIdmFeatures(t *testing.T) {
	idm := avfs.NewDummyIdm()

	if idm.Features() != 0 {
		t.Errorf("Features : want Features to be 0, got %d", idm.Features())
	}
}

func TestNewGroup(t *testing.T) {
	const (
		groupName = "aGroup"
		gid       = 1
	)

	aGroup := avfs.NewGroup(groupName, gid)
	if aGroup.Name() != groupName {
		t.Errorf("want group to be %s, got %s", groupName, aGroup.Name())
	}

	if aGroup.Gid() != gid {
		t.Errorf("want group id to be %d, got %d", gid, aGroup.Gid())
	}
}

func TestNewUser(t *testing.T) {
	const (
		userName = "aUser"
		uid      = 1
		gid      = 2
	)

	aUser := avfs.NewUser(userName, uid, gid)
	if aUser.Name() != userName {
		t.Errorf("want user to be %s, got %s", userName, aUser.Name())
	}

	if aUser.Gid() != gid {
		t.Errorf("want group id to be %d, got %d", gid, aUser.Gid())
	}

	if aUser.Uid() != uid {
		t.Errorf("want user id to be %d, got %d", uid, aUser.Uid())
	}
}
