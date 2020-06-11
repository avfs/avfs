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

	// User implements avfs.UserReader interface.
	_ avfs.UserReader = &osidm.User{}

	// Group implements avfs.GroupReader interface.
	_ avfs.GroupReader = &osidm.Group{}
)

// TestOsIdmAll run all tests.
func TestOsIdmAll(t *testing.T) {
	idm := osidm.New()
	ci := test.NewConfigIdm(t, idm)
	ci.SuiteAll()
}
