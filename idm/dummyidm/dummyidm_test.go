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

// +build !datarace

package dummyidm_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/dummyidm"
	"github.com/avfs/avfs/test"
)

var (
	// DummyIdm implements avfs.IdentityMgr interface.
	_ avfs.IdentityMgr = &dummyidm.DummyIdm{}

	// DummyIdm implements avfs.UserConnecter interface.
	_ avfs.UserConnecter = &dummyidm.DummyIdm{}

	// DummyIdm.User struct implements avfs.UserReader interface.
	_ avfs.UserReader = &dummyidm.User{}

	// DummyIdm.Group struct implements avfs.GroupReader interface.
	_ avfs.GroupReader = &dummyidm.Group{}
)

func TestDummyIdm(t *testing.T) {
	idm := dummyidm.New()

	t.Logf("Idm = %v", idm.Type())

	sidm := test.NewSuiteIdm(t, idm)
	sidm.SuitePermDenied()

	u := idm.CurrentUser()

	gid := u.Gid()
	if gid != -1 {
		t.Errorf("Gid : want Gid to be -1, got %d", gid)
	}

	isRoot := u.IsRoot()
	if isRoot {
		t.Errorf("IsRoot : want isRoot to be false, got %t", isRoot)
	}

	name := u.Name()
	if name != avfs.NotImplemented {
		t.Errorf("Name : want name to be empty, got %s", name)
	}

	uid := u.Uid()
	if uid != -1 {
		t.Errorf("Uid : want Uid to be -1, got %d", uid)
	}

	g := dummyidm.Group{}

	gid = g.Gid()
	if gid != 0 {
		t.Errorf("Gid : want Gid to be 0, got %d", gid)
	}

	name = g.Name()
	if name != "" {
		t.Errorf("Name : want name to be empty, got %s", name)
	}
}
