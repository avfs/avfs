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
	"os"
	"testing"

	"github.com/avfs/avfs"
)

// SuiteIdm is a test suite for an identity manager.
type SuiteIdm struct {
	// idm is the identity manager to be tested.
	idm avfs.IdentityMgr

	// Groups contains the test groups created with the identity manager.
	Groups []avfs.GroupReader

	// Users contains the test users created with the identity manager.
	Users []avfs.UserReader

	// uc is the implementation of avfs.UserConnecter.
	uc avfs.UserConnecter

	// canTest is true when the identity manager can be tested..
	canTest bool
}

// NewSuiteIdm returns a new test suite for an identity manager.
func NewSuiteIdm(t *testing.T, idm avfs.IdentityMgr) *SuiteIdm {
	sIdm := &SuiteIdm{idm: idm}

	defer func() {
		t.Logf("Info Idm = %s, Can test = %t", sIdm.Type(), sIdm.canTest)
	}()

	uc, ok := idm.(avfs.UserConnecter)
	if ok {
		sIdm.uc = uc

		u := uc.CurrentUser()
		if u == nil {
			return sIdm
		}

		if !u.IsRoot() {
			return sIdm
		}
	}

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		return sIdm
	}

	sIdm.canTest = true

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
	sIdm.TestCurrentUser(t)
	sIdm.TestGroupAddDel(t)
	sIdm.TestUserAddDel(t)
	sIdm.TestLookup(t)
	sIdm.TestUser(t)
	sIdm.TestUserDenied(t)
	sIdm.TestPermDenied(t)
}

// GroupInfo contains information to create a test group.
type GroupInfo struct {
	Name string
}

const (
	// grpTest is the default group of the default test user UsrTest.
	grpTest = "grpTest"

	// grpOther is the group to test users who are not members of grpTest.
	grpOther = "grpOther"

	// grpEmpty is a group without users.
	grpEmpty = "grpEmpty"
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
	// UsrTest is used to test user access rights.
	UsrTest = "UsrTest"

	// UsrGrp is a member of the group GrpTest used to test default group access rights.
	UsrGrp = "UsrGrp"

	// UsrOth is a member of the group GrpOth used to test non members access rights.
	UsrOth = "UsrOth"
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
			case *os.PathError:
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
