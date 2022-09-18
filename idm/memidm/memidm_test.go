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

	// MemUser implements avfs.UserReader interface.
	_ avfs.UserReader = &memidm.MemUser{}

	// MemGroup implements avfs.GroupReader interface.
	_ avfs.GroupReader = &memidm.MemGroup{}
)

// TestMemIdmAll run all tests.
func TestMemIdmAll(t *testing.T) {
	idm := memidm.New()
	sIdm := test.NewSuiteIdm(t, idm)
	sIdm.TestAll(t)
}

// TestMemIdmAllOSType run all tests with the current OS.
func TestMemIdmAllOSType(t *testing.T) {
	idm := memidm.NewWithOptions(&memidm.Options{OSType: avfs.CurrentOSType()})
	sIdm := test.NewSuiteIdm(t, idm)
	sIdm.TestAll(t)
}

func TestMemIdmFeatures(t *testing.T) {
	idm := memidm.New()

	if idm.Features() != avfs.FeatIdentityMgr {
		t.Errorf("Features : want Features to be %d, got %d", avfs.FeatIdentityMgr, idm.Features())
	}
}
