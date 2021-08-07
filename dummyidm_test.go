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

//go:build !datarace
// +build !datarace

package avfs_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/test"
)

var (
	// avfs.DummyIdm implements avfs.IdentityMgr interface.
	_ avfs.IdentityMgr = &avfs.DummyIdm{}

	// avfs.User struct implements avfs.UserReader interface.
	_ avfs.UserReader = &avfs.DummyUser{}

	// avfs.Group struct implements avfs.GroupReader interface.
	_ avfs.GroupReader = &avfs.DummyGroup{}
)

func TestDummyIdm(t *testing.T) {
	idm := avfs.NewDummyIdm()

	t.Logf("Idm = %v", idm.Type())

	sIdm := test.NewSuiteIdm(t, idm)
	sIdm.TestAll(t)
}

func TestDummyIdmFeatures(t *testing.T) {
	idm := avfs.NewDummyIdm()

	if idm.Features() != 0 {
		t.Errorf("Features : want Features to be 0, got %d", idm.Features())
	}
}
