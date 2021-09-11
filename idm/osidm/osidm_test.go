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

package osidm_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
)

var (
	// OsIdm implements avfs.IdentityMgr interface.
	_ avfs.IdentityMgr = &osidm.OsIdm{}

	// OsUser implements avfs.UserReader interface.
	_ avfs.UserReader = &osidm.OsUser{}

	// OsGroup implements avfs.GroupReader interface.
	_ avfs.GroupReader = &osidm.OsGroup{}
)

// TestOsIdmAll run all tests.
func TestOsIdmAll(t *testing.T) {
	idm := osidm.New()

	sidm := test.NewSuiteIdm(t, idm)
	sidm.TestAll(t)
}

func TestOsIdmCfg(t *testing.T) {
	var wantFeat avfs.Feature

	switch avfs.CurrentOSType() {
	case avfs.OsWindows:
		wantFeat = 0
	default:
		wantFeat = avfs.FeatIdentityMgr
	}

	idm := osidm.New()
	if idm.Features() != wantFeat {
		t.Errorf("want feature to be %v, got %v", wantFeat, idm.Features())
	}

	if idm.HasFeature(avfs.FeatIdentityMgr) {
		if osidm.CurrentUser() == idm.CurrentUser() {
			t.Errorf("want current user (%v) = OS current user (%v)", osidm.CurrentUser(), idm.CurrentUser())
		}
	}
}
