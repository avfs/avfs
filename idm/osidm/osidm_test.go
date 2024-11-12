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

package osidm_test

import (
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/idm/osidm"
	"github.com/avfs/avfs/test"
)

var (
	// OsGroup implements avfs.GroupReader interface.
	_ avfs.GroupReader = &osidm.OsGroup{}

	// OsIdm implements avfs.IdentityMgr interface.
	_ avfs.IdentityMgr = &osidm.OsIdm{}

	// OsUser implements avfs.UserReader interface.
	_ avfs.UserReader = &osidm.OsUser{}
)

func TestOsIdmAll(t *testing.T) {
	idm := osidm.New()

	ts := test.NewSuiteIdm(t, idm)
	ts.TestIdmAll(t)
}

func TestOsIdmCfg(t *testing.T) {
	idm := osidm.New()

	switch avfs.CurrentOSType() {
	case avfs.OsWindows:
		if idm.Features() != 0 {
			t.Errorf("want feature to be %v, got %v", 0, idm.Features())
		}
	default:
		if idm.Features()&avfs.FeatIdentityMgr == 0 {
			t.Errorf("want feature to be %v, got %v", avfs.FeatIdentityMgr, idm.Features())
		}
	}
}
