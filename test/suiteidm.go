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
	"os/exec"
	"testing"

	"github.com/avfs/avfs"
)

// SuiteIdm is a test suite for an identity manager.
type SuiteIdm struct {
	t   *testing.T
	idm avfs.IdentityMgr

	// hasIdm is true when the identity manager has the feature avfs.FeatIdentityMgr.
	hasIdm bool
	// hasUser is true when the identity manager implements avfs.UserConnecter.
	hasUser bool
	// hasRoot is true when the current user is root.
	hasRoot bool
}

// NewSuiteIdm returns a new test suite for an identity manager.
func NewSuiteIdm(t *testing.T, idm avfs.IdentityMgr) *SuiteIdm {
	sIdm := &SuiteIdm{
		t:       t,
		idm:     idm,
		hasIdm:  false,
		hasUser: false,
		hasRoot: false,
	}

	defer func() {
		t.Logf("Info Idm = %s, Idm = %t, User = %t, Root = %t",
			sIdm.Type(), sIdm.hasIdm, sIdm.hasUser, sIdm.hasRoot)
	}()

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		return sIdm
	}

	sIdm.hasIdm = true

	uc, ok := idm.(avfs.UserConnecter)
	if ok {
		u := uc.CurrentUser()
		if u == nil {
			return sIdm
		}

		sIdm.hasUser = true

		if !u.IsRoot() {
			return sIdm
		}

		sIdm.hasRoot = true
	}

	CreateGroups(t, idm, "")
	CreateUsers(t, idm, "")

	return sIdm
}

// Type returns the type of the identity manager.
func (sIdm *SuiteIdm) Type() string {
	return sIdm.idm.Type()
}

// Group contains the data to test groups.
type Group struct {
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

// GetGroups returns the test groups.
func GetGroups() []*Group {
	groups := []*Group{
		{Name: grpTest},
		{Name: grpOther},
		{Name: grpEmpty},
	}

	return groups
}

// CreateGroups creates test groups with a suffix appended to each group.
// Errors are ignored if the group already exists or the function GroupAdd is not implemented.
func CreateGroups(t *testing.T, idm avfs.IdentityMgr, suffix string) []*Group {
	groups := GetGroups()
	for _, group := range groups {
		groupName := group.Name + suffix

		_, err := idm.GroupAdd(groupName)
		if err != nil &&
			err != exec.ErrNotFound &&
			err != avfs.ErrPermDenied &&
			err != avfs.AlreadyExistsGroupError(groupName) {
			t.Fatalf("GroupAdd %s : want error to be nil, got %v", groupName, err)
		}
	}

	return groups
}

// User contains the data to test users.
type User struct {
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

// GetUsers returns the test users.
func GetUsers() []*User {
	users := []*User{
		{Name: UsrTest, GroupName: grpTest},
		{Name: UsrGrp, GroupName: grpTest},
		{Name: UsrOth, GroupName: grpOther},
	}

	return users
}

// CreateUsers creates test users with a suffix appended to each user.
// Errors are ignored if the user already exists or the function UserAdd is not implemented.
func CreateUsers(t *testing.T, idm avfs.IdentityMgr, suffix string) []*User {
	users := GetUsers()
	for _, user := range users {
		userName := user.Name + suffix
		groupName := user.GroupName + suffix

		_, err := idm.UserAdd(userName, groupName)
		if err != nil &&
			err != exec.ErrNotFound &&
			err != avfs.ErrPermDenied &&
			err != avfs.AlreadyExistsUserError(userName) {
			t.Fatalf("UserAdd %s : want error to be nil, got %v", userName, err)
		}
	}

	return users
}
