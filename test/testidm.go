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
	"math"
	"testing"

	"github.com/avfs/avfs"
)

// TestAdminGroupUser tests AdminGroup and AdminUser.
func (sIdm *SuiteIdm) TestAdminGroupUser(t *testing.T) {
	idm := sIdm.idm

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

	t.Run("Admin", func(t *testing.T) {
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

// TestGroupAddDel tests GroupAdd and GroupDel functions.
func (sIdm *SuiteIdm) TestGroupAddDel(t *testing.T) {
	idm := sIdm.idm
	suffix := "GroupAddDel" + sIdm.Type()

	if !idm.HasFeature(avfs.FeatIdentityMgr) || idm.HasFeature(avfs.FeatReadOnlyIdm) {
		groupName := "AGroup" + suffix

		_, err := idm.GroupAdd(groupName)
		if err != avfs.ErrPermDenied {
			t.Errorf("GroupAdd : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = idm.GroupDel(groupName)
		if err != avfs.ErrPermDenied {
			t.Errorf("GroupDel : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	if !sIdm.canTest {
		return
	}

	gis := GroupInfos()
	prevGid := 0

	t.Run("GroupAdd", func(t *testing.T) {
		for _, gi := range gis {
			groupName := gi.Name + suffix
			wantGroupErr := avfs.UnknownGroupError(groupName)

			_, err := idm.LookupGroup(groupName)
			if err != wantGroupErr {
				t.Errorf("LookupGroupName %s : want error to be %v, got %v", groupName, wantGroupErr, err)
			}

			g, err := idm.GroupAdd(groupName)
			if !AssertNoError(t, err, "GroupAdd %s", groupName) {
				continue
			}

			if g.Name() != groupName {
				t.Errorf("GroupAdd %s : want Name to be %s, got %s", groupName, groupName, g.Name())
			}

			if g.Gid() <= prevGid {
				t.Errorf("GroupAdd %s : want gid to be > %d, got %d", groupName, prevGid, g.Gid())
			} else {
				prevGid = g.Gid()
			}

			_, err = idm.LookupGroup(groupName)
			RequireNoError(t, err, "LookupGroup %s", groupName)

			_, err = idm.LookupGroupId(g.Gid())
			RequireNoError(t, err, "LookupGroupId %s", groupName)
		}
	})

	t.Run("GroupAddExists", func(t *testing.T) {
		for _, gi := range gis {
			groupName := gi.Name + suffix

			_, err := idm.GroupAdd(groupName)
			if err != avfs.AlreadyExistsGroupError(groupName) {
				t.Errorf("GroupAdd %s : want error to be %v, got %v",
					groupName, avfs.AlreadyExistsGroupError(groupName), err)
			}
		}
	})

	t.Run("GroupDel", func(t *testing.T) {
		for _, gi := range gis {
			groupName := gi.Name + suffix

			g, err := idm.LookupGroup(groupName)
			RequireNoError(t, err, "LookupGroup %s", groupName)

			err = idm.GroupDel(groupName)
			RequireNoError(t, err, "GroupDel %s", groupName)

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

			err = idm.GroupDel(groupName)
			if err != wantGroupErr {
				t.Errorf("GroupDel %s : want error to be %v, got %v", groupName, wantGroupErr, err)
			}
		}
	})
}

// TestUserAddDel tests UserAdd and UserDel functions.
func (sIdm *SuiteIdm) TestUserAddDel(t *testing.T) {
	idm := sIdm.idm
	suffix := "UserAddDel" + sIdm.Type()

	if !idm.HasFeature(avfs.FeatIdentityMgr) || idm.HasFeature(avfs.FeatReadOnlyIdm) {
		groupName := "InvalidGroup" + suffix
		userName := "InvalidUser" + suffix

		_, err := idm.UserAdd(userName, groupName)
		if err != avfs.ErrPermDenied {
			t.Errorf("UserAdd : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = idm.UserDel(userName)
		if err != avfs.ErrPermDenied {
			t.Errorf("UserDel : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		return
	}

	if !sIdm.canTest {
		return
	}

	_ = CreateGroups(t, idm, suffix)
	uis := UserInfos()

	prevUid := 0

	t.Run("UserAdd", func(t *testing.T) {
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

			u, err := idm.UserAdd(userName, groupName)
			RequireNoError(t, err, "UserAdd %s", userName)

			if u == nil {
				t.Errorf("UserAdd %s : want user to be not nil, got nil", userName)

				continue
			}

			if u.Name() != userName {
				t.Errorf("UserAdd %s : want Name to be %s, got %s", userName, userName, u.Name())
			}

			if u.Uid() <= prevUid {
				t.Errorf("UserAdd %s : want uid to be > %d, got %d", userName, prevUid, u.Uid())
			} else {
				prevUid = u.Uid()
			}

			if u.Gid() != g.Gid() {
				t.Errorf("UserAdd %s : want gid to be %d, got %d", userName, g.Gid(), u.Gid())
			}

			if u.IsAdmin() {
				t.Errorf("IsAdmin %s : want IsAdmin to be false, got true", userName)
			}

			_, err = idm.LookupUser(userName)
			RequireNoError(t, err, "LookupUser %s", userName)

			_, err = idm.LookupUserId(u.Uid())
			RequireNoError(t, err, "LookupUserId %s", userName)
		}
	})

	t.Run("UserAddDelErrors", func(t *testing.T) {
		for _, ui := range uis {
			groupName := ui.GroupName + suffix
			userName := ui.Name + suffix

			_, err := idm.UserAdd(userName, groupName)
			if err != avfs.AlreadyExistsUserError(userName) {
				t.Errorf("UserAdd %s : want error to be %v, got %v", userName,
					avfs.AlreadyExistsUserError(userName), err)
			}

			groupNameNotFound := ui.GroupName + "NotFound"

			_, err = idm.UserAdd(userName, groupNameNotFound)
			if err != avfs.UnknownGroupError(groupNameNotFound) {
				t.Errorf("UserAdd %s : want error to be %v, got %v", userName,
					avfs.UnknownGroupError(groupNameNotFound), err)
			}

			userNameNotFound := ui.Name + "NotFound"

			err = idm.UserDel(userNameNotFound)
			if err != avfs.UnknownUserError(userNameNotFound) {
				t.Errorf("UserDel %s : want error to be %v, got %v", userName,
					avfs.UnknownUserError(userNameNotFound), err)
			}
		}
	})

	t.Run("UserDel", func(t *testing.T) {
		for _, ui := range uis {
			userName := ui.Name + suffix

			u, err := idm.LookupUser(userName)
			RequireNoError(t, err, "LookupUser %s", userName)

			err = idm.UserDel(userName)
			RequireNoError(t, err, "UserDel %s", userName)

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

// TestLookup tests Lookup* functions.
func (sIdm *SuiteIdm) TestLookup(t *testing.T) {
	idm := sIdm.idm
	suffix := "Lookup" + sIdm.Type()

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

	if !sIdm.canTest {
		return
	}

	CreateGroups(t, idm, suffix)
	CreateUsers(t, idm, suffix)

	t.Run("LookupGroup", func(t *testing.T) {
		for _, gi := range GroupInfos() {
			groupName := gi.Name + suffix
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

			if g.Gid() <= 0 {
				t.Errorf("LookupGroup %s : want gid to be > 0, got %d", groupName, g.Gid())
			}
		}
	})

	t.Run("LookupUser", func(t *testing.T) {
		for _, ui := range UserInfos() {
			userName := ui.Name + suffix
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

			if u.Uid() <= 0 {
				t.Errorf("LookupUser %s : want uid to be > 0, got %d", userName, u.Uid())
			}

			if u.Gid() <= 0 {
				t.Errorf("LookupUser %s : want gid to be > 0, got %d", userName, u.Gid())
			}

			if (u.Uid() != 0 && u.Gid() != 0) && u.IsAdmin() {
				t.Errorf("LookupUser %s : want IsAdmin to be false, got true", userName)
			}
		}
	})
}
