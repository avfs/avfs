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
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"testing"

	"github.com/avfs/avfs"
)

var ErrInvalidName = avfs.InvalidNameError("")

// TestIdmAll runs all identity manager tests.
func (ts *Suite) TestIdmAll(t *testing.T) {
	defer ts.setInitUser(t)

	ts.TestIdmAdmin(t)
	ts.TestIdmGroup(t)
	ts.TestIdmLookup(t)
	ts.TestIdmUser(t)
	ts.TestIdmUserGroups(t)
}

// TestIdmAdmin tests AdminGroup and AdminUser.
func (ts *Suite) TestIdmAdmin(t *testing.T) {
	idm := ts.idm

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		wantGroup, wantUser := avfs.DefaultName, avfs.DefaultName

		ag := idm.AdminGroup()
		if ag.Name() != wantGroup {
			t.Errorf("AdminGroup : want name to be %s, got %s", wantGroup, ag.Name())
		}

		if ag.Gid() != math.MaxInt {
			t.Errorf("AdminGroup : want Gid to be %d, got %d", math.MaxInt, ag.Gid())
		}

		au := idm.AdminUser()
		if au.Name() != wantUser {
			t.Errorf("AdminUser : want name to be %s, got %s", wantUser, au.Name())
		}

		if au.Uid() != math.MaxInt {
			t.Errorf("AdminUser : want Uid to be %d, got %d", math.MaxInt, au.Uid())
		}

		if au.Gid() != math.MaxInt {
			t.Errorf("AdminUser : want Gid to be %d, got %d", math.MaxInt, au.Gid())
		}

		if au.IsAdmin() {
			t.Errorf("AdminUser : want IsAdmin to be false, got true")
		}

		return
	}

	t.Run("AdminGroupUserName", func(t *testing.T) {
		wantGroupName := avfs.AdminGroupName(avfs.CurrentOSType())
		ag := idm.AdminGroup()

		if ag.Name() != wantGroupName {
			t.Errorf("AdminGroup : want name to be %s, got %s", wantGroupName, ag.Name())
		}

		wantUserName := avfs.AdminUserName(avfs.CurrentOSType())
		au := idm.AdminUser()

		if au.Name() != wantUserName {
			t.Errorf("AdminUser : want name to be %s, got %s", wantUserName, au.Name())
		}
	})
}

