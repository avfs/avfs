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

package memidm_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/memidm"
	"github.com/avfs/avfs/test"
)

var (
	// MemIdm implements avfs.IdentityMgr interface.
	_ avfs.IdentityMgr = &memidm.MemIdm{}

	// User implements avfs.UserReader interface.
	_ avfs.UserReader = &memidm.User{}

	// Group implements avfs.GroupReader interface.
	_ avfs.GroupReader = &memidm.Group{}
)

// TestMemIdmAll run all tests.
func TestMemIdmAll(t *testing.T) {
	idm := memidm.New()
	sidm := test.NewSuiteIdm(t, idm)
	sidm.All()
}

func TestMemIdmFeatures(t *testing.T) {
	idm := memidm.New()

	if idm.Features() != avfs.FeatIdentityMgr {
		t.Errorf("Features : want Features to be %d, got %d", avfs.FeatIdentityMgr, idm.Features())
	}
}
