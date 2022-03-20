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

package test

import (
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
)

// SuiteIdm is a test suite for an identity manager.
type SuiteIdm struct {
	idm     avfs.IdentityMgr   // idm is the identity manager to be tested.
	Groups  []avfs.GroupReader // Groups contains the test groups created with the identity manager.
	Users   []avfs.UserReader  // Users contains the test users created with the identity manager.
	utils   avfs.Utils         // utils regroups common functions used by emulated file systems.
	canTest bool               // canTest is true when the identity manager can be tested.
}

// NewSuiteIdm returns a new test suite for an identity manager.
func NewSuiteIdm(t *testing.T, idm avfs.IdentityMgr) *SuiteIdm {
	sIdm := &SuiteIdm{
		idm:   idm,
		utils: avfs.OSUtils,
	}

	defer func() {
		t.Logf("Info Idm = %s, can test permissions = %t", sIdm.Type(), sIdm.canTest)
	}()

	sIdm.canTest = idm.HasFeature(avfs.FeatIdentityMgr) && !idm.HasFeature(avfs.FeatReadOnlyIdm)
	if !sIdm.canTest {
		return sIdm
	}

	sIdm.Groups = CreateGroups(t, idm, "")
	sIdm.Users = CreateUsers(t, idm, "")

	return sIdm
}

// Type returns the type of the identity manager.
func (sIdm *SuiteIdm) Type() string {
	return sIdm.idm.Type()
}

// TestAll run all identity manager tests.
func (sIdm *SuiteIdm) TestAll(t *testing.T) {
	sIdm.TestAdmin(t)
	sIdm.TestGroupAddDel(t)
	sIdm.TestUserAddDel(t)
	sIdm.TestLookup(t)
}

// GroupInfo contains information to create a test group.
type GroupInfo struct {
	Name string
}

const (
	grpTest  = "grpTest"  // grpTest is the default group of the default test user UsrTest.
	grpOther = "grpOther" // grpOther is the group to test users who are not members of grpTest.
	grpEmpty = "grpEmpty" // grpEmpty is a group without users.
)

// GroupInfos returns a GroupInfo slice describing the test groups.
func GroupInfos() []*GroupInfo {
	gis := []*GroupInfo{
		{Name: grpTest},
		{Name: grpOther},
		{Name: grpEmpty},
	}

	return gis
}

// CreateGroups creates and returns test groups with a suffix appended to each group.
// Errors are ignored if the group already exists.
func CreateGroups(tb testing.TB, idm avfs.IdentityMgr, suffix string) (groups []avfs.GroupReader) {
	for _, group := range GroupInfos() {
		groupName := group.Name + suffix

		g, err := idm.GroupAdd(groupName)
		if err != nil && err != avfs.AlreadyExistsGroupError(groupName) {
			tb.Fatalf("GroupAdd %s : want error to be nil, got %v", groupName, err)
		}

		groups = append(groups, g)
	}

	return
}

// UserInfo contains information to create a test user.
type UserInfo struct {
	Name      string
	GroupName string
}

const (
	UsrTest = "UsrTest" // UsrTest is used to test user access rights.
	UsrGrp  = "UsrGrp"  // UsrGrp is a member of the group GrpTest used to test default group access rights.
	UsrOth  = "UsrOth"  // UsrOth is a member of the group GrpOth used to test non-members access rights.
)

// UserInfos returns a UserInfo slice describing the test users.
func UserInfos() []*UserInfo {
	uis := []*UserInfo{
		{Name: UsrTest, GroupName: grpTest},
		{Name: UsrGrp, GroupName: grpTest},
		{Name: UsrOth, GroupName: grpOther},
	}

	return uis
}

// CreateUsers creates and returns test users with a suffix appended to each user.
// Errors are ignored if the user or his home directory already exists.
func CreateUsers(tb testing.TB, idm avfs.IdentityMgr, suffix string) (users []avfs.UserReader) {
	for _, ui := range UserInfos() {
		userName := ui.Name + suffix
		groupName := ui.GroupName + suffix

		u, err := idm.UserAdd(userName, groupName)
		if err != nil {
			switch e := err.(type) {
			case *fs.PathError:
				if e.Op != "mkdir" || e.Err != avfs.ErrFileExists {
					tb.Fatalf("UserAdd %s : want Mkdir error, got %v", userName, err)
				}
			default:
				if err != avfs.AlreadyExistsUserError(userName) {
					tb.Fatalf("UserAdd %s : want error to be nil, got %v", userName, err)
				}
			}

			if u == nil {
				u, err = idm.LookupUser(userName)
				if err != nil {
					tb.Fatalf("LookupUser %s : want error to be nil, got %v", userName, err)
				}
			}
		}

		users = append(users, u)
	}

	return
}
