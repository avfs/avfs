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
	"runtime"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/fsutil"
)

// SuiteAll run all identity manager tests.
func (ci *ConfigIdm) SuiteAll() {
	if !ci.cantTest {
		ci.SuitePermDenied()
		return
	}

	ci.SuiteGroupAddDel()
	ci.SuiteUserAddDel()
	ci.SuiteLookup()
	ci.SuiteUser()
	ci.SuiteUserDenied()
}

// SuiteGroupAddDel tests GroupAdd and GroupDel functions.
func (ci *ConfigIdm) SuiteGroupAddDel() {
	suffix := "GroupAddDel" + ci.Type()
	t, idm := ci.t, ci.idm

	uc, ok := idm.(avfs.UserConnecter)
	if ok && !uc.CurrentUser().IsRoot() {
		return
	}

	groups := GetGroups()
	prevGid := 0

	t.Run("GroupAdd", func(t *testing.T) {
		for _, group := range groups {
			groupName := group.Name + suffix
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
		for _, group := range groups {
			groupName := group.Name + suffix

			_, err := idm.GroupAdd(groupName)
			if err != avfs.AlreadyExistsGroupError(groupName) {
				t.Errorf("GroupAdd %s : want error to be %v, got %v",
					groupName, avfs.AlreadyExistsGroupError(groupName), err)
			}
		}
	})

	t.Run("GroupDel", func(t *testing.T) {
		for _, group := range groups {
			groupName := group.Name + suffix

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

// SuiteUserAddDel tests UserAdd and UserDel functions.
func (ci *ConfigIdm) SuiteUserAddDel() {
	suffix := "UserAddDel" + ci.Type()
	t, idm := ci.t, ci.idm

	uc, ok := idm.(avfs.UserConnecter)
	if ok && !uc.CurrentUser().IsRoot() {
		return
	}

	_ = CreateGroups(t, idm, suffix)
	users := GetUsers()

	prevUid := 0

	t.Run("UserAdd", func(t *testing.T) {
		for _, user := range users {
			groupName := user.GroupName + suffix

			g, err := idm.LookupGroup(groupName)
			if err != nil {
				t.Fatalf("LookupGroup %s : want error to be nil, got %v", groupName, err)
			}

			userName := user.Name + suffix
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
		for _, user := range users {
			groupName := user.GroupName + suffix
			userName := user.Name + suffix

			_, err := idm.UserAdd(userName, groupName)
			if err != avfs.AlreadyExistsUserError(userName) {
				t.Errorf("UserAdd %s : want error to be %v, got %v", userName,
					avfs.AlreadyExistsUserError(userName), err)
			}

			groupNameNotFound := user.GroupName + "NotFound"

			_, err = idm.UserAdd(userName, groupNameNotFound)
			if err != avfs.UnknownGroupError(groupNameNotFound) {
				t.Errorf("UserAdd %s : want error to be %v, got %v", userName,
					avfs.UnknownGroupError(groupNameNotFound), err)
			}

			userNameNotFound := user.Name + "NotFound"

			err = idm.UserDel(userNameNotFound)
			if err != avfs.UnknownUserError(userNameNotFound) {
				t.Errorf("UserDel %s : want error to be %v, got %v", userName,
					avfs.UnknownUserError(userNameNotFound), err)
			}
		}
	})

	t.Run("UserDel", func(t *testing.T) {
		for _, user := range users {
			userName := user.Name + suffix

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

// SuiteLookup tests Lookup* functions.
func (ci *ConfigIdm) SuiteLookup() {
	suffix := "Lookup" + ci.Type()
	t, idm := ci.t, ci.idm

	uc, ok := idm.(avfs.UserConnecter)
	if ok && !uc.CurrentUser().IsRoot() {
		return
	}

	groups := CreateGroups(t, idm, suffix)
	users := CreateUsers(t, idm, suffix)

	t.Run("LookupGroup", func(t *testing.T) {
		for _, group := range groups {
			groupName := group.Name + suffix

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
		for _, user := range users {
			userName := user.Name + suffix

			u, err := idm.LookupUser(userName)
			if err != nil {
				t.Errorf("LookupUser %s : want error to be nil, got %v", userName, err)
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

// SuiteUser tests User and CurrentUser functions.
func (ci *ConfigIdm) SuiteUser() {
	suffix := "User" + ci.Type()
	t, idm := ci.t, ci.idm

	uc, ok := idm.(avfs.UserConnecter)
	if !ok || !uc.CurrentUser().IsRoot() {
		return
	}

	_ = CreateGroups(t, idm, suffix)
	users := CreateUsers(t, idm, suffix)

	t.Run("UserNotExists", func(t *testing.T) {
		const userName = "notExistingUser"

		wantErr := avfs.UnknownUserError(userName)

		_, err := idm.LookupUser(userName)
		if err != wantErr {
			t.Fatalf("LookupUser %s : want error to be %v, got %v", userName, wantErr, err)
		}

		_, err = uc.User(userName)
		if err != wantErr {
			t.Errorf("User %s : want error to be %v, got %v", userName, wantErr, err)
		}
	})

	t.Run("UserExists", func(t *testing.T) {
		for _, user := range users {
			userName := user.Name + suffix

			lu, err := idm.LookupUser(userName)
			if err != nil {
				t.Fatalf("LookupUser %s : want error to be nil, got %v", userName, err)
			}

			uid := lu.Uid()
			gid := lu.Gid()

			// loop to test change with the same user
			for i := 0; i < 2; i++ {
				u, err := uc.User(userName)
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

				cu := uc.CurrentUser()
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

// SuiteUserDenied tests if non root users are denied write access.
func (ci *ConfigIdm) SuiteUserDenied() {
	suffix := "Denied" + ci.Type()
	t, idm := ci.t, ci.idm

	uc, ok := idm.(avfs.UserConnecter)
	if !ok {
		t.Skipf("%s does not implement avfs.UserConnecter : skipping", ci.Type())
	}

	_, err := uc.User(UsrTest)
	if err != nil {
		t.Fatalf("User: want error to be nil, got %v", err)
	}

	for _, u := range GetUsers() {
		// skip UsrTest : can't try to delete a user in an active process.
		if u.Name == UsrTest {
			continue
		}

		name := u.Name + suffix

		_, err = idm.UserAdd(name, u.GroupName)
		if err != avfs.ErrPermDenied {
			t.Errorf("UserAdd %s : want error to be %v, got %v", name, avfs.ErrPermDenied, err)
		}

		err = idm.UserDel(u.Name)
		if err != avfs.ErrPermDenied {
			t.Errorf("UserDel %s : want error to be %v, got %v", name, avfs.ErrPermDenied, err)
		}
	}

	for _, g := range GetGroups() {
		name := g.Name + suffix

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

// SuitePermDenied tests if all functions of the identity manager return avfs.ErrPermDenied.
func (ci *ConfigIdm) SuitePermDenied() {
	t, idm := ci.t, ci.idm

	const name = ""

	uc, ok := idm.(avfs.UserConnecter)
	if ok {
		u := uc.CurrentUser()
		if u == nil {
			t.Fatal("CurrentUser : want user to be not nil, got nil")
		}

		_, err := uc.User(name)
		if err != avfs.ErrPermDenied {
			t.Errorf("User : want error to be %v, got %v", avfs.ErrPermDenied, err)
		}
	}

	_, err := idm.GroupAdd(name)
	if err != avfs.ErrPermDenied {
		t.Errorf("GroupAdd : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	err = idm.GroupDel(name)
	if err != avfs.ErrPermDenied {
		t.Errorf("GroupDel : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	_, err = idm.LookupGroup(name)
	if err != avfs.ErrPermDenied {
		t.Errorf("LookupGroupName : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	_, err = idm.LookupGroupId(0)
	if err != avfs.ErrPermDenied {
		t.Errorf("LookupGroupId : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	_, err = idm.LookupUser(name)
	if err != avfs.ErrPermDenied {
		t.Errorf("LookupUser : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	_, err = idm.LookupUserId(0)
	if err != avfs.ErrPermDenied {
		t.Errorf("LookupUserId : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	_, err = idm.UserAdd(name, name)
	if err != avfs.ErrPermDenied {
		t.Errorf("UserAdd : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}

	err = idm.UserDel(name)
	if err != avfs.ErrPermDenied {
		t.Errorf("UserDel : want error to be %v, got %v", avfs.ErrPermDenied, err)
	}
}

// checkHomeDir tests that the user home directory exists and has the correct permissions.
func checkHomeDir(t *testing.T, idm avfs.IdentityMgr, u avfs.UserReader) {
	fs, ok := idm.(avfs.Fs)
	if !ok || (fs.Type() == "OsFs" && runtime.GOOS == "Windows") {
		return
	}

	homeDir := fs.Join(avfs.HomeDir, u.Name())

	info, err := fs.Stat(homeDir)
	if err != nil {
		t.Errorf("Stat %s : want error to be nil, got %v", homeDir, err)
	}

	wantMode := os.ModeDir | avfs.HomeDirPerm&^fs.GetUMask()
	if info.Mode() != wantMode {
		t.Errorf("Stat %s : want mode to be %o, got %o", homeDir, wantMode, info.Mode())
	}

	sys := info.Sys()
	statT := fsutil.AsStatT(sys)

	uid, gid := int(statT.Uid), int(statT.Gid)
	if uid != u.Uid() || gid != u.Gid() {
		t.Errorf("Stat %s : want uid=%d, gid=%d, got uid=%d, gid=%d", homeDir, u.Uid(), u.Gid(), uid, gid)
	}
}
