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
	"math"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfsutils"
)

// TestCurrentUser tests CurrentUser function.
func (sIdm *SuiteIdm) TestCurrentUser(t *testing.T) {
	idm := sIdm.idm

	uc, ok := idm.(avfs.UserConnecter)
	if !ok {
		return
	}

	u := uc.CurrentUser()

	name := u.Name()
	if name == "" {
		t.Errorf("Name : want name to be not empty, got empty")
	}

	uid := u.Uid()
	if uid < 0 {
		t.Errorf("Uid : want uid to be >= 0, got %d", uid)
	}

	gid := u.Gid()
	if uid < 0 {
		t.Errorf("Uid : want gid to be >= 0, got %d", gid)
	}
}

// TestGroupAddDel tests GroupAdd and GroupDel functions.
func (sIdm *SuiteIdm) TestGroupAddDel(t *testing.T) {
	idm := sIdm.idm
	suffix := "GroupAddDel" + sIdm.Type()

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		_, err := idm.GroupAdd("")
		if err != avfs.ErrPermDenied {
			t.Errorf("GroupAdd : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = idm.GroupDel("")
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
			if err != nil {
				t.Errorf("GroupAdd %s : want error to be nil, got %v", groupName, err)

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
			if err != nil {
				t.Errorf("LookupGroup %s : want error to be nil, got %v", groupName, err)
			}

			_, err = idm.LookupGroupId(g.Gid())
			if err != nil {
				t.Errorf("LookupGroupId %s : want error to be nil, got %v", groupName, err)
			}
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
			if err != nil {
				t.Fatalf("LookupGroup %s : want error to be nil, got %v", groupName, err)
			}

			err = idm.GroupDel(groupName)
			if err != nil {
				t.Errorf("GroupDel %s : want error to be nil, got %v", groupName, err)
			}

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

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		_, err := idm.UserAdd("", "")
		if err != avfs.ErrPermDenied {
			t.Errorf("UserAdd : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}

		err = idm.UserDel("")
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
			if err != nil {
				t.Fatalf("LookupGroup %s : want error to be nil, got %v", groupName, err)
			}

			userName := ui.Name + suffix
			wantUserErr := avfs.UnknownUserError(userName)

			_, err = idm.LookupUser(userName)
			if err != wantUserErr {
				t.Errorf("LookupUser %s : want error to be %v, got %v", userName, wantUserErr, err)
			}

			u, err := idm.UserAdd(userName, groupName)
			if err != nil {
				t.Errorf("UserAdd %s : want error to be nil, got %v", userName, err)
			}

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

			if u.IsRoot() {
				t.Errorf("IsRoot %s : want IsRoot to be false, got true", userName)
			}

			_, err = idm.LookupUser(userName)
			if err != nil {
				t.Errorf("LookupUser %s : want error to be nil, got %v", userName, err)
			}

			_, err = idm.LookupUserId(u.Uid())
			if err != nil {
				t.Errorf("LookupUserId %s : want error to be nil, got %v", userName, err)
			}

			checkHomeDir(t, idm, u)
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
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", userName, err)
			}

			err = idm.UserDel(userName)
			if err != nil {
				t.Errorf("UserDel %s : want error to be nil, got %v", userName, err)
			}

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
		_, err := idm.LookupGroup("")
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

	if !sIdm.canTest {
		return
	}

	CreateGroups(t, idm, suffix)
	CreateUsers(t, idm, suffix)

	t.Run("LookupGroup", func(t *testing.T) {
		for _, gi := range GroupInfos() {
			groupName := gi.Name + suffix

			g, err := idm.LookupGroup(groupName)
			if err != nil {
				t.Errorf("LookupGroup %s : want error to be nil, got %v", groupName, err)

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

			u, err := idm.LookupUser(userName)
			if err != nil {
				t.Errorf("LookupUser %s : want error to be nil, got %v", userName, err)

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

			if (u.Uid() != 0 && u.Gid() != 0) && u.IsRoot() {
				t.Errorf("LookupUser %s : want isRoot to be false, got true", userName)
			}
		}
	})
}

// TestUser tests User and CurrentUser functions.
func (sIdm *SuiteIdm) TestUser(t *testing.T) {
	idm := sIdm.idm
	suffix := "User" + sIdm.Type()

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		if uc, ok := idm.(avfs.UserConnecter); ok {
			_, err := uc.User("")
			if err != avfs.ErrPermDenied {
				t.Errorf("User : want error to be %v, got %v", avfs.ErrPermDenied, err)
			}
		}

		return
	}

	if !sIdm.canTest || sIdm.uc == nil {
		return
	}

	defer sIdm.uc.User(avfs.UsrRoot) //nolint:errcheck // Ignore errors.

	CreateGroups(t, idm, suffix)
	CreateUsers(t, idm, suffix)

	t.Run("UserNotExists", func(t *testing.T) {
		const userName = "notExistingUser"

		wantErr := avfs.UnknownUserError(userName)

		_, err := idm.LookupUser(userName)
		if err != wantErr {
			t.Fatalf("LookupUser %s : want error to be %v, got %v", userName, wantErr, err)
		}

		_, err = sIdm.uc.User(userName)
		if err != wantErr {
			t.Errorf("User %s : want error to be %v, got %v", userName, wantErr, err)
		}
	})

	t.Run("UserExists", func(t *testing.T) {
		for _, ui := range UserInfos() {
			userName := ui.Name + suffix

			lu, err := idm.LookupUser(userName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", userName, err)
			}

			uid := lu.Uid()
			gid := lu.Gid()

			// loop to test change with the same user
			for i := 0; i < 2; i++ {
				u, err := sIdm.uc.User(userName)
				if err != nil {
					t.Errorf("User %s : want error to be nil, got %v", userName, err)

					continue
				}

				if u.Name() != userName {
					t.Errorf("User %s : want name to be %s, got %s", userName, userName, u.Name())
				}

				if u.Uid() != uid {
					t.Errorf("User %s : want uid to be %d, got %d", userName, uid, u.Uid())
				}

				if u.Gid() != gid {
					t.Errorf("User %s : want gid to be %d, got %d", userName, gid, u.Gid())
				}

				cu := sIdm.uc.CurrentUser()
				if cu.Name() != userName {
					t.Errorf("User %s : want name to be %s, got %s", userName, userName, cu.Name())
				}

				if cu.Uid() != uid {
					t.Errorf("User %s : want uid to be %d, got %d", userName, uid, cu.Uid())
				}

				if cu.Gid() != gid {
					t.Errorf("User %s : want gid to be %d, got %d", userName, gid, cu.Gid())
				}
			}
		}
	})
}

// TestUserDenied tests if non root users are denied write access.
func (sIdm *SuiteIdm) TestUserDenied(t *testing.T) {
	suffix := "Denied" + sIdm.Type()
	idm := sIdm.idm

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		return
	}

	if !sIdm.canTest || sIdm.uc == nil {
		return
	}

	defer sIdm.uc.User(avfs.UsrRoot) //nolint:errcheck // Ignore errors.

	_, err := sIdm.uc.User(UsrTest)
	if err != nil {
		t.Fatalf("User: want error to be nil, got %v", err)
	}

	for _, ui := range UserInfos() {
		// skip UsrTest : can't try to delete a user in an active process.
		if ui.Name == UsrTest {
			continue
		}

		name := ui.Name + suffix

		_, err = idm.UserAdd(name, ui.GroupName)
		if err != avfs.ErrPermDenied {
			t.Errorf("UserAdd %s : want error to be %v, got %v", name, avfs.ErrPermDenied, err)
		}

		err = idm.UserDel(ui.Name)
		if err != avfs.ErrPermDenied {
			t.Errorf("UserDel %s : want error to be %v, got %v", name, avfs.ErrPermDenied, err)
		}
	}

	for _, gi := range GroupInfos() {
		name := gi.Name + suffix

		_, err = idm.GroupAdd(name)
		if err != avfs.ErrPermDenied {
			t.Errorf("GroupAdd %s : want error to be %v, got %v", name, avfs.ErrPermDenied, err)
		}
	}

	err = idm.GroupDel(grpEmpty)
	if err != avfs.ErrPermDenied {
		t.Errorf("GroupDel %s : want error to be %v, got %v", grpEmpty, avfs.ErrPermDenied, err)
	}
}

// TestPermDenied tests if all functions of the identity manager return avfs.ErrPermDenied
// when the identity manager can't be tested.
func (sIdm *SuiteIdm) TestPermDenied(t *testing.T) {
	idm := sIdm.idm

	if !idm.HasFeature(avfs.FeatIdentityMgr) {
		return
	}

	if sIdm.canTest {
		return
	}

	defer sIdm.uc.User(avfs.UsrRoot) //nolint:errcheck // Ignore errors.

	const (
		grpName = "grpDenied"
		grpId   = math.MaxInt32
		usrName = "usrDenied"
		usrId   = math.MaxInt32
	)

	_, err := idm.GroupAdd(grpName)
	if err != avfs.ErrPermDenied {
		t.Errorf("GroupAdd : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	err = idm.GroupDel(grpName)
	if err != avfs.ErrPermDenied {
		t.Errorf("GroupDel : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	_, err = idm.LookupGroup(grpName)
	if idm.HasFeature(avfs.FeatIdentityMgr) {
		if err != avfs.UnknownGroupError(grpName) {
			t.Errorf("LookupGroupName : want error to be %v, got %v", avfs.UnknownGroupError(grpName), err)
		}
	} else {
		if err != avfs.ErrPermDenied {
			t.Errorf("LookupGroupName : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}
	}

	_, err = idm.LookupGroupId(grpId)
	if idm.HasFeature(avfs.FeatIdentityMgr) {
		if err != avfs.UnknownGroupIdError(grpId) {
			t.Errorf("LookupGroupId : want error to be %v, got %v", avfs.UnknownGroupIdError(grpId), err)
		}
	} else {
		if err != avfs.ErrPermDenied {
			t.Errorf("LookupGroupId : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}
	}

	_, err = idm.LookupUser(usrName)
	if idm.HasFeature(avfs.FeatIdentityMgr) {
		if err != avfs.UnknownUserError(usrName) {
			t.Errorf("LookupUser : want error to be %v, got %v", avfs.UnknownUserError(usrName), err)
		}
	} else {
		if err != avfs.ErrPermDenied {
			t.Errorf("LookupUser : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}
	}

	_, err = idm.LookupUserId(usrId)
	if idm.HasFeature(avfs.FeatIdentityMgr) {
		if err != avfs.UnknownUserIdError(usrId) {
			t.Errorf("LookupUserId : want error to be %v, got %v", avfs.UnknownUserIdError(usrId), err)
		}
	} else {
		if err != avfs.ErrPermDenied {
			t.Errorf("LookupUserId : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}
	}

	_, err = idm.UserAdd(usrName, grpName)
	if err != avfs.ErrPermDenied {
		t.Errorf("UserAdd : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	err = idm.UserDel(usrName)
	if err != avfs.ErrPermDenied {
		t.Errorf("UserDel : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	if sIdm.uc == nil {
		return
	}

	_, err = sIdm.uc.User(usrName)
	if err != avfs.ErrPermDenied {
		t.Errorf("User : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}
}

// checkHomeDir tests that the user home directory exists and has the correct permissions.
func checkHomeDir(t *testing.T, idm avfs.IdentityMgr, u avfs.UserReader) {
	vfs, ok := idm.(avfs.VFS)
	if !ok || vfs.OSType() == avfs.OsWindows {
		return
	}

	homeDir := vfs.Join(avfs.HomeDir, u.Name())

	fst, err := vfs.Stat(homeDir)
	if err != nil {
		t.Errorf("Stat %s : want error to be nil, got %v", homeDir, err)
	}

	wantMode := fs.ModeDir | avfs.HomeDirPerm&^vfs.GetUMask()
	if fst.Mode() != wantMode {
		t.Errorf("Stat %s : want mode to be %o, got %o", homeDir, wantMode, fst.Mode())
	}

	sst := vfsutils.ToSysStat(fst.Sys())

	uid, gid := sst.Uid(), sst.Gid()
	if uid != u.Uid() || gid != u.Gid() {
		t.Errorf("Stat %s : want uid=%d, gid=%d, got uid=%d, gid=%d", homeDir, u.Uid(), u.Gid(), uid, gid)
	}
}