// TestIdmGroup tests group related functions.
func (ts *Suite) TestIdmGroup(t *testing.T) {
	idm := ts.idm
	suffix := fmt.Sprintf("GroupAddDel%x", rand.Uint32())

	if !idm.HasFeature(avfs.FeatIdentityMgr) || idm.HasFeature(avfs.FeatReadOnlyIdm) {
		groupName := "AGroup" + suffix

		_, err := idm.AddGroup(groupName)
		if err != avfs.ErrPermDenied {
			t.Errorf("AddGroup : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = idm.DelGroup(groupName)
		if err != avfs.ErrPermDenied {
			t.Errorf("DelGroup : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	gis := GroupInfos()
	prevGid := 0

	t.Run("DelGroupNotExists", func(t *testing.T) {
		for _, gi := range gis {
			groupName := gi.Name + suffix
			wantErr := avfs.UnknownGroupError(groupName)

			err := idm.DelGroup(groupName)
			if err != wantErr {
				t.Errorf("DelGroup %s : want error to be %v, got %v", groupName, wantErr, err)
			}
		}
	})

	t.Run("AddGroup", func(t *testing.T) {
		for _, gi := range gis {
			groupName := gi.Name + suffix
			wantGroupErr := avfs.UnknownGroupError(groupName)

			_, err := idm.LookupGroup(groupName)
			if err != wantGroupErr {
				t.Errorf("LookupGroupName %s : want error to be %v, got %v", groupName, wantGroupErr, err)
			}

			g, err := idm.AddGroup(groupName)
			if !AssertNoError(t, err, "AddGroup %s", groupName) {
				continue
			}

			if g.Name() != groupName {
				t.Errorf("AddGroup %s : want Name to be %s, got %s", groupName, groupName, g.Name())
			}

			if g.Gid() <= prevGid {
				t.Errorf("AddGroup %s : want gid to be > %d, got %d", groupName, prevGid, g.Gid())
			} else {
				prevGid = g.Gid()
			}

			_, err = idm.LookupGroup(groupName)
			RequireNoError(t, err, "LookupGroup %s", groupName)

			_, err = idm.LookupGroupId(g.Gid())
			RequireNoError(t, err, "LookupGroupId %s", groupName)
		}
	})

	t.Run("AddGroupInvalid", func(t *testing.T) {
		_, err := idm.AddGroup("")
		if err != ErrInvalidName {
			t.Errorf("AddGroup '' : want error to be %v, got %v", ErrInvalidName, err)
		}
	})

	t.Run("AddGroupExists", func(t *testing.T) {
		for _, gi := range gis {
			groupName := gi.Name + suffix
			wantErr := avfs.AlreadyExistsGroupError(groupName)

			_, err := idm.AddGroup(groupName)
			if err != wantErr {
				t.Errorf("AddGroup %s : want error to be %v, got %v", groupName, wantErr, err)
			}
		}
	})

	t.Run("DelGroupInvalid", func(t *testing.T) {
		err := idm.DelGroup("")
		if err != ErrInvalidName {
			t.Errorf("DelGroup '' : want error to be %v, got %v", ErrInvalidName, err)
		}
	})

	t.Run("DelGroup", func(t *testing.T) {
		for _, gi := range gis {
			groupName := gi.Name + suffix

			g, err := idm.LookupGroup(groupName)
			RequireNoError(t, err, "LookupGroup %s", groupName)

			err = idm.DelGroup(groupName)
			RequireNoError(t, err, "DelGroup %s", groupName)

			_, err = idm.LookupGroup(g.Name())
			wantGroupErr := avfs.UnknownGroupError(groupName)

			if err != wantGroupErr {
				t.Errorf("LookupGroup %s : want error to be %v, got %v", groupName, wantGroupErr, err)
			}

			_, err = idm.LookupGroupId(g.Gid())
			wantGroupIdErr := avfs.UnknownGroupIdError(g.Gid())

			if err != wantGroupIdErr {
				t.Errorf("LookupGroupId %s : want error to be %v, got %v", groupName, wantGroupIdErr, err)
			}

			err = idm.DelGroup(groupName)
			if err != wantGroupErr {
				t.Errorf("DelGroup %s : want error to be %v, got %v", groupName, wantGroupErr, err)
			}
		}
	})
}

// TestIdmUser tests user related functions.
func (ts *Suite) TestIdmUser(t *testing.T) {
	idm := ts.idm
	suffix := fmt.Sprintf("UserAddDel%x", rand.Uint32())

	if !idm.HasFeature(avfs.FeatIdentityMgr) || idm.HasFeature(avfs.FeatReadOnlyIdm) {
		groupName := "InvalidGroup" + suffix
		userName := "InvalidUser" + suffix

		_, err := idm.AddUser(userName, groupName)
		if err != avfs.ErrPermDenied {
			t.Errorf("AddUser : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = idm.DelUser(userName)
		if err != avfs.ErrPermDenied {
			t.Errorf("DelUser : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	ts.CreateGroups(t, suffix)

	prevUid := 0
	uis := UserInfos()

	t.Run("AddUserInvalid", func(t *testing.T) {
		_, err := idm.AddUser("", avfs.DefaultName)
		if err != ErrInvalidName {
			t.Errorf("AddUser : want error to be %v, got %v", ErrInvalidName, err)
		}

		_, err = idm.AddUser(avfs.DefaultName, "")
		if err != ErrInvalidName {
			t.Errorf("AddUser : want error to be %v, got %v", ErrInvalidName, err)
		}
	})

	t.Run("AddUser", func(t *testing.T) {
		for _, ui := range uis {
			groupName := ui.GroupName + suffix

			g, err := idm.LookupGroup(groupName)
			RequireNoError(t, err, "LookupGroup %s", groupName)

			userName := ui.Name + suffix
			wantUserErr := avfs.UnknownUserError(userName)

			_, err = idm.LookupUser(userName)
			if err != wantUserErr {
				t.Errorf("LookupUser %s : want error to be %v, got %v", userName, wantUserErr, err)
			}

			u, err := idm.AddUser(userName, groupName)
			RequireNoError(t, err, "AddUser %s", userName)

			if u == nil {
				t.Errorf("AddUser %s : want user to be not nil, got nil", userName)

				continue
			}

			if u.Name() != userName {
				t.Errorf("AddUser %s : want Name to be %s, got %s", userName, userName, u.Name())
			}

			if u.Uid() <= prevUid {
				t.Errorf("AddUser %s : want uid to be > %d, got %d", userName, prevUid, u.Uid())
			} else {
				prevUid = u.Uid()
			}

			if u.Gid() != g.Gid() {
				t.Errorf("AddUser %s : want gid to be %d, got %d", userName, g.Gid(), u.Gid())
			}

			if u.IsAdmin() {
				t.Errorf("IsAdmin %s : want IsAdmin to be false, got true", userName)
			}

			_, err = idm.LookupUser(userName)
			RequireNoError(t, err, "LookupUser %s", userName)

			lu, err := idm.LookupUserId(u.Uid())
			RequireNoError(t, err, "LookupUserId %s", userName)

			lpg := lu.PrimaryGroup()
			if lpg != groupName {
				t.Errorf("PrimaryGroup %s : want group to be '%s', got '%s'", userName, groupName, lpg)
			}

			lgid := lu.PrimaryGroupId()
			if lgid != g.Gid() {
				t.Errorf("PrimaryGroupId %s : want gid to be %d, got %d", userName, g.Gid(), lgid)
			}
		}
	})

	t.Run("UserAddDelNotFound", func(t *testing.T) {
		for _, ui := range uis {
			groupName := ui.GroupName + suffix
			userName := ui.Name + suffix

			_, err := idm.AddUser(userName, groupName)
			if err != avfs.AlreadyExistsUserError(userName) {
				t.Errorf("AddUser %s : want error to be %v, got %v", userName,
					avfs.AlreadyExistsUserError(userName), err)
			}

			groupNameNotFound := ui.GroupName + "NotFound"

			_, err = idm.AddUser(userName, groupNameNotFound)
			if err != avfs.UnknownGroupError(groupNameNotFound) {
				t.Errorf("AddUser %s : want error to be %v, got %v", userName,
					avfs.UnknownGroupError(groupNameNotFound), err)
			}

			userNameNotFound := ui.Name + "NotFound"

			err = idm.DelUser(userNameNotFound)
			if err != avfs.UnknownUserError(userNameNotFound) {
				t.Errorf("DelUser %s : want error to be %v, got %v", userName,
					avfs.UnknownUserError(userNameNotFound), err)
			}
		}
	})

	t.Run("DelUserInvalid", func(t *testing.T) {
		err := idm.DelUser("")
		if err != ErrInvalidName {
			t.Errorf("DelUser '' : want error to be %v, got %v", ErrInvalidName, err)
		}
	})

	t.Run("DelUser", func(t *testing.T) {
		for _, ui := range uis {
			userName := ui.Name + suffix

			u, err := idm.LookupUser(userName)
			RequireNoError(t, err, "LookupUser %s", userName)

			err = idm.DelUser(userName)
			RequireNoError(t, err, "DelUser %s", userName)

			_, err = idm.LookupUser(u.Name())
			wantUserErr := avfs.UnknownUserError(userName)

			if err != wantUserErr {
				t.Errorf("LookupUser %s : want error to be %v, got %v", userName, wantUserErr, err)
			}

			_, err = idm.LookupUserId(u.Uid())
			wantUserIdErr := avfs.UnknownUserIdError(u.Uid())

			if err != wantUserIdErr {
				t.Errorf("LookupUserId %s : want error to be %v, got %v", userName, wantUserIdErr, err)
			}
		}
	})
}

// TestIdmLookup tests Lookup* functions.
func (ts *Suite) TestIdmLookup(t *testing.T) {
	idm := ts.idm
	suffix := fmt.Sprintf("Lookup%x", rand.Uint32())

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		groupName := "InvalidGroup" + suffix

		_, err := idm.LookupGroup(groupName)
		if err != avfs.ErrPermDenied {
			t.Errorf("LookupGroup : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		_, err = idm.LookupGroupId(0)
		if err != avfs.ErrPermDenied {
			t.Errorf("LookupGroupId : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		_, err = idm.LookupUser("")
		if err != avfs.ErrPermDenied {
			t.Errorf("LookupUser : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		_, err = idm.LookupUserId(0)
		if err != avfs.ErrPermDenied {
			t.Errorf("LookupUserId : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	if idm.HasFeature(avfs.FeatReadOnlyIdm) {
		groupName := "InvalidGroup" + suffix
		_, err := idm.LookupGroup(groupName)

		wantGroupErr := avfs.UnknownGroupError(groupName)
		if err != wantGroupErr {
			t.Errorf("LookupGroup %s : want error to be %v, got %v", groupName, wantGroupErr, err)
		}
	}

	groups := ts.CreateGroups(t, suffix)
	users := ts.CreateUsers(t, suffix)

	t.Run("LookupGroupInvalid", func(t *testing.T) {
		_, err := idm.LookupGroup("")
		if err != ErrInvalidName {
			t.Errorf("LookupGroup '' : want error to be %v, got %v", ErrInvalidName, err)
		}
	})

	t.Run("LookupGroup", func(t *testing.T) {
		for _, wantGroup := range groups {
			groupName := wantGroup.Name()
			wantErr := avfs.UnknownGroupError(groupName)

			g, err := idm.LookupGroup(groupName)
			if idm.HasFeature(avfs.FeatReadOnlyIdm) {
				if err != wantErr {
					t.Errorf("LookupGroup %s : want error to be %v, got %v", groupName, wantErr, err)
				}

				continue
			}

			if !AssertNoError(t, err, "LookupGroup %s", groupName) {
				continue
			}

			if g.Name() != groupName {
				t.Errorf("LookupGroup %s : want name to be %s, got %s", groupName, groupName, g.Name())
			}

			if g.Gid() != wantGroup.Gid() {
				t.Errorf("LookupGroup %s : want gid to be %d, got %d", groupName, wantGroup.Gid(), g.Gid())
			}
		}
	})

	t.Run("LookupUserInvalid", func(t *testing.T) {
		_, err := idm.LookupUser("")
		if err != ErrInvalidName {
			t.Errorf("LookupUser '' : want error to be %v, got %v", ErrInvalidName, err)
		}
	})

	t.Run("LookupUser", func(t *testing.T) {
		for _, wantUser := range users {
			userName := wantUser.Name()
			wantErr := avfs.UnknownUserError(userName)

			u, err := idm.LookupUser(userName)
			if idm.HasFeature(avfs.FeatReadOnlyIdm) {
				if err != wantErr {
					t.Errorf("LookupUser %s : want error to be %v, got %v", userName, wantErr, err)
				}

				continue
			}

			if !AssertNoError(t, err, "LookupUser %s", userName) {
				continue
			}

			if u.Name() != userName {
				t.Errorf("LookupUser %s : want name to be %s, got %s", userName, userName, u.Name())
			}

			if u.Uid() != wantUser.Uid() {
				t.Errorf("LookupUser %s : want uid to be %d, got %d", userName, wantUser.Uid(), u.Uid())
			}

			if u.Gid() != wantUser.Gid() {
				t.Errorf("LookupUser %s : want gid to be %d, got %d", userName, wantUser.Gid(), u.Gid())
			}

			if (u.Uid() != 0 && u.Gid() != 0) && u.IsAdmin() {
				t.Errorf("LookupUser %s : want IsAdmin to be false, got true", userName)
			}
		}
	})
}

func (ts *Suite) TestIdmUserGroups(t *testing.T) {
	idm := ts.idm
	suffix := fmt.Sprintf("UserGroups%x", rand.Uint32())

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		err := idm.AddUserToGroup("", "")
		if err != avfs.ErrPermDenied {
			t.Errorf("AddUserToGroup : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = idm.DelUserFromGroup("", "")
		if err != avfs.ErrPermDenied {
			t.Errorf("DelUserFromGroup : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = idm.SetUserPrimaryGroup("", "")
		if err != avfs.ErrPermDenied {
			t.Errorf("SetUserPrimaryGroup : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	_ = ts.CreateGroups(t, suffix)
	users := ts.CreateUsers(t, suffix)

	t.Run("UserGroupsInvalid", func(t *testing.T) {
		err := idm.AddUserToGroup(UsrTest, "")
		if err != ErrInvalidName {
			t.Errorf("AddUserToGroup %s, '' : want error to be %v, got %v", UsrTest, ErrInvalidName, err)
		}

		err = idm.AddUserToGroup("", grpTest)
		if err != ErrInvalidName {
			t.Errorf("AddUserToGroup '', %s : want error to be %v, got %v", grpTest, ErrInvalidName, err)
		}

		err = idm.DelUserFromGroup(UsrTest, "")
		if err != ErrInvalidName {
			t.Errorf("DelUserFromGroup %s, '' : want error to be %v, got %v", UsrTest, ErrInvalidName, err)
		}

		err = idm.DelUserFromGroup("", grpTest)
		if err != ErrInvalidName {
			t.Errorf("DelUserFromGroup '', %s : want error to be %v, got %v", grpTest, ErrInvalidName, err)
		}

		err = idm.SetUserPrimaryGroup(UsrTest, "")
		if err != ErrInvalidName {
			t.Errorf("SetUserPrimaryGroup %s, '' : want error to be %v, got %v", UsrTest, ErrInvalidName, err)
		}

		err = idm.SetUserPrimaryGroup("", grpTest)
		if err != ErrInvalidName {
			t.Errorf("SetUserPrimaryGroup '', %s : want error to be %v, got %v", grpTest, ErrInvalidName, err)
		}
	})

	t.Run("UserGroupsNotFound", func(t *testing.T) {
		for _, u := range users {
			userNameNotFound := u.Name() + "NotFound"
			groupNameNotFound := u.PrimaryGroup() + "NotFound"

			err := idm.AddUserToGroup(u.Name(), groupNameNotFound)
			if err != avfs.UnknownGroupError(groupNameNotFound) {
				t.Errorf("AddUserToGroup %s : want error to be %v, got %v", u.Name(),
					avfs.UnknownGroupError(groupNameNotFound), err)
			}

			err = idm.AddUserToGroup(userNameNotFound, u.PrimaryGroup())
			if err != avfs.UnknownUserError(userNameNotFound) {
				t.Errorf("AddUserToGroup %s : want error to be %v, got %v", u.PrimaryGroup(),
					avfs.UnknownGroupError(groupNameNotFound), err)
			}

			err = idm.DelUserFromGroup(u.Name(), groupNameNotFound)
			if err != avfs.UnknownGroupError(groupNameNotFound) {
				t.Errorf("DelUserFromGroup %s : want error to be %v, got %v", u.Name(),
					avfs.UnknownGroupError(groupNameNotFound), err)
			}

			err = idm.DelUserFromGroup(userNameNotFound, u.PrimaryGroup())
			if err != avfs.UnknownUserError(userNameNotFound) {
				t.Errorf("DelUserFromGroup %s : want error to be %v, got %v", u.PrimaryGroup(),
					avfs.UnknownGroupError(groupNameNotFound), err)
			}

			err = idm.SetUserPrimaryGroup(u.Name(), groupNameNotFound)
			if err != avfs.UnknownGroupError(groupNameNotFound) {
				t.Errorf("SetUserPrimaryGroup %s : want error to be %v, got %v", u.Name(),
					avfs.UnknownGroupError(groupNameNotFound), err)
			}

			err = idm.SetUserPrimaryGroup(userNameNotFound, u.PrimaryGroup())
			if err != avfs.UnknownUserError(userNameNotFound) {
				t.Errorf("SetUserPrimaryGroup %s : want error to be %v, got %v", u.PrimaryGroup(),
					avfs.UnknownGroupError(groupNameNotFound), err)
			}
		}
	})

	t.Run("UserGroupsAddDel", func(t *testing.T) {
		newGroup := "newGroup" + suffix

		g, err := idm.AddGroup(newGroup)
		if err != nil {
			t.Errorf("AddGroup %s : want error to be nil, got %v", newGroup, err)
		}

		for _, u := range users {
			err = idm.AddUserToGroup(u.Name(), newGroup)
			if err != nil {
				t.Errorf("AddUserToGroup %s %s : want error to be nil, got %v", u.Name(), newGroup, err)
			}

			if !u.IsInGroupId(g.Gid()) {
				t.Errorf("AddUserToGroup %s %s : want user to be in group, got false", u.Name(), newGroup)
			}

			err = idm.DelUserFromGroup(u.Name(), newGroup)
			if err != nil {
				t.Errorf("DelUserFromGroup %s %s : want error to be nil, got %v", u.Name(), newGroup, err)
			}

			if u.IsInGroupId(g.Gid()) {
				t.Errorf("DelUserFromGroup %s %s : want user to be out of group, got true", u.Name(), newGroup)
			}
		}
	})
}

const (
	grpTest  = "grpTest"  // grpTest is the default group of the default test user UsrTest.
	grpOther = "grpOther" // grpOther is the group to test users who are not members of grpTest.
	grpEmpty = "grpEmpty" // grpEmpty is a group without users.
)

// GroupInfo contains information to create a test group.
type GroupInfo struct {
	Name string
}

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
func (ts *Suite) CreateGroups(tb testing.TB, suffix string) (groups []avfs.GroupReader) {
	idm := ts.idm
	if !idm.HasFeature(avfs.FeatIdentityMgr) || idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return nil
	}

	for _, group := range GroupInfos() {
		groupName := group.Name + suffix

		g, err := idm.AddGroup(groupName)
		if err != avfs.AlreadyExistsGroupError(groupName) {
			RequireNoError(tb, err, "AddGroup %s", groupName)
		}

		groups = append(groups, g)
	}

	return groups
}

const (
	UsrTest = "UsrTest" // UsrTest is used to test user access rights.
	UsrGrp  = "UsrGrp"  // UsrGrp is a member of the group GrpTest used to test default group access rights.
	UsrOth  = "UsrOth"  // UsrOth is a member of the group GrpOth used to test non-members access rights.
)

// UserInfo contains information to create a test user.
type UserInfo struct {
	Name      string
	GroupName string
}

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
func (ts *Suite) CreateUsers(tb testing.TB, suffix string) (users []avfs.UserReader) {
	idm := ts.idm
	if !idm.HasFeature(avfs.FeatIdentityMgr) || idm.HasFeature(avfs.FeatReadOnlyIdm) {
		return nil
	}

	for _, ui := range UserInfos() {
		userName := ui.Name + suffix
		groupName := ui.GroupName + suffix

		u, err := idm.AddUser(userName, groupName)
		if err != nil {
			switch e := err.(type) {
			case *fs.PathError:
				if e.Op != "mkdir" || e.Err != avfs.ErrFileExists {
					tb.Fatalf("AddUser %s : want Mkdir error, got %v", userName, err)
				}
			default:
				if err != avfs.AlreadyExistsUserError(userName) {
					tb.Fatalf("AddUser %s : want error to be nil, got %v", userName, err)
				}
			}

			if u == nil {
				u, err = idm.LookupUser(userName)
				RequireNoError(tb, err, "LookupUser %s", userName)
			}
		}

		users = append(users, u)
	}

	return users
}
